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
	Code int32 `json:"code" binding:"required"`
}

func (c *UserController) PostActivate(ctx *gin.Context) {
	userID := ctx.Param("user_id")
	userIDInt, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		panic(err)
	}

	var req PostActivateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err = c.srv.Activate(userIDInt, req.Code)
	if err != nil {
		if err == user.ErrWrongCode {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Неверный код активации"})
		} else {
			panic(err)
		}
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Пользователь успешно активирован"})
}
