package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

type Config struct {
	DBURL        string
	RedisURL     string
	Port         string
	JWTSecret    string
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string
	Logger       *zap.Logger
}

func Load() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}

	smtpPort := 587
	if p := os.Getenv("SMTP_PORT"); p != "" {
		fmt.Sscanf(p, "%d", &smtpPort)
	}

	return &Config{
		DBURL:        os.Getenv("DATABASE_URL"),
		RedisURL:     os.Getenv("REDIS_URL"),
		Port:         os.Getenv("PORT"),
		JWTSecret:    os.Getenv("JWT_SECRET"),
		SMTPHost:     os.Getenv("SMTP_HOST"),
		SMTPPort:     smtpPort,
		SMTPUser:     os.Getenv("SMTP_USER"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:     os.Getenv("SMTP_FROM"),
		Logger:       logger,
	}, nil
}
