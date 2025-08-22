package healthcheck

import (
	"fmt"
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
				if err := d.do(); err != nil {
					d.sender.SendMessage(d.adminTo, fmt.Sprintf("TG health check failed: %v", err))
					d.logger.Error("TG health check failed", zap.Error(err))
				} else {
					d.logger.Debug("TG health check passed")
				}
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

func (d *TgHealthCheck) do() error {
	status, err := d.sender.GetStatus()
	if err != nil {
		return err
	}

	if !status {
		return fmt.Errorf("TG %s is not healthy", d.sender.GetName())
	}

	return nil
}
