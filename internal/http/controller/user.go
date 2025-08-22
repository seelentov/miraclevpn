package controller

import (
	"errors"
	"miraclevpn/internal/models"
	"miraclevpn/internal/services/user"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
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
	userIDInt, err := strconv.ParseInt(userID.(string), 10, 64)
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
	ctx.JSON(http.StatusOK, GetUserRes(u))
}

type PostChangePasswordSendReq struct {
	Username string `json:"username" binding:"required"`
}

type PostChangePasswordSendRes *MessageRes

func (c *UserController) PostChangePasswordSend(ctx *gin.Context) {
	var req PostChangePasswordSendReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": HandleValidation(ve, req)})
			return
		}
		panic(err)
	}

	if err := c.srv.ResetPasswordSend(req.Username); err != nil {

		panic(err)
	}

	ctx.JSON(http.StatusOK, PostChangePasswordSendRes(NewMessageRes("Код для смены пароля отправлен")))
}

type PostChangePasswordVerifyReq struct {
	Username          string `json:"username" binding:"required"`
	Code              int32  `json:"code" binding:"required"`
	NewPassword       string `json:"new_password" binding:"required,min=8,max=64"`
	NewPasswordVerify string `json:"new_password_verify" binding:"required"`
}

type PostChangePasswordVerifyRes *MessageRes

func (c *UserController) PostChangePasswordVerify(ctx *gin.Context) {
	var req PostChangePasswordVerifyReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": HandleValidation(ve, req)})
			return
		}
		panic(err)
	}

	if err := c.srv.ResetPasswordVerify(req.Username, req.Code, req.NewPassword, req.NewPasswordVerify); err != nil {
		if errors.Is(err, user.ErrPasswordDuplicate) {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Новый пароль не должен совпадать с текущим"})
			return
		} else if errors.Is(err, user.ErrPasswordNotEqual) {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": gin.H{"password": "Пароли не совпадают", "new_password_verify": "Пароли не совпадают"}})
			return
		} else if errors.Is(err, user.ErrWrongCode) {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "Неверный код"}})
			return
		}
		panic(err)
	}

	ctx.JSON(http.StatusOK, PostChangePasswordVerifyRes(NewMessageRes("Пароль успешно изменен")))
}
