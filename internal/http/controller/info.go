package controller

import (
	"miraclevpn/internal/models"
	"miraclevpn/internal/services/info"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type InfoController struct {
	srv *info.InfoService
}

func NewInfoController(srv *info.InfoService) *InfoController {
	return &InfoController{srv: srv}
}

type GetTechWorkRes struct {
	TechWork     bool   `json:"tech_work"`
	TechWorkText string `json:"tech_work_text"`
}

func (c *InfoController) GetTechWork(ctx *gin.Context) {
	techWork, techWorkText, err := c.srv.GetTechWork()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, GetTechWorkRes{TechWork: techWork, TechWorkText: techWorkText})
}

type GetNewsRes []*models.News

func (c *InfoController) GetNews(ctx *gin.Context) {
	userID, _ := ctx.Get("user_id")
	news, err := c.srv.GetNews(userID.(string))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, GetNewsRes(news))
}

type GetInfoRes *models.Info

func (c *InfoController) GetInfo(ctx *gin.Context) {
	slug := ctx.Param("slug")
	if slug == "" {
		panic("slug is empty")
	}

	i, err := c.srv.GetInfo(slug)
	if err != nil {
		panic(err)
	}

	ctx.JSON(http.StatusOK, GetInfoRes(i))
}

type GetInfosRes []*models.Info

func (c *InfoController) GetInfos(ctx *gin.Context) {
	i, err := c.srv.GetInfos()
	if err != nil {
		panic(err)
	}

	ctx.JSON(http.StatusOK, GetInfosRes(i))
}

type GetSupportRes map[string]string

func (c *InfoController) GetSupport(ctx *gin.Context) {
	i, err := c.srv.GetSupport()
	if err != nil {
		panic(err)
	}

	ctx.JSON(http.StatusOK, GetSupportRes(i))
}

type GetPaymentPlansRes []*models.PaymentPlan

func (c *InfoController) GetPaymentPlans(ctx *gin.Context) {
	i, err := c.srv.GetPaymentPlans()
	if err != nil {
		panic(err)
	}

	ctx.JSON(http.StatusOK, GetPaymentPlansRes(i))
}

type GetPaymentPlanRes *models.PaymentPlan

func (c *InfoController) GetPaymentPlan(ctx *gin.Context) {
	planIDStr := ctx.Param("plan_id")
	if planIDStr == "" {
		panic("plan_id is empty")
	}

	planID, err := strconv.Atoi(planIDStr)
	if err != nil {
		panic(err)
	}

	i, err := c.srv.GetPaymentPlan(int64(planID))
	if err != nil {
		panic(err)
	}

	ctx.JSON(http.StatusOK, GetPaymentPlanRes(i))
}

func (c *InfoController) GetPing(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"message": "pong"})
}
