package paymentdaemon

import (
	"fmt"
	"miraclevpn/internal/repo"
	"miraclevpn/internal/services/sender"
	"miraclevpn/internal/utils"
	"time"

	"go.uber.org/zap"
)

type ServerClients struct {
	id      int64
	clients int
}

type PaymentRemoveExpired struct {
	payRepo *repo.PaymentRepository

	sender  sender.Sender
	adminTo string

	duration time.Duration

	logger *zap.Logger

	stopChan chan struct{}
}

func NewPaymentRemoveExpired(duration time.Duration, logger *zap.Logger, payRepo *repo.PaymentRepository, sender sender.Sender, adminTo string) *PaymentRemoveExpired {
	return &PaymentRemoveExpired{
		payRepo:  payRepo,
		duration: duration,
		logger:   logger,
		sender:   sender,
		adminTo:  adminTo,
		stopChan: make(chan struct{}),
	}
}

func (d *PaymentRemoveExpired) Start() {
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

func (d *PaymentRemoveExpired) Stop() {
	close(d.stopChan)
}

func (d *PaymentRemoveExpired) do() error {
	return d.payRepo.DeleteExpired()
}
