package controller

import (
	"errors"
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

type PostResetSendReq struct {
	Phone string `json:"phone" binding:"required"`
}

func (c *UserController) PostResetSend(ctx *gin.Context) {
	var req PostResetSendReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": HandleValidation(ve, req)})
			return
		}

		panic(err)
	}

	if _, err := c.srv.ResetPasswordSend(req.Phone); err != nil {
		panic(err)
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Код для восстановления пароля отправлен в Telegram"})
}

type PostResetVerifyReq struct {
	Phone             string `json:"phone" binding:"required"`
	Code              int32  `json:"code" binding:"required"`
	NewPassword       string `json:"new_password" binding:"required"`
	NewPasswordVerify string `json:"new_password_verify" binding:"required"`
}

func (c *UserController) PostResetVerify(ctx *gin.Context) {
	var req PostResetVerifyReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": HandleValidation(ve, req)})
			return
		}

		panic(err)
	}

	if err := c.srv.ResetPasswordVerify(req.Phone, req.Code, req.NewPassword, req.NewPasswordVerify); err != nil {
		if errors.Is(err, user.ErrPasswordNotEqual) {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Пароли не совпадают"})
			return
		}

		if errors.Is(err, user.ErrWrongCode) {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Неверный код"})
			return
		}

		panic(err)
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Пароль успешно сброшен"})
}
