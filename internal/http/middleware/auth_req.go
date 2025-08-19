package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RequireUserIDMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID, exists := ctx.Get("user_id")
		if !exists || userID == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Пользователь не авторизован"})
			return
		}
		ctx.Next()
	}
}
