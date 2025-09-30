// Package paymentdaemon provides payment daemons for the application.
package paymentdaemon

import (
	"fmt"
	"miraclevpn/internal/services/payment"
	"miraclevpn/internal/services/sender"
	"time"

	"go.uber.org/zap"
)

type AutoPaymentDaemon struct {
	paySrv *payment.AutoPaymentService

	sender  sender.Sender
	adminTo string

	duration time.Duration

	logger *zap.Logger

	stopChan chan struct{}
}

func NewAutoPaymentDaemon(duration time.Duration, logger *zap.Logger, paySrv *payment.AutoPaymentService, sender sender.Sender, adminTo string) *AutoPaymentDaemon {
	return &AutoPaymentDaemon{
		paySrv:   paySrv,
		duration: duration,
		logger:   logger,
		sender:   sender,
		adminTo:  adminTo,
		stopChan: make(chan struct{}),
	}
}

func (d *AutoPaymentDaemon) Start() {
	ticker := time.NewTicker(d.duration)

	d.logger.Info("Starting auto-payment daemon",
		zap.Duration("interval", d.duration))

	go func() {
		for {
			select {
			case <-ticker.C:
				d.do()
			case <-d.stopChan:
				d.logger.Info("Stopping auto-payment daemon")
				return
			}
		}
	}()
}

func (d *AutoPaymentDaemon) Stop() {
	close(d.stopChan)
}

func (d *AutoPaymentDaemon) do() {
	ups, err := d.paySrv.FindForAutoPayment()
	if err != nil {
		d.processErr(err)
	}

	for _, up := range ups {
		if err := d.paySrv.Process(
			up.ID,
			*up.Email,
			*up.PaymentID,
			true,
		); err != nil {
			d.processErr(err)
			continue
		}

		d.logger.Info("Auto-payment for", zap.Int("user_id", len(up.ID)))
	}

	d.logger.Info("Auto-payment daemon end iteration", zap.Int("ups", len(ups)))
}

func (d *AutoPaymentDaemon) processErr(err error) {
	if err := d.sender.SendMessage(d.adminTo, fmt.Sprintf("Auto-payment daemon failed: %v", err)); err != nil {
		d.logger.Error("ADMIN TG SEND FAILED", zap.Error(err))
	}
	d.logger.Error("Auto-payment daemon failed", zap.Error(err))
}
