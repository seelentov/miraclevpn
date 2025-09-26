// Package vpndaemon provides vpn daemons for the application.
package vpndaemon

import (
	"fmt"
	"miraclevpn/internal/services/sender"
	"miraclevpn/internal/services/servers"
	"time"

	"go.uber.org/zap"
)

type VpnRefreshDaemon struct {
	srvSrv *servers.ServersService

	sender  sender.Sender
	adminTo string

	duration time.Duration

	logger *zap.Logger

	stopChan chan struct{}

	expiration time.Duration
}

func NewVpnRefreshDaemon(duration time.Duration, logger *zap.Logger, srvSrv *servers.ServersService, sender sender.Sender, adminTo string, expiration time.Duration) *VpnRefreshDaemon {
	return &VpnRefreshDaemon{
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
					if err := d.sender.SendMessage(d.adminTo, fmt.Sprintf("VPN refresh daemon failed: %v", err)); err != nil {
						d.logger.Error("ADMIN TG SEND FAILED", zap.Error(err))
					}
					d.logger.Error("VPN refresh daemon failed", zap.Error(err))
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
	if err := d.srvSrv.UpdateOnline(); err != nil {
		return err
	}

	if err := d.srvSrv.UpdateExpired(d.expiration); err != nil {
		return err
	}

	return nil
}
