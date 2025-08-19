package controller

import (
	"errors"
	"miraclevpn/internal/services/user"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	srv *user.UserService
}

func NewUserController(srv *user.UserService) *UserController {
	return &UserController{
		srv,
	}
}

func (c *UserController) GetUser(ctx *gin.Context) {
	userID := ctx.Param("user_id")
	userIDInt, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		panic(err)
	}
	u, err := c.srv.GetUserByID(userIDInt)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			ctx.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		} else {
			panic(err)
		}
		return
	}
	ctx.JSON(http.StatusOK, u)
}

type PostActivateReq struct {
	TgToken string `json:"tg_token" binding:"required"`
	ChatID  int64  `json:"chat_id" binding:"required"`
}
