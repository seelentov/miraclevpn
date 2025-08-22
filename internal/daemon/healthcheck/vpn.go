package healthcheck

import (
	"fmt"
	"miraclevpn/internal/repo"
	"miraclevpn/internal/services/sender"
	"miraclevpn/internal/services/vpn"
	"time"

	"go.uber.org/zap"
)

type VpnHealthCheck struct {
	vpnClient vpn.VpnService
	srvRepo   *repo.ServerRepository

	sender  sender.Sender
	adminTo string

	duration time.Duration

	logger *zap.Logger

	stopChan chan struct{}
}

func NewVpnHealthCheck(duration time.Duration, logger *zap.Logger, vpnClient vpn.VpnService, srvRepo *repo.ServerRepository, sender sender.Sender, adminTo string) *VpnHealthCheck {
	return &VpnHealthCheck{
		vpnClient: vpnClient,
		srvRepo:   srvRepo,
		duration:  duration,
		logger:    logger,
		sender:    sender,
		adminTo:   adminTo,
		stopChan:  make(chan struct{}),
	}
}

func (d *VpnHealthCheck) Start() {
	ticker := time.NewTicker(d.duration)

	d.logger.Info("Starting VPN health check",
		zap.Duration("interval", d.duration))

	go func() {
		for {
			select {
			case <-ticker.C:
				if err := d.do(); err != nil {
					d.sender.SendMessage(d.adminTo, fmt.Sprintf("VPN health check failed: %v", err))
					d.logger.Error("VPN health check failed", zap.Error(err))
				} else {
					d.logger.Debug("VPN health check passed")
				}
			case <-d.stopChan:
				d.logger.Info("Stopping VPN health check")
				return
			}
		}
	}()
}

func (d *VpnHealthCheck) Stop() {
	close(d.stopChan)
}

func (d *VpnHealthCheck) do() error {
	srvs, err := d.srvRepo.FindAll()
	if err != nil {
		return err
	}

	for _, srv := range srvs {
		status, err := d.vpnClient.GetStatus(srv.Host)
		if err != nil {
			return err
		}

		if !status.Online {
			return fmt.Errorf("server %s is offline. err: %w", srv.Host, err)
		}
	}

	return nil
}
