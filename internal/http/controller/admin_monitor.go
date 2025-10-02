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

type AGetHostViewModel struct {
	Host          string
	Count         int
	BytesReceived int64
	BytesSent     int64
	Clients       []*admin.ClientData
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

	ctx.HTML(http.StatusOK, "host.html", AGetHostViewModel{
		Host:          host,
		Count:         count,
		BytesReceived: bytesReceived,
		BytesSent:     bytesSent,
		Clients:       clients,
	})

}
