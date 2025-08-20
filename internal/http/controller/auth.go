package controller

import (
	"errors"
	"net/http"

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

func (c *AuthController) PostLogin(ctx *gin.Context) {
	var req PostLoginReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": HandleValidation(ve, req)})
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

	res := gin.H{"token": token, "active": true}

	if tgLink != "" {
		res["tg_link"] = tgLink
		res["active"] = false
	}

	ctx.JSON(http.StatusOK, res)
}

type PostRegisterReq struct {
	Username      string `json:"username" binding:"required"`
	Password      string `json:"password" binding:"required,min=8,max=64"`
	CheckPassword string `json:"check_password" binding:"required"`
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
	ctx.JSON(http.StatusOK, gin.H{"token": token, "tg_link": tgLink})
}
