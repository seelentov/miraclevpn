// Package vpndaemon provides vpn daemons for the application.
package vpndaemon

import (
	"fmt"
	"miraclevpn/internal/models"
	"miraclevpn/internal/services/sender"
	"miraclevpn/internal/services/servers"
	"miraclevpn/internal/services/vpn"
	"sync"
	"time"

	"go.uber.org/zap"
)

type KickHighloadDaemon struct {
	srvSrv        *servers.ServersService
	vpnClient     vpn.VpnService
	highLoadBytes int64

	sender  sender.Sender
	adminTo string

	duration time.Duration

	logger *zap.Logger

	stopChan chan struct{}

	expiration time.Duration
}

func NewKickHighloadDaemon(duration time.Duration, logger *zap.Logger, srvSrv *servers.ServersService, vpnClient vpn.VpnService, sender sender.Sender, adminTo string, expiration time.Duration, highLoadBytes int64) *KickHighloadDaemon {
	return &KickHighloadDaemon{
		srvSrv:        srvSrv,
		vpnClient:     vpnClient,
		duration:      duration,
		logger:        logger,
		sender:        sender,
		adminTo:       adminTo,
		stopChan:      make(chan struct{}),
		expiration:    expiration,
		highLoadBytes: highLoadBytes,
	}
}

func (d *KickHighloadDaemon) Start() {
	ticker := time.NewTicker(d.duration)

	d.logger.Info("Starting VPN kick highload",
		zap.Duration("interval", d.duration))

	go func() {
		for {
			select {
			case <-ticker.C:
				if err := d.do(); err != nil {
					if err := d.sender.SendMessage(d.adminTo, fmt.Sprintf("VPN kick highload failed: %v", err)); err != nil {
						d.logger.Error("ADMIN TG SEND FAILED", zap.Error(err))
					}
					d.logger.Error("VPN kick highload failed", zap.Error(err))
				}
			case <-d.stopChan:
				d.logger.Info("Stopping VPN kick highload")
				return
			}
		}
	}()
}

func (d *KickHighloadDaemon) Stop() {
	close(d.stopChan)
}

func (d *KickHighloadDaemon) do() error {
	srvs, err := d.srvSrv.GetAllServers()
	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}

	for _, srv := range srvs {
		wg.Add(1)
		go func(srv *models.Server) {
			defer wg.Done()

			trafics, err := d.vpnClient.GetAllRate(srv.Host, int(d.duration))
			if err != nil {
				d.logger.Error("failed get trafic", zap.String("host", srv.Host))
				if err := d.sender.SendMessage(d.adminTo, fmt.Sprintf("VPN kick highload: failed get trafic %s: %v", srv.Host, err)); err != nil {
					d.logger.Error("ADMIN TG SEND FAILED", zap.Error(err))
				}
				return
			}

			wg := sync.WaitGroup{}

			for _, trafic := range trafics {
				wg.Add(1)
				go func(trafic *vpn.TraficStatus) {
					defer wg.Done()

					if trafic.BytesReceived > d.highLoadBytes || trafic.BytesSend > d.highLoadBytes {
						if err := d.vpnClient.KickUser(srv.Host, trafic.ClientName); err != nil {
							d.logger.Error("failed kick", zap.String("host", srv.Host), zap.String("client", trafic.ClientName))
							if err := d.sender.SendMessage(d.adminTo, fmt.Sprintf("VPN kick highload: failed kick %s - %s: %v", srv.Host, trafic.ClientName, err)); err != nil {
								d.logger.Error("ADMIN TG SEND FAILED", zap.Error(err))
							}
						}
					}
				}(trafic)
			}
			wg.Wait()

		}(srv)
	}
	wg.Wait()

	return nil
}
