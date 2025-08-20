package controller

import (
	"errors"
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
		panic(err)
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
		panic(err)
	}
	ctx.JSON(http.StatusOK, servers)
}

func (c *ServerController) GetServer(ctx *gin.Context) {
	userID, _ := ctx.Get("user_id")
	userIDInt, err := strconv.ParseInt(userID.(string), 10, 64)
	if err != nil {
		panic(err)
	}

	id := ctx.Param("id")
	if id == "" {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID не указан"})
		return
	}
	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		panic(err)
	}
	server, err := c.srv.GetServerByID(idInt)
	if err != nil {
		if errors.Is(err, servers.ErrNotFound) {
			ctx.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Сервер не найден"})
			return
		}
		panic(err)
	}
	config, err := c.srv.GetConfig(userIDInt, idInt)
	if err != nil {
		panic(err)
	}

	ctx.JSON(http.StatusOK, gin.H{"server": server, "config": config})
}

func (c *ServerController) GetRegions(ctx *gin.Context) {
	regions, err := c.srv.GetRegions()
	if err != nil {
		panic(err)
	}
	ctx.JSON(http.StatusOK, regions)
}
