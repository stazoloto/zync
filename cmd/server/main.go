package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/stazoloto/zync/config"
	"github.com/stazoloto/zync/internal/handlers"
	"github.com/stazoloto/zync/internal/infrastructure/sfu"
	postgresRepo "github.com/stazoloto/zync/internal/repository/postgres"
	redisRepo "github.com/stazoloto/zync/internal/repository/redis"
	"github.com/stazoloto/zync/internal/service"
	"github.com/stazoloto/zync/pkg/email"
	"github.com/stazoloto/zync/pkg/jwt"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load error: %s", err)
	}
	defer cfg.Logger.Sync()

	// Проверяем соединение с базой данных
	db, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		log.Fatal("database initialize error")
	}

	if err = db.Ping(); err != nil {
		log.Fatalf("database ping error: %v", err)
	}

	// Проверяем соединение с Redis
	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisURL})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping error: %v", err)
	}

	redisRepos := redisRepo.New(rdb)

	postgresRepos := postgresRepo.New(db)
	roomService := service.NewRoomService(
		postgresRepos.Users,
		postgresRepos.Rooms,
		postgresRepos.Participants,
		redisRepos.Presence,
		redisRepos.PubSub, // chatPublisher
		redisRepos.PubSub, // chatSubscriber
		cfg.Logger,
	)

	jwtManager := jwt.NewManager([]byte(cfg.JWTSecret))
	emailSender := email.NewSMTPSender(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPFrom)
	authService := service.NewAuthService(
		postgresRepos.Users,
		redisRepos.VerifyCode,
		jwtManager,
		emailSender,
		cfg.Logger,
	)
	authHandler := handlers.NewAuthHandler(
		authService,
		cfg.Logger,
	)

	adminService := service.NewAdminService(postgresRepos.Users, postgresRepos.Rooms, redisRepos.Presence, cfg.Logger)
	adminHandler := handlers.NewAdminHandler(adminService, cfg.Logger)

	mediaGateway := sfu.NewSFU(cfg.Logger)
	signalingService := service.NewSignalingService(mediaGateway, roomService, cfg.Logger)

	wsHandler := handlers.NewWebSocketHandler(
		roomService,
		signalingService,
		cfg.Logger,
	)

	authMW  := handlers.AuthMiddleware(jwtManager)
	adminMW := handlers.AdminMiddleware(jwtManager)

	r := gin.Default()

	r.Static("/static", "./web")
	r.StaticFile("/", "./web/index.html")
	r.StaticFile("/login.html", "./web/login.html")
	r.StaticFile("/register.html", "./web/register.html")
	r.StaticFile("/meet.html", "./web/meet.html")
	r.StaticFile("/admin.html", "./web/admin.html")
	r.StaticFile("/admin.css", "./web/admin.css")
	r.StaticFile("/admin.js", "./web/admin.js")
	r.StaticFile("/app.js", "./web/app.js")
	r.StaticFile("/login.css", "./web/login.css")
	r.StaticFile("/login.js", "./web/login.js")
	r.StaticFile("/register.js", "./web/register.js")
	r.StaticFile("/meet.css", "./web/meet.css")
	r.StaticFile("/style.css", "./web/style.css")
	r.StaticFile("/index.js", "./web/index.js")

	r.GET("/ws", authMW, wsHandler.ServeGin)

	auth := r.Group("/auth")
	auth.POST("/check-email", authHandler.CheckEmail)
	auth.POST("/send-code", authHandler.SendCode)
	auth.POST("/verify-code", authHandler.VerifyCode)
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)

	admin := r.Group("/admin", adminMW)
	admin.GET("/users", adminHandler.ListUsers)
	admin.DELETE("/users/:id", adminHandler.DeleteUser)
	admin.PATCH("/users/:id/role", adminHandler.SetRole)
	admin.GET("/rooms", adminHandler.ListRooms)
	admin.DELETE("/rooms/:id", adminHandler.CloseRoom)

	srv := &http.Server{Addr: cfg.Port, Handler: r}
	go func() {
		cfg.Logger.Info("server started", zap.String("address", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			cfg.Logger.Fatal("server error", zap.Error(err))
		}
	}()

	<-ctx.Done()
	cfg.Logger.Info("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		cfg.Logger.Error("shutdown error", zap.Error(err))
	}
}
