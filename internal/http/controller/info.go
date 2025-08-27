package controller

import (
	"errors"
	"miraclevpn/internal/models"
	"miraclevpn/internal/services/info"
	"net/http"

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
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "slug не указан"})
		return
	}

	i, err := c.srv.GetInfo(slug)
	if err != nil {
		if errors.Is(err, info.ErrNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "информация не найдена"})
			return
		}
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
