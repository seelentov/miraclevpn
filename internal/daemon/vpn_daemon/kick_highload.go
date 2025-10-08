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

	duration       time.Duration
	checkStatusSec int

	logger *zap.Logger

	stopChan chan struct{}

	expiration time.Duration
}

func NewKickHighloadDaemon(duration time.Duration, logger *zap.Logger, srvSrv *servers.ServersService, vpnClient vpn.VpnService, sender sender.Sender, adminTo string, expiration time.Duration, highLoadBytes int64, checkStatusSec int) *KickHighloadDaemon {
	return &KickHighloadDaemon{
		srvSrv:         srvSrv,
		vpnClient:      vpnClient,
		duration:       duration,
		logger:         logger,
		sender:         sender,
		adminTo:        adminTo,
		stopChan:       make(chan struct{}),
		expiration:     expiration,
		highLoadBytes:  highLoadBytes,
		checkStatusSec: checkStatusSec,
	}
}

func (d *KickHighloadDaemon) Start() {
	ticker := time.NewTicker(d.duration)

	d.logger.Info("Starting VPN kick highload daemon",
		zap.Duration("interval", d.duration),
		zap.Int64("high_load_threshold_bytes", d.highLoadBytes),
		zap.Int("check_status_seconds", d.checkStatusSec))

	go func() {
		for {
			select {
			case <-ticker.C:
				d.logger.Info("Starting new highload check cycle")
				if err := d.do(); err != nil {
					if err := d.sender.SendMessage(d.adminTo, fmt.Sprintf("VPN kick highload failed: %v", err)); err != nil {
						d.logger.Error("ADMIN TG SEND FAILED", zap.Error(err))
					}
					d.logger.Error("VPN kick highload failed", zap.Error(err))
				} else {
					d.logger.Info("Highload check cycle completed successfully")
				}
			case <-d.stopChan:
				d.logger.Info("Stopping VPN kick highload daemon")
				ticker.Stop()
				return
			}
		}
	}()
}

func (d *KickHighloadDaemon) Stop() {
	d.logger.Info("Stop signal received for VPN kick highload daemon")
	close(d.stopChan)
}

func (d *KickHighloadDaemon) do() error {
	d.logger.Info("Retrieving all servers for highload monitoring")
	srvs, err := d.srvSrv.GetAllServers()
	if err != nil {
		return err
	}

	d.logger.Info("Servers retrieved successfully",
		zap.Int("server_count", len(srvs)),
		zap.Any("servers", getServerHosts(srvs)))

	wg := sync.WaitGroup{}
	kickedUsersCount := 0
	checkedUsersCount := 0

	for _, srv := range srvs {
		wg.Add(1)
		go func(srv *models.Server) {
			defer wg.Done()

			d.logger.Info("Checking server for highload users",
				zap.String("server_host", srv.Host))

			trafics, err := d.vpnClient.GetAllRate(srv.Host, d.checkStatusSec)
			if err != nil {
				d.logger.Error("failed get trafic", zap.String("host", srv.Host))
				if err := d.sender.SendMessage(d.adminTo, fmt.Sprintf("VPN kick highload: failed get trafic %s: %v", srv.Host, err)); err != nil {
					d.logger.Error("ADMIN TG SEND FAILED", zap.Error(err))
				}
				return
			}

			d.logger.Info("Traffic data retrieved for server",
				zap.String("server_host", srv.Host),
				zap.Int("user_count", len(trafics)))

			wgs := sync.WaitGroup{}
			serverKickedCount := 0

			for _, trafic := range trafics {
				wgs.Add(1)
				go func(trafic *vpn.TraficStatus) {
					defer wgs.Done()
					checkedUsersCount++

					d.logger.Info("Checking user traffic",
						zap.String("server_host", srv.Host),
						zap.String("client_name", trafic.ClientName),
						zap.Int64("bytes_received", trafic.BytesReceived),
						zap.Int64("bytes_sent", trafic.BytesSend),
						zap.Int64("threshold_bytes", d.highLoadBytes))

					if trafic.BytesReceived > d.highLoadBytes || trafic.BytesSend > d.highLoadBytes {
						d.logger.Info("Highload user detected, initiating kick",
							zap.String("server_host", srv.Host),
							zap.String("client_name", trafic.ClientName),
							zap.Int64("bytes_received", trafic.BytesReceived),
							zap.Int64("bytes_sent", trafic.BytesSend),
							zap.Int64("threshold_bytes", d.highLoadBytes))

						if err := d.vpnClient.KickUser(srv.Host, trafic.ClientName); err != nil {
							d.logger.Error("failed kick", zap.String("host", srv.Host), zap.String("client", trafic.ClientName))
							if err := d.sender.SendMessage(d.adminTo, fmt.Sprintf("VPN kick highload: failed kick %s - %s: %v", srv.Host, trafic.ClientName, err)); err != nil {
								d.logger.Error("ADMIN TG SEND FAILED", zap.Error(err))
							}
						} else {
							d.logger.Info("User kicked successfully",
								zap.String("user", trafic.ClientName),
								zap.String("host", srv.Host),
								zap.Int64("BytesReceived", trafic.BytesReceived),
								zap.Int64("BytesSend", trafic.BytesSend))
							kickedUsersCount++
							serverKickedCount++
						}
					} else {
						d.logger.Debug("User traffic within normal limits",
							zap.String("server_host", srv.Host),
							zap.String("client_name", trafic.ClientName))
					}
				}(trafic)
			}
			wgs.Wait()

			if serverKickedCount > 0 {
				d.logger.Info("Server highload check completed",
					zap.String("server_host", srv.Host),
					zap.Int("users_kicked", serverKickedCount),
					zap.Int("total_users_checked", len(trafics)))
			} else {
				d.logger.Info("Server highload check completed - no users kicked",
					zap.String("server_host", srv.Host),
					zap.Int("total_users_checked", len(trafics)))
			}

		}(srv)
	}
	wg.Wait()

	d.logger.Info("Highload monitoring cycle summary",
		zap.Int("total_servers_checked", len(srvs)),
		zap.Int("total_users_checked", checkedUsersCount),
		zap.Int("total_users_kicked", kickedUsersCount))

	return nil
}

// Вспомогательная функция для получения списка хостов серверов
func getServerHosts(servers []*models.Server) []string {
	hosts := make([]string, len(servers))
	for i, srv := range servers {
		hosts[i] = srv.Host
	}
	return hosts
}
