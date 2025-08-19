package controller

import (
	"errors"
	"net/http"
	"strconv"

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
			panic(err)
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
	token, tgLink, err := c.srv.SignIn(req.Phone, req.Password, req.CheckPassword)
	if err != nil {
		if errors.Is(err, auth.ErrAlreadyExists) {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Пользователь с этим номером телефона уже существует"})
		} else if errors.Is(err, auth.ErrNotEqualPasswords) {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Пароли не совпадают"})
		} else {
			panic(err)
		}
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"token": token, "tg_link": tgLink})
}

type PostActivateReq struct {
	TgToken string `json:"tg_token" binding:"required"`
	ChatID  int64  `json:"chat_id" binding:"required"`
}

func (c *AuthController) PostActivate(ctx *gin.Context) {
	var req PostActivateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	claims, err := c.jwt.ParseToken(req.TgToken)
	if err != nil || claims.UserID == "" {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Неверный токен"})
		return
	}

	userID, err := strconv.ParseInt(claims.UserID, 10, 64)
	if err != nil {
		panic(err)
	}

	if err := c.srv.Activate(userID, req.ChatID); err != nil {
		panic(err)
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Пользователь успешно активирован"})
}
