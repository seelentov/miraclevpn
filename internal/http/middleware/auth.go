package middleware

import (
	"strings"

	"miraclevpn/internal/services/crypt"

	"github.com/gin-gonic/gin"
)

func SetUserIDMiddleware(jwtSrv *crypt.JwtService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		if authHeader != "" && !strings.HasPrefix(authHeader, "Bearer 0") && strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := jwtSrv.ParseToken(tokenStr)
			if err == nil && claims.UserID != "" {
				ctx.Set("user_id", claims.UserID)
			}
		}
		ctx.Next()
	}
}
