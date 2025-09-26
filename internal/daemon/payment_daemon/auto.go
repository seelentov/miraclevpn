// Package paymentdaemon provides payment daemons for the application.
package paymentdaemon

import (
	"fmt"
	"miraclevpn/internal/services/payment"
	"miraclevpn/internal/services/sender"
	"miraclevpn/internal/utils"
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

	d.logger.Info("Starting VPN refresh daemon",
		zap.Duration("interval", d.duration))

	go func() {
		for {
			select {
			case <-ticker.C:
				d.do()
			case <-d.stopChan:
				d.logger.Info("Stopping VPN refresh daemon")
				return
			}
		}
	}()
}

func (d *AutoPaymentDaemon) Stop() {
	close(d.stopChan)
}

func (d *AutoPaymentDaemon) do() {
	ups, err := d.paySrv.FindForPayment()
	if err != nil {
		d.processErr(err)
	}

	for _, up := range ups {
		if err := d.paySrv.Process(
			up.User.ID,
			*up.User.Email,
			*up.User.PaymentID,
			up.Plan,
			false,
		); err != nil {
			d.processErr(err)
		}
	}
}

func (d *AutoPaymentDaemon) processErr(err error) {
	er := utils.GetStackTrace(err)
	d.sender.SendMessage(d.adminTo, fmt.Sprintf("VPN refresh daemon failed: %v", er))
	d.logger.Error("VPN refresh daemon failed", zap.String("error", er))
}
