// Package serverdaemon provides server daemons
package serverdaemon

import (
	"fmt"
	"miraclevpn/internal/repo"
	"miraclevpn/internal/services/sender"
	"miraclevpn/internal/services/vpn"
	"sort"
	"time"

	"go.uber.org/zap"
)

type ServerClients struct {
	id      int64
	clients int
}

type ServerAutoPriority struct {
	vpnClient vpn.VpnService
	srvRepo   *repo.ServerRepository

	sender  sender.Sender
	adminTo string

	duration time.Duration

	logger *zap.Logger

	stopChan chan struct{}
}

func NewServerAutoPriority(duration time.Duration, logger *zap.Logger, vpnClient vpn.VpnService, srvRepo *repo.ServerRepository, sender sender.Sender, adminTo string) *ServerAutoPriority {
	return &ServerAutoPriority{
		vpnClient: vpnClient,
		srvRepo:   srvRepo,
		duration:  duration,
		logger:    logger,
		sender:    sender,
		adminTo:   adminTo,
		stopChan:  make(chan struct{}),
	}
}

func (d *ServerAutoPriority) Start() {
	ticker := time.NewTicker(d.duration)

	d.logger.Info("Starting",
		zap.Duration("interval", d.duration))

	go func() {
		for {
			select {
			case <-ticker.C:
				if err := d.do(); err != nil {
					if err := d.sender.SendMessage(d.adminTo, fmt.Sprintf("failed: %v", err)); err != nil {
						d.logger.Error("ADMIN TG SEND FAILED", zap.Error(err))
					}
					d.logger.Error("failed", zap.Error(err))
				}
			case <-d.stopChan:
				d.logger.Info("Stopping")
				return
			}
		}
	}()
}

func (d *ServerAutoPriority) Stop() {
	close(d.stopChan)
}

func (d *ServerAutoPriority) do() error {
	srvs, err := d.srvRepo.FindAll()
	if err != nil {
		return err
	}

	if len(srvs) == 0 {
		d.logger.Info("No servers found")
		return nil
	}

	srvsClients := make([]*ServerClients, 0, len(srvs))

	for _, srv := range srvs {
		status, err := d.vpnClient.GetStatus(srv.Host)
		if err != nil {
			d.logger.Warn("Failed to get server status",
				zap.String("host", srv.Host),
				zap.Error(err))
			// Пропускаем сервер с ошибкой
			continue
		}

		srvsClients = append(srvsClients, &ServerClients{
			id:      srv.ID,
			clients: len(status.Clients),
		})
	}

	if len(srvsClients) == 0 {
		d.logger.Info("No servers with valid status found")
		return nil
	}

	sort.Slice(srvsClients, func(i, j int) bool {
		return srvsClients[i].clients > srvsClients[j].clients
	})

	d.logger.Info("Updating server priorities",
		zap.Int("servers_count", len(srvsClients)))

	for i, srv := range srvsClients {
		if err := d.srvRepo.UpdatePriority(srv.id, i*1000); err != nil {
			d.logger.Error("Failed to update priority",
				zap.Int64("server_id", srv.id),
				zap.Error(err))
			return err
		}
		d.logger.Debug("Priority updated",
			zap.Int64("server_id", srv.id),
			zap.Int("priority", i*1000),
			zap.Int("clients", srv.clients))
	}

	d.logger.Info("Server priorities updated successfully",
		zap.Int("servers_processed", len(srvsClients)))

	return nil
}
