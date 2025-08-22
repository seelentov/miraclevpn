// Package healthcheck provides health check services for the application.
package healthcheck

import (
	"context"
	"database/sql"
	"fmt"
	"miraclevpn/internal/services/sender"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type DBHealthCheck struct {
	duration time.Duration
	db       *gorm.DB
	sender   sender.Sender
	adminTo  string

	logger *zap.Logger

	stopChan chan struct{}
}

func NewDBHealthCheck(db *gorm.DB, duration time.Duration, logger *zap.Logger, sender sender.Sender, adminTo string) *DBHealthCheck {
	return &DBHealthCheck{
		db:       db,
		duration: duration,
		logger:   logger,
		stopChan: make(chan struct{}),
		sender:   sender,
		adminTo:  adminTo,
	}
}

func (d *DBHealthCheck) Start() {
	sqlDB, err := d.db.DB()
	if err != nil {
		d.logger.Error("Failed to get database connection", zap.Error(err))
		return
	}

	ticker := time.NewTicker(d.duration)

	d.logger.Info("Starting database health check",
		zap.Duration("interval", d.duration))

	go func() {
		for {
			select {
			case <-ticker.C:
				if err := d.do(sqlDB); err != nil {
					d.sender.SendMessage(d.adminTo, fmt.Sprintf("Database health check failed: %v", err))
					d.logger.Error("Database health check failed", zap.Error(err))
				} else {
					d.logger.Debug("Database health check passed")
				}
			case <-d.stopChan:
				d.logger.Info("Stopping database health check")
				return
			}
		}
	}()
}

func (d *DBHealthCheck) Stop() {
	close(d.stopChan)
}

func (d *DBHealthCheck) do(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := db.PingContext(ctx)
	if err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Дополнительные проверки (опционально)
	if err := d.checkConnectionStats(db); err != nil {
		return err
	}

	return nil
}

func (d *DBHealthCheck) checkConnectionStats(db *sql.DB) error {
	stats := db.Stats()

	if stats.MaxOpenConnections > 0 && stats.OpenConnections >= stats.MaxOpenConnections {
		return fmt.Errorf("max open connections reached: %d/%d",
			stats.OpenConnections, stats.MaxOpenConnections)
	}

	d.logger.Debug("Database connection stats",
		zap.Int("open_connections", stats.OpenConnections),
		zap.Int("max_open_connections", stats.MaxOpenConnections),
		zap.Int("in_use", stats.InUse),
		zap.Int("idle", stats.Idle),
		zap.Int64("wait_count", stats.WaitCount),
	)

	return nil
}
