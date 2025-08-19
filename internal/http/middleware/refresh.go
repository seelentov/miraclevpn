package middleware

import (
	"net/http"
	"strings"
	"time"

	"miraclevpn/internal/services/crypt"

	"github.com/gin-gonic/gin"
)

func RefreshTokenMiddleware(jwtSrv *crypt.JwtService, duration time.Duration) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			ctx.Next()
			return
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := jwtSrv.ParseToken(tokenStr)
		if err != nil || claims.UserID == "" {
			ctx.Next()
			return
		}

		newToken, err := jwtSrv.GenerateToken(claims.UserID, duration)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Не удалось обновить токен"})
			return
		}

		ctx.Header("Authorization", "Bearer "+newToken)
		ctx.Next()
	}
}
