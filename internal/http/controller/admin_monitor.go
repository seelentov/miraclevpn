package controller

import (
	"miraclevpn/internal/services/admin"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AdminMonitorController struct {
	monitorSrv *admin.MonitorService
}

func NewAdminMonitorController(monitorSrv *admin.MonitorService) *AdminMonitorController {
	return &AdminMonitorController{monitorSrv}
}

type AGetIndexViewModel struct {
	Hosts []*admin.HostData
}

func (c *AdminMonitorController) GetIndex(ctx *gin.Context) {
	hosts, err := c.monitorSrv.GetHosts()
	if err != nil {
		panic(err)
	}
	ctx.HTML(http.StatusOK, "index.html", AGetIndexViewModel{hosts})
}

type AGetHostRes struct {
	Host          string              `json:"host"`
	Count         int                 `json:"count"`
	BytesReceived int64               `json:"bytes_received"`
	BytesSent     int64               `json:"bytes_sent"`
	Clients       []*admin.ClientData `json:"clients"`
}

func (c *AdminMonitorController) GetHost(ctx *gin.Context) {
	host := ctx.Param("host")
	if host == "" {
		panic("host is nil")
	}

	clients, count, bytesReceived, bytesSent, err := c.monitorSrv.GetStatus(host, true)
	if err != nil {
		panic(err)
	}

	json := ctx.Query("json")

	data := AGetHostRes{
		Host:          host,
		Count:         count,
		BytesReceived: bytesReceived,
		BytesSent:     bytesSent,
		Clients:       clients,
	}

	if json == "1" {
		ctx.JSON(http.StatusOK, data)
	} else {
		ctx.HTML(http.StatusOK, "host.html", data)
	}
}
