package controller

import (
	"errors"
	"miraclevpn/internal/models"
	"miraclevpn/internal/repo"
	"miraclevpn/internal/services/servers"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ServerController struct {
	srv *servers.ServersService

	configDuration int
}

func NewServerController(srv *servers.ServersService, configDuration int) *ServerController {
	return &ServerController{
		srv,
		configDuration,
	}
}

type GetServers []*models.Server

func (c *ServerController) GetServers(ctx *gin.Context) {
	servers, err := c.srv.GetAllServers()
	if err != nil {
		panic(err)
	}
	ctx.JSON(http.StatusOK, GetServers(servers))
}

type GetServersByRegionRes []*models.Server

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
	ctx.JSON(http.StatusOK, GetServersByRegionRes(servers))
}

type GetServerRes struct {
	Server           *models.Server `json:"server"`
	Config           string         `json:"config"`
	ConfigExpiration int            `json:"config_expiration"`
}

func (c *ServerController) GetServer(ctx *gin.Context) {
	userID, _ := ctx.Get("user_id")

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

	if server.Preview {
		ctx.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Сервер не найден"})
		return
	}

	config, err := c.srv.GetConfig(userID.(string), idInt)
	if err != nil {
		panic(err)
	}

	ctx.JSON(http.StatusOK, GetServerRes{
		Server:           server,
		Config:           config,
		ConfigExpiration: c.configDuration,
	})
}

type GetRegionsRes []*models.Region

func (c *ServerController) GetRegions(ctx *gin.Context) {
	regions, err := c.srv.GetRegions()
	if err != nil {
		panic(err)
	}
	ctx.JSON(http.StatusOK, GetRegionsRes(regions))
}

type GetServerStatusRes struct {
	Server            *models.Server `json:"server"`
	CurrentUsersCount int            `json:"current_users_count"`
}

func (c *ServerController) GetServerStatus(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID не указан"})
		return
	}
	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		panic(err)
	}
	server, currentUsersCount, err := c.srv.GetServerStatus(idInt)
	if err != nil {
		panic(err)
	}
	ctx.JSON(http.StatusOK, GetServerStatusRes{
		Server:            server,
		CurrentUsersCount: currentUsersCount,
	})
}

type GetRegionStatusRes struct {
	Servers           []*models.Server `json:"servers"`
	CurrentUsersCount int              `json:"current_users_count"`
}

func (c *ServerController) GetRegionStatus(ctx *gin.Context) {
	region := ctx.Param("region")
	if region == "" {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID не указан"})
		return
	}
	servers, currentUsersCount, err := c.srv.GetRegionStatus(region)
	if err != nil {
		panic(err)
	}
	ctx.JSON(http.StatusOK, GetRegionStatusRes{
		Servers:           servers,
		CurrentUsersCount: currentUsersCount,
	})
}

type GetPreviewRes []*models.Server

func (c *ServerController) GetPreview(ctx *gin.Context) {
	srv, err := c.srv.FindPreview()
	if err != nil {
		panic(err)
	}

	ctx.JSON(http.StatusOK, GetPreviewRes(srv))
}

type PostRequestReq struct {
	Region string `json:"region" binding:"required"`
}

type PostRequestRes []*models.Server

func (c *ServerController) PostRequest(ctx *gin.Context) {
	userID, _ := ctx.Get("user_id")

	var req PostRequestReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		panic(err)
	}

	if err := c.srv.SendRequest(req.Region, userID.(string)); err != nil {
		if errors.Is(err, repo.ErrReqAlreadyExist) {
			ctx.JSON(http.StatusOK, nil)
			return
		}

		panic(err)
	}

	ctx.JSON(http.StatusOK, nil)
}
