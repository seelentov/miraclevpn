package controller

import (
	"errors"
	"net/http"

	"miraclevpn/internal/services/auth"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	srv *auth.AuthService
}

func NewAuthController(srv *auth.AuthService) *AuthController {
	return &AuthController{
		srv: srv,
	}
}

type PostLoginReq struct {
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (c *AuthController) PostLogin(ctx *gin.Context) {
	var req PostLoginReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	token, err := c.srv.Authenticate(req.Phone, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		} else if errors.Is(err, auth.ErrWrongPassword) {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный логин или пароль"})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сервера"})
		}
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"token": token})
}

type PostRegisterReq struct {
	Phone         string `json:"phone" binding:"required"`
	Password      string `json:"password" binding:"required"`
	CheckPassword string `json:"check_password" binding:"required"`
}

func (c *AuthController) PostRegister(ctx *gin.Context) {
	var req PostRegisterReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	token, err := c.srv.SignIn(req.Phone, req.Password, req.CheckPassword)
	if err != nil {
		if errors.Is(err, auth.ErrAlreadyExists) {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Пользователь с этим номером телефона уже существует"})
		} else if errors.Is(err, auth.ErrNotEqualPasswords) {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Пароли не совпадают"})
		} else {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сервера"})
		}
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"token": token})
}
