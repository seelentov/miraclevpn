// Package authdaemon provides auth daemons
package authdaemon

import (
	"encoding/json"
	"fmt"
	"miraclevpn/internal/models"
	"miraclevpn/internal/repo"
	"miraclevpn/internal/services/sender"
	"miraclevpn/internal/utils"
	"time"

	"go.uber.org/zap"
)

type AuthFindSuspicious struct {
	authRepo *repo.AuthDataRepository

	sender  sender.Sender
	adminTo string

	duration time.Duration

	logger *zap.Logger

	stopChan chan struct{}
}

func NewAuthFindSuspicious(duration time.Duration, logger *zap.Logger, authRepo *repo.AuthDataRepository, sender sender.Sender, adminTo string) *AuthFindSuspicious {
	return &AuthFindSuspicious{
		authRepo: authRepo,
		duration: duration,
		logger:   logger,
		sender:   sender,
		adminTo:  adminTo,
		stopChan: make(chan struct{}),
	}
}

func (d *AuthFindSuspicious) Start() {
	ticker := time.NewTicker(d.duration)

	d.logger.Info("Starting Auth find suspicios",
		zap.Duration("interval", d.duration))

	go func() {
		for {
			select {
			case <-ticker.C:
				auths, err := d.do()
				if err != nil {
					er := utils.GetStackTrace(err)
					err := d.sender.SendMessage(d.adminTo, fmt.Sprintf("Auth find suspicios failed: %v", er))
					if err != nil {
						d.logger.Error("ADMIN TG SEND FAILED", zap.Error(err))
					}
					d.logger.Error("Auth find suspicios failed", zap.String("error", er))
				} else if len(auths) > 0 {
					d.logger.Info("FIND SUSPICIOUS AUTHS", zap.Any("list", auths))
					authsS, _ := json.Marshal(auths)

					err := d.sender.SendMessage(d.adminTo, fmt.Sprintf("FIND SUSPICIOUS AUTHS: %s", authsS))
					if err != nil {
						d.logger.Error("ADMIN TG SEND FAILED", zap.Error(err))
					}
				}
			case <-d.stopChan:
				d.logger.Info("Stopping Auth find suspicios")
				return
			}
		}
	}()
}

func (d *AuthFindSuspicious) Stop() {
	close(d.stopChan)
}

func (d *AuthFindSuspicious) do() ([]*models.AuthData, error) {
	return d.authRepo.FindSuspicious(int(d.duration.Hours()))
}
