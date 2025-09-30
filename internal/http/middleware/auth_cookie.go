package middleware

import (
	"log"
	"miraclevpn/internal/services/cookie"
	"miraclevpn/internal/services/crypt"

	"github.com/gin-gonic/gin"
)

func AuthCookie(jwtSrv *crypt.JwtService, cookieSrv *cookie.CookieService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		auth, err := cookieSrv.GetAuth(ctx)

		token := ctx.Request.URL.Query().Get("token")
		log.Println(token)
		if token != "" {
			cookieSrv.SetAuth(ctx, token)
			auth = token
		}

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
		} else {
			log.Println(err)
		}
		ctx.Next()
	}
}
