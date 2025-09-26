package healthcheck

import (
	"fmt"
	"miraclevpn/internal/repo"
	"miraclevpn/internal/services/sender"
	"miraclevpn/internal/services/vpn"
	"sync"
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
				d.do()
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

func (d *VpnHealthCheck) do() {
	srvs, err := d.srvRepo.FindAll()
	if err != nil {
		d.logger.Error("VPN health check failed", zap.Error(err))
		return
	}

	if len(srvs) == 0 {
		d.logger.Info("VPN health check passed - servers is nil")
	}

	wg := sync.WaitGroup{}

	for _, srv := range srvs {
		wg.Add(1)

		go func() {
			defer wg.Done()

			status, err := d.vpnClient.GetStatus(srv.Host)
			if err != nil {
				if err := d.sender.SendMessage(d.adminTo, fmt.Sprintf("VPN health check %s failed: %s", srv.Host, err)); err != nil {
					d.logger.Error("ADMIN TG SEND FAILED", zap.Error(err))
				}
				d.logger.Error("VPN health check failed", zap.String("host", srv.Host), zap.Error(err))
				return
			}

			tenPersentUsers := srv.MaxUsers - (srv.MaxUsers / 10)

			clients := len(status.Clients)
			if len(status.Clients) > tenPersentUsers {
				if err := d.sender.SendMessage(d.adminTo, fmt.Sprintf("VPN HIGHLOAD! on %s: %d/%d", srv.Host, clients, srv.MaxUsers)); err != nil {
					d.logger.Error("ADMIN TG SEND FAILED", zap.Error(err))
				}
				d.logger.Warn(fmt.Sprintf("VPN HIGHLOAD! on %s: %d/%d", srv.Host, clients, srv.MaxUsers))
				return
			}

			d.logger.Info(srv.Host+" health check - OK", zap.Error(err))
		}()
	}

	wg.Wait()
}
