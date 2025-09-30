package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ViewIndexController struct {
}

func NewViewIndexController() *ViewIndexController {
	return &ViewIndexController{}
}

func (c *ViewIndexController) GetIndex(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "index.html", nil)
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
