package controller

import (
	"errors"
	"net/http"

	"miraclevpn/internal/services/auth"
	"miraclevpn/internal/services/crypt"

	"github.com/gin-gonic/gin"
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
	UID  string                 `json:"uid" binding:"required"`
	Data map[string]interface{} `json:"data" binding:"required"`
}

type PostLoginRes struct {
	Token string `json:"token"`
}

func (c *AuthController) PostLogin(ctx *gin.Context) {
	var req PostLoginReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		panic(err)
	}

	req.Data["ip"] = ctx.ClientIP()

	token, err := c.srv.Authenticate(req.UID, req.Data)
	if err != nil {
		if errors.Is(err, auth.ErrBanned) {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Пользователь заблокирован"})
			return
		}

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

	token, err := c.srv.GenerateToken(userID.(string))
	if err != nil {
		panic(err)
	}
	ctx.JSON(http.StatusOK, &PostRefreshRes{
		Token: token,
	})
}
