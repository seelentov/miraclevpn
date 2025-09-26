package controller

import (
	"miraclevpn/internal/services/payment"
	"miraclevpn/internal/services/user"
	"miraclevpn/pkg/yookassa"
	"net/http"

	"github.com/gin-gonic/gin"
)

type PaymentController struct {
	payService  *payment.PaymentService
	userService *user.UserService
}

func NewPaymentController(payService *payment.PaymentService, userService *user.UserService) *PaymentController {
	return &PaymentController{
		payService:  payService,
		userService: userService,
	}
}

type PostCreateReq struct {
	Email  string `json:"email" binding:"required"`
	UserID string `json:"user_id" binding:"required"`
	PlanID int64  `json:"plan_id" binding:"required"`
}

type PostCreateRes struct {
	PayURL string `json:"pay_url"`
}

func (c *PaymentController) PostCreate(ctx *gin.Context) {
	var req PostCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		panic(err)
	}

	plan, err := c.payService.FindPlanByID(req.PlanID)
	if err != nil {
		panic(err)
	}

	payURL, err := c.payService.Create(req.UserID, req.Email, plan, true)
	if err != nil {
		panic(err)
	}

	if err := c.userService.UpdateEmail(req.UserID, req.Email); err != nil {
		panic(err)
	}

	ctx.JSON(http.StatusOK, PostCreateRes{
		PayURL: payURL,
	})
}

type PostPaymentHookReq yookassa.WebHookRes

func (c *PaymentController) PostPaymentHook(ctx *gin.Context) {
	var req PostPaymentHookReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		panic(err)
	}

	payment, err := c.payService.Find(req.Object.ID)
	if err != nil {
		panic(err)
	}

	token := req.Object.Metadata["token"]

	if err := c.payService.ValidateToken(token, payment.UserID, payment.PlanID); err != nil {
		panic(err)
	}

	if err := c.userService.AddDays(payment.UserID, payment.Days); err != nil {
		panic(err)
	}

	if req.Object.PaymentMethod.Saved {
		if err := c.userService.UpdatePaymentMethod(payment.UserID, req.Object.PaymentMethod.ID, payment.PlanID); err != nil {
			panic(err)
		}
	}

	if err := c.payService.Done(payment.YooKassaID); err != nil {
		panic(err)
	}

	ctx.JSON(http.StatusOK, NewMessageRes("ok"))
}

func (c *PaymentController) PostRemovePaymentMethod(ctx *gin.Context) {
	userID, _ := ctx.Get("user_id")

	if err := c.userService.RemovePaymentMethod(userID.(string)); err != nil {
		panic(err)
	}

	ctx.JSON(http.StatusOK, NewMessageRes("ok"))
}
