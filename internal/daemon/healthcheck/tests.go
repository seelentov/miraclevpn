package healthcheck

import (
	"miraclevpn/internal/services/sender"
	"os/exec"
	"time"

	"go.uber.org/zap"
)

type TestsHealthCheck struct {
	duration time.Duration
	sender   sender.Sender
	adminTo  string

	logger *zap.Logger

	stopChan chan struct{}
}

func NewTestsHealthCheck(duration time.Duration, logger *zap.Logger, sender sender.Sender, adminTo string) *TestsHealthCheck {
	return &TestsHealthCheck{
		duration: duration,
		logger:   logger,
		stopChan: make(chan struct{}),
		sender:   sender,
		adminTo:  adminTo,
	}
}

func (d *TestsHealthCheck) Start() {
	ticker := time.NewTicker(d.duration)

	d.logger.Info("Starting tests health check",
		zap.Duration("interval", d.duration))

	go func() {
		for {
			select {
			case <-ticker.C:
				if err := d.do(); err != nil {
					d.sender.SendMessage(d.adminTo, "Tests health check failed: "+err.Error())
					d.logger.Error("Tests health check failed", zap.Error(err))
				} else {
					d.logger.Debug("Tests health check passed")
				}
			case <-d.stopChan:
				d.logger.Info("Stopping tests health check")
				return
			}
		}
	}()
}

func (d *TestsHealthCheck) Stop() {
	close(d.stopChan)
}

func (d *TestsHealthCheck) do() error {
	cmd := exec.Command("go", "test", "./...")

	_, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}

	return nil
}
