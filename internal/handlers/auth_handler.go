package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stazoloto/zync/internal/service"
	"go.uber.org/zap"
)

type sendCodeRequest struct {
	Email string `json:"email"`
}

type verifyCodeRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type registerRequest struct {
	VerifiedToken string `json:"verified_token"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	Password      string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string `json:"token"`
}

type AuthHandler struct {
	auth   service.UserService
	logger *zap.Logger
}

func NewAuthHandler(auth service.UserService, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{auth: auth, logger: logger}
}

// SendCode отправляет код подтверждения на указанный email.
func (h *AuthHandler) CheckEmail(c *gin.Context) {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	exists, err := h.auth.EmailExists(c.Request.Context(), req.Email)
	if err != nil {
		h.logger.Error("check email failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.Status(http.StatusOK)
}

func (h *AuthHandler) SendCode(c *gin.Context) {
	var req sendCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := h.auth.SendVerificationCode(c.Request.Context(), req.Email); err != nil {
		h.logger.Error("send code failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send code"})
		return
	}
	c.Status(http.StatusNoContent)
}

// VerifyCode проверяет код из email и возвращает JWT-токен для регистрации.
func (h *AuthHandler) VerifyCode(c *gin.Context) {
	var req verifyCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Email == "" || req.Code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	token, err := h.auth.VerifyCode(c.Request.Context(), req.Email, req.Code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid code"})
		return
	}
	c.JSON(http.StatusOK, authResponse{Token: token})
}

// Register регистрирует нового пользователя и возвращает JWT-токен.
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil ||
		req.VerifiedToken == "" || req.Password == "" ||
		req.FirstName == "" || req.LastName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	token, err := h.auth.Register(c.Request.Context(), req.VerifiedToken, req.FirstName, req.LastName, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, authResponse{Token: token})
}

// Login проверяет учетные данные пользователя и возвращает JWT-токен при успешной аутентификации.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	token, err := h.auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}
	c.JSON(http.StatusOK, authResponse{Token: token})
}
