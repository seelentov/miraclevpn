// Package cookie provides cookie logic
package cookie

import (
	"time"

	"github.com/gin-gonic/gin"
)

var AuthCookie = "auth"

type CookieService struct {
	domain string
}

func NewCookieService(domain string) *CookieService {
	return &CookieService{domain}
}

func (s *CookieService) SetAuth(ctx *gin.Context, token string) {
	ctx.SetCookie(AuthCookie, token, int(time.Now().Add(time.Hour).Unix()), "/", s.domain, false, true)
}

func (s *CookieService) RemoveAuth(ctx *gin.Context) {
	ctx.SetCookie(AuthCookie, "", -1, "/", s.domain, false, true)
}

func (s *CookieService) GetAuth(ctx *gin.Context) (string, error) {
	return ctx.Cookie(AuthCookie)
}
