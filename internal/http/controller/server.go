package controller

import (
	"miraclevpn/internal/services/servers"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ServerController struct {
	srv *servers.ServersService
}

func NewServerController(srv *servers.ServersService) *ServerController {
	return &ServerController{
		srv,
	}
}

func (c *ServerController) GetServers(ctx *gin.Context) {
	servers, err := c.srv.GetAllServers()
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сервера"})
		return
	}
	ctx.JSON(http.StatusOK, servers)
}

func (c *ServerController) GetServersByRegion(ctx *gin.Context) {
	region := ctx.Param("region")
	if region == "" {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Регион не указан"})
		return
	}
	servers, err := c.srv.GetServersByRegion(region)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сервера"})
		return
	}
	ctx.JSON(http.StatusOK, servers)
}

func (c *ServerController) GetServer(ctx *gin.Context) {
	userID := ctx.Param("user_id")
	userIDInt, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сервера"})
		return
	}

	id := ctx.Param("id")
	if id == "" {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID не указан"})
		return
	}
	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сервера"})
		return
	}
	server, err := c.srv.GetServerByID(idInt)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сервера"})
		return
	}
	config, err := c.srv.GetConfig(userIDInt, idInt)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сервера"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"server": server, "config": config})
}
