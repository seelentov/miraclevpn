package controller

import (
	"bytes"
	"io"
	"log"
	"miraclevpn/internal/models"
	"miraclevpn/internal/services/info"
	"net/http"
	"strconv"
	"time"

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

func (c *InfoController) PostEcho(ctx *gin.Context) {
	requestInfo := gin.H{
		"method":      ctx.Request.Method,
		"url":         ctx.Request.URL.String(),
		"protocol":    ctx.Request.Proto,
		"host":        ctx.Request.Host,
		"remote_addr": ctx.Request.RemoteAddr,
		"timestamp":   time.Now().Format(time.RFC3339),
	}

	// Собираем заголовки
	headers := gin.H{}
	for name, values := range ctx.Request.Header {
		if len(values) == 1 {
			headers[name] = values[0]
		} else {
			headers[name] = values
		}
	}
	requestInfo["headers"] = headers

	// Собираем query parameters
	queryParams := gin.H{}
	for name, values := range ctx.Request.URL.Query() {
		if len(values) == 1 {
			queryParams[name] = values[0]
		} else {
			queryParams[name] = values
		}
	}
	requestInfo["query_params"] = queryParams

	// Читаем тело запроса
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		requestInfo["body"] = "Error reading body: " + err.Error()
	} else {
		requestInfo["body"] = string(body)

		// Восстанавливаем тело для возможного дальнейшего использования
		ctx.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	}

	// Логируем всю информацию
	log.Printf("=== INCOMING REQUEST ===")
	log.Printf("Method: %s", requestInfo["method"])
	log.Printf("URL: %s", requestInfo["url"])
	log.Printf("RemoteAddr: %s", requestInfo["remote_addr"])
	log.Printf("Headers: %+v", headers)
	log.Printf("QueryParams: %+v", queryParams)
	log.Printf("Body: %s", requestInfo["body"])
	log.Printf("========================")

	// Формируем ответ
	response := gin.H{
		"status":   "success",
		"message":  "Request received",
		"request":  requestInfo,
		"received": time.Now().Format(time.RFC3339),
	}

	ctx.JSON(http.StatusOK, response)
}
