package middleware

import (
	"miraclevpn/internal/services/cookie"
	"miraclevpn/internal/services/crypt"

	"github.com/gin-gonic/gin"
)

func AuthCookie(jwtSrv *crypt.JwtService, cookieSrv *cookie.CookieService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token := ctx.Param("token")
		if token != "" {
			cookieSrv.SetAuth(ctx, token)
		}

		auth, err := cookieSrv.GetAuth(ctx)
		if auth != "" && err == nil {
			claims, err := jwtSrv.ParseToken(auth)
			if err == nil {
				userID, ok := claims.Data["user_id"]
				if ok {
					ctx.Set("user_id", userID)
				}
			} else {
				cookieSrv.RemoveAuth(ctx)
			}
		}
		ctx.Next()

	}
}
