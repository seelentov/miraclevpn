package controller

import (
	"errors"
	"miraclevpn/internal/models"
	"miraclevpn/internal/services/user"
	"net/http"

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

type GetUserRes *models.User

func (c *UserController) GetUser(ctx *gin.Context) {
	userID, _ := ctx.Get("user_id")

	u, err := c.srv.GetUserByID(userID.(string))
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			ctx.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		} else {
			panic(err)
		}
		return
	}

	u.PaymentID = nil

	ctx.JSON(http.StatusOK, GetUserRes(u))
}
