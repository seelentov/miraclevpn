package controller

import (
	"miraclevpn/internal/models"
	"miraclevpn/internal/services/payment"
	"miraclevpn/internal/services/user"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ViewPaymentController struct {
	payService  *payment.PaymentService
	userService *user.UserService
}

func NewViewPaymentController(payService *payment.PaymentService, userService *user.UserService) *ViewPaymentController {
	return &ViewPaymentController{payService, userService}
}

type GetPaymentsViewModel struct {
	ViewBase
	Plans []*models.PaymentPlan
}

func (c *ViewPaymentController) GetPayments(ctx *gin.Context) {
	plans, err := c.payService.FindAllPlans()
	if err != nil {
		panic(err)
	}

	ctx.HTML(http.StatusOK, "payments.html", GetPaymentsViewModel{
		Plans: plans,
	})
}

type PostPaymentReq struct {
	Email  string `form:"email" binding:"required,email"`
	PlanID int64  `form:"plan_id" binding:"required"`
}

func (c *ViewPaymentController) PostPayment(ctx *gin.Context) {
	userIDAny, _ := ctx.Get("user_id")
	userID := userIDAny.(string)

	var req PostPaymentReq
	if err := ctx.ShouldBind(&req); err != nil {
		panic(err)
	}

	plan, err := c.payService.FindPlanByID(req.PlanID)
	if err != nil {
		panic(err)
	}

	payURL, err := c.payService.Create(userID, req.Email, plan, true)
	if err != nil {
		panic(err)
	}

	if err := c.userService.UpdateEmail(userID, req.Email); err != nil {
		panic(err)
	}

	ctx.Redirect(http.StatusMovedPermanently, payURL)
}

func (c *ViewPaymentController) PostRemovePaymentMethod(ctx *gin.Context) {
	userID, _ := ctx.Get("user_id")

	if err := c.userService.RemovePaymentMethod(userID.(string)); err != nil {
		panic(err)
	}

	ctx.Redirect(http.StatusMovedPermanently, "/user")
}
