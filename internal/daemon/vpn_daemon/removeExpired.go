package vpndaemon

import (
	"fmt"
	"miraclevpn/internal/services/sender"
	"miraclevpn/internal/services/servers"
	"time"

	"go.uber.org/zap"
)

type VpnRemoveExpiredDaemon struct {
	srvSrv *servers.ServersService

	sender  sender.Sender
	adminTo string

	duration time.Duration

	logger *zap.Logger

	stopChan chan struct{}
}

func NewVpnRemoveExpiredDaemon(duration time.Duration, logger *zap.Logger, srvSrv *servers.ServersService, sender sender.Sender, adminTo string) *VpnRemoveExpiredDaemon {
	return &VpnRemoveExpiredDaemon{
		srvSrv:   srvSrv,
		duration: duration,
		logger:   logger,
		sender:   sender,
		adminTo:  adminTo,
		stopChan: make(chan struct{}),
	}
}

func (d *VpnRemoveExpiredDaemon) Start() {
	ticker := time.NewTicker(d.duration)

	d.logger.Info("Starting VPN remove expired daemon",
		zap.Duration("interval", d.duration))

	go func() {
		for {
			select {
			case <-ticker.C:
				if err := d.do(); err != nil {
					if err := d.sender.SendMessage(d.adminTo, fmt.Sprintf("VPN remove expired daemon failed: %v", err)); err != nil {
						d.logger.Error("ADMIN TG SEND FAILED", zap.Error(err))
					}
					d.logger.Error("VPN remove expired daemon failed", zap.Error(err))
				}
			case <-d.stopChan:
				d.logger.Info("Stopping VPN remove expired daemon")
				return
			}
		}
	}()
}

func (d *VpnRemoveExpiredDaemon) Stop() {
	close(d.stopChan)
}

func (d *VpnRemoveExpiredDaemon) do() error {
	return d.srvSrv.RemoveExpiredByUser()
}
