package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/stazoloto/zync/pkg/jwt"
)

// AuthMiddleware проверяет JWT из query-параметра ?token=
// и пробрасывает userID, имя и роль в заголовки запроса.
// Используется для WebSocket — браузер не может добавить Authorization header при ws://.
func AuthMiddleware(tokens jwt.TokenManager) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token := ctx.Query("token")
		if token == "" {
			ctx.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
			return
		}
		userID, fullName, role, err := tokens.Validate(token)
		if err != nil {
			ctx.AbortWithStatusJSON(401, gin.H{"error": "invalid token"})
			return
		}
		ctx.Set("user_id", userID)
		ctx.Set("full_name", fullName)
		ctx.Set("role", role)
		ctx.Next()
	}
}

// AdminMiddleware проверяет что пользователь имеет роль admin.
// Вешается поверх AuthMiddleware — сначала проверяем токен, потом роль.
func AdminMiddleware(tokens jwt.TokenManager) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token := ctx.GetHeader("Authorization")
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}
		if token == "" {
			ctx.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
			return
		}
		userID, fullName, role, err := tokens.Validate(token)
		if err != nil {
			ctx.AbortWithStatusJSON(401, gin.H{"error": "invalid token"})
			return
		}
		if role != "admin" {
			ctx.AbortWithStatusJSON(403, gin.H{"error": "forbidden"})
			return
		}
		ctx.Set("user_id", userID)
		ctx.Set("full_name", fullName)
		ctx.Set("role", role)
		ctx.Next()
	}
}
