package middleware

import (
	"errors"
	"fmt"
	"log"
	"miraclevpn/internal/repo"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

var (
	ErrFAuth         = errors.New("auth err")
	ErrFAuthEmpty    = fmt.Errorf("%w:%s", ErrFAuth, "empty auth")
	ErrFAuthNotFound = fmt.Errorf("%w:%s", ErrFAuth, "user not fount")
	ErrFAuthBanned   = fmt.Errorf("%w:%s", ErrFAuth, "user banned")
	ErrFAuthDisabled = fmt.Errorf("%w:%s", ErrFAuth, "user disabled")
)

func AuthReqFrontend(userRepo *repo.UserRepository) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID, exists := ctx.Get("user_id")
		if !exists || userID == "" {
			url := ctx.Request.URL.String()
			log.Println("REDIRECT " + url)
			ctx.Redirect(http.StatusMovedPermanently, "/login?redirectUrl="+strings.ReplaceAll(url, "/", "%2F"))
			ctx.Abort()
			return
		}

		u, err := userRepo.FindByID(userID.(string))
		if err != nil || u == nil {
			ctx.Abort()
			panic(ErrFAuthNotFound)
		}

		if u.Banned {
			ctx.Abort()
			panic(ErrFAuthBanned)
		}

		if !u.Active {
			ctx.Abort()
			panic(ErrFAuthDisabled)
		}

		ctx.Next()
	}
}
