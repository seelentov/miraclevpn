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
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type PostLoginRes struct {
	Token  string  `json:"token"`
	Active bool    `json:"active"`
	TgLink *string `json:"tg_link,omitempty"`
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
	token, tgLink, err := c.srv.Authenticate(req.Username, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		} else if errors.Is(err, auth.ErrWrongPassword) {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный логин или пароль"})
		} else {
			panic(err)
		}
		return
	}

	res := &PostLoginRes{
		Token:  token,
		Active: true,
	}

	if tgLink != "" {
		res.TgLink = &tgLink
		res.Active = false
	}

	ctx.JSON(http.StatusOK, res)
}

type PostRegisterReq struct {
	Username      string `json:"username" binding:"required"`
	Password      string `json:"password" binding:"required,min=8,max=64"`
	CheckPassword string `json:"check_password" binding:"required"`
}

type PostRegisterRes struct {
	Token  string  `json:"token"`
	TgLink *string `json:"tg_link,omitempty"`
}

func (c *AuthController) PostRegister(ctx *gin.Context) {
	var req PostRegisterReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": HandleValidation(ve, req)})
			return
		}

		panic(err)
	}
	token, tgLink, err := c.srv.SignUp(req.Username, req.Password, req.CheckPassword)
	if err != nil {
		if errors.Is(err, auth.ErrAlreadyExists) {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": gin.H{"username": "Пользователь с этим логином уже существует"}})
			return
		} else if errors.Is(err, auth.ErrNotEqualPasswords) {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": gin.H{"password": "Пароли не совпадают", "check_password": "Пароли не совпадают"}})
			return
		}

		panic(err)
	}

	ctx.JSON(http.StatusOK, &PostRegisterRes{
		Token:  token,
		TgLink: &tgLink,
	})
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
