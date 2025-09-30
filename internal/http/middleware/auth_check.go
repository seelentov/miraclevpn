package middleware

import (
	"miraclevpn/internal/repo"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func CheckUserMiddleware(userRepo *repo.UserRepository) gin.HandlerFunc {
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

		if u.ExpiredAt.Before(time.Now()) {
			ctx.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{"error": "Подписка истекла"})
			return
		}

		if u.Banned {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Аккаунт заблокирован"})
			return
		}

		if !u.Active {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Аккаунт деактивирован"})
			return
		}

		ctx.Next()
	}
}
