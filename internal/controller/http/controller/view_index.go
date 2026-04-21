package controller

import (
	"miraclevpn/internal/models"
	"miraclevpn/internal/repo"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ViewIndexController struct {
	reviewRepo *repo.ReviewRepository
}

func NewViewIndexController(reviewRepo *repo.ReviewRepository) *ViewIndexController {
	return &ViewIndexController{reviewRepo: reviewRepo}
}

type GetIndexViewModel struct {
	ViewBase
	Reviews []*models.Review
}

func (c *ViewIndexController) GetIndex(ctx *gin.Context) {
	reviews, _ := c.reviewRepo.FindActive()
	ctx.HTML(http.StatusOK, "index.html", GetIndexViewModel{
		ViewBase: ViewBase{ShowHeaderNav: true},
		Reviews:  reviews,
	})
}

func (c *ViewIndexController) NotFound(ctx *gin.Context) {
	ctx.HTML(http.StatusNotFound, "404.html", nil)
}

func (c *ViewIndexController) Panic(ctx *gin.Context, err interface{}) {
	ctx.HTML(http.StatusInternalServerError, "500.html", nil)
}

func (c *ViewIndexController) GetSuccessPayment(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "success_payment.html", nil)
}
