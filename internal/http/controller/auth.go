package controller

import (
	"errors"
	"net/http"
	"strconv"

	"miraclevpn/internal/services/auth"
	"miraclevpn/internal/services/crypt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type AuthController struct {
	srv *auth.AuthService
	jwt *crypt.JwtService
}

func NewAuthController(srv *auth.AuthService, jwt *crypt.JwtService) *AuthController {
	return &AuthController{
		srv: srv,
		jwt: jwt,
	}
}

type PostLoginReq struct {
	UID string `json:"uid" binding:"required"`
}

type PostLoginRes struct {
	Token string `json:"token"`
}

func (c *AuthController) PostLogin(ctx *gin.Context) {
	var req PostLoginReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			ctx.JSON(http.StatusBadRequest, gin.H{"errors": HandleValidation(ve, req)})
			return
		}

		panic(err)
	}

	token, err := c.srv.Authenticate(req.UID)
	if err != nil {
		panic(err)
	}

	res := &PostLoginRes{
		Token: token,
	}

	ctx.JSON(http.StatusOK, res)
}

type PostRefreshRes struct {
	Token string `json:"token"`
}

func (c *AuthController) PostRefresh(ctx *gin.Context) {
	userID, _ := ctx.Get("user_id")
	userIDInt, err := strconv.ParseInt(userID.(string), 10, 64)
	if err != nil {
		panic(err)
	}

	token, err := c.srv.GenerateToken(userIDInt)
	if err != nil {
		panic(err)
	}
	ctx.JSON(http.StatusOK, &PostRefreshRes{
		Token: token,
	})
}
