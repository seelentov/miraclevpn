package controller

import (
	"errors"
	"net/http"

	"miraclevpn/internal/services/auth"
	"miraclevpn/internal/services/crypt"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	srv         *auth.AuthService
	jwt         *crypt.JwtService
	jwtDuration int
}

func NewAuthController(srv *auth.AuthService, jwt *crypt.JwtService, jwtDuration int) *AuthController {
	return &AuthController{
		srv:         srv,
		jwt:         jwt,
		jwtDuration: jwtDuration,
	}
}

type PostLoginReq struct {
	UID  string                 `json:"uid" binding:"required"`
	Data map[string]interface{} `json:"data" binding:"required"`
}

type PostLoginRes struct {
	Token         string `json:"token"`
	ExpirationMin int    `json:"expiration_min"`
}

func (c *AuthController) PostLogin(ctx *gin.Context) {
	var req PostLoginReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		panic(err)
	}

	req.Data["ip"] = ctx.GetHeader("X-Real-Ip")

	token, err := c.srv.Authenticate(req.UID, req.Data)
	if err != nil && !errors.Is(err, auth.ErrBanned) && !errors.Is(err, auth.ErrExpired) {
		panic(err)
	}

	res := &PostLoginRes{
		Token:         token,
		ExpirationMin: c.jwtDuration,
	}

	ctx.JSON(http.StatusOK, res)
}

type PostRefreshRes struct {
	Token         string `json:"token"`
	ExpirationMin int    `json:"expiration_min"`
}

func (c *AuthController) PostRefresh(ctx *gin.Context) {
	userID, _ := ctx.Get("user_id")

	token, err := c.srv.GenerateToken(userID.(string))
	if err != nil {
		panic(err)
	}
	ctx.JSON(http.StatusOK, &PostRefreshRes{
		Token:         token,
		ExpirationMin: c.jwtDuration,
	})
}
