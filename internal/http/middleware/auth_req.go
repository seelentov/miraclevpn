package middleware

import (
	"miraclevpn/internal/repo"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RequireAuthMiddleware(userRepo *repo.UserRepository) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID, exists := ctx.Get("user_id")
		if !exists || userID == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Пользователь не авторизован"})
			return
		}

		u, err := userRepo.FindByID(userID.(string))
		if err != nil || u == nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Пользователь не авторизован"})
			return
		}
		ctx.Next()
	}
}
