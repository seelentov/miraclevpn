package middleware

import (
	"errors"
	"fmt"
	"miraclevpn/internal/repo"
	"net/http"

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
			ctx.Redirect(http.StatusOK, "/login")
		}

		u, err := userRepo.FindByID(userID.(string))
		if err != nil || u == nil {
			panic(ErrFAuthNotFound)
		}

		if u.Banned {
			panic(ErrFAuthBanned)
		}

		if !u.Active {
			panic(ErrFAuthDisabled)
		}

		ctx.Next()
	}
}
