package middleware

import (
	"net/http"
	"strings"

	"miraclevpn/internal/services/crypt"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(jwtSrv *crypt.JwtService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Пользователь не авторизован"})
			return
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := jwtSrv.ParseToken(tokenStr)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Неверный токен"})
			return
		}
		ctx.Set("user_id", claims.UserID)
		ctx.Next()
	}
}
