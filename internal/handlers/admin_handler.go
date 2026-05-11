package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stazoloto/zync/internal/service"
	"go.uber.org/zap"
)

type AdminHandler struct {
	admin  service.AdminService
	logger *zap.Logger
}

func NewAdminHandler(admin service.AdminService, logger *zap.Logger) *AdminHandler {
	return &AdminHandler{admin: admin, logger: logger}
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	users, err := h.admin.ListUsers()
	if err != nil {
		h.logger.Error("admin: list users failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	type userDTO struct {
		ID        string `json:"id"`
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Role      string `json:"role"`
		CreatedAt string `json:"created_at"`
	}

	result := make([]userDTO, 0, len(users))
	for _, u := range users {
		result = append(result, userDTO{
			ID:        u.ID,
			Email:     u.Email,
			FirstName: u.FirstName,
			LastName:  u.LastName,
			Role:      u.Role,
			CreatedAt: u.CreatedAt.Format("02.01.2006 15:04"),
		})
	}

	c.JSON(http.StatusOK, result)
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user id"})
		return
	}
	if err := h.admin.DeleteUser(userID); err != nil {
		h.logger.Error("admin: delete user failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *AdminHandler) ListRooms(c *gin.Context) {
	rooms, err := h.admin.ListActiveRooms(c.Request.Context())
	if err != nil {
		h.logger.Error("admin: list rooms failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, rooms)
}

func (h *AdminHandler) CloseRoom(c *gin.Context) {
	roomID := c.Param("id")
	if roomID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing room id"})
		return
	}
	if err := h.admin.CloseRoom(c.Request.Context(), roomID); err != nil {
		h.logger.Error("admin: close room failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *AdminHandler) SetRole(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user id"})
		return
	}

	var req struct {
		Role string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := h.admin.SetRole(userID, req.Role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
