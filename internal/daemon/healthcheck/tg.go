package healthcheck

import (
	"miraclevpn/internal/services/sender"
	"time"

	"go.uber.org/zap"
)

type TgHealthCheck struct {
	sender  sender.Sender
	adminTo string

	duration time.Duration

	logger *zap.Logger

	stopChan chan struct{}
}

func NewTgHealthCheck(duration time.Duration, logger *zap.Logger, sender sender.Sender, adminTo string) *TgHealthCheck {
	return &TgHealthCheck{
		duration: duration,
		logger:   logger,
		sender:   sender,
		adminTo:  adminTo,
		stopChan: make(chan struct{}),
	}
}

func (d *TgHealthCheck) Start() {
	ticker := time.NewTicker(d.duration)

	d.logger.Info("Starting TG health check",
		zap.Duration("interval", d.duration))

	go func() {
		for {
			select {
			case <-ticker.C:
				d.do()
			case <-d.stopChan:
				d.logger.Info("Stopping TG health check")
				return
			}
		}
	}()
}

func (d *TgHealthCheck) Stop() {
	close(d.stopChan)
}

func (d *TgHealthCheck) do() {
	if _, err := d.sender.GetStatus(); err != nil {
		d.logger.Error("ADMIN TG STATUS CHECK FAILED", zap.Error(err))
	}
}
