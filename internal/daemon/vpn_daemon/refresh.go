package vpn_daemon

import (
	"fmt"
	"miraclevpn/internal/services/sender"
	"miraclevpn/internal/services/servers"
	"miraclevpn/internal/services/vpn"
	"miraclevpn/internal/utils"
	"time"

	"go.uber.org/zap"
)

type VpnRefreshDaemon struct {
	vpnClient vpn.VpnService
	srvSrv    *servers.ServersService

	sender  sender.Sender
	adminTo string

	duration time.Duration

	logger *zap.Logger

	stopChan chan struct{}

	expiration time.Duration
}

func NewVpnRefreshDaemon(duration time.Duration, logger *zap.Logger, vpnClient vpn.VpnService, srvSrv *servers.ServersService, sender sender.Sender, adminTo string, expiration time.Duration) *VpnRefreshDaemon {
	return &VpnRefreshDaemon{
		vpnClient:  vpnClient,
		srvSrv:     srvSrv,
		duration:   duration,
		logger:     logger,
		sender:     sender,
		adminTo:    adminTo,
		stopChan:   make(chan struct{}),
		expiration: expiration,
	}
}

func (d *VpnRefreshDaemon) Start() {
	ticker := time.NewTicker(d.duration)

	d.logger.Info("Starting VPN refresh daemon",
		zap.Duration("interval", d.duration))

	go func() {
		for {
			select {
			case <-ticker.C:
				if err := d.do(); err != nil {
					er := utils.GetStackTrace(err)
					d.sender.SendMessage(d.adminTo, fmt.Sprintf("VPN refresh daemon failed: %v", er))
					d.logger.Error("VPN refresh daemon failed", zap.String("error", er))
				} else {
					d.logger.Debug("VPN refresh daemon passed")
				}
			case <-d.stopChan:
				d.logger.Info("Stopping VPN refresh daemon")
				return
			}
		}
	}()
}

func (d *VpnRefreshDaemon) Stop() {
	close(d.stopChan)
}

func (d *VpnRefreshDaemon) do() error {
	if err := d.srvSrv.UpdateExpired(d.expiration); err != nil {
		return err
	}

	return nil
}
