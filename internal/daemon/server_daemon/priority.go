package serverdaemon

import (
	"fmt"
	"miraclevpn/internal/repo"
	"miraclevpn/internal/services/sender"
	"miraclevpn/internal/services/vpn"
	"miraclevpn/internal/utils"
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
					er := utils.GetStackTrace(err)
					err := d.sender.SendMessage(d.adminTo, fmt.Sprintf("failed: %v", er))
					if err != nil {
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

	srvsClients := make([]*ServerClients, 0)

	for _, srv := range srvs {
		status, err := d.vpnClient.GetStatus(srv.Host)
		if err != nil {
			srvsClients = append(srvsClients, &ServerClients{
				id:      srv.ID,
				clients: len(status.Clients),
			})
		}
	}

	sort.Slice(srvsClients, func(i, j int) bool {
		return srvsClients[i].clients > srvsClients[j].clients
	})

	for i, srv := range srvsClients {
		if err := d.srvRepo.UpdatePriority(srv.id, i*1000); err != nil {
			return err
		}
	}

	return nil
}
