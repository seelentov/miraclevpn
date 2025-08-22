// Package logg provides logging utilities for the application.
package logg

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ZapLogger struct {
	Logger   *zap.Logger
	logDir   string
	retain   int
	appLog   *os.File
	dayTimer *time.Timer
}

func NewZapLogger(logDir string, retainDays int, debug bool) (*ZapLogger, error) {
	var mainSync, appSync zapcore.WriteSyncer
	var appLogFile *os.File

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	consoleSync := zapcore.Lock(os.Stdout)

	var logLevel zapcore.Level
	if debug {
		logLevel = zapcore.DebugLevel
	} else {
		logLevel = zapcore.InfoLevel
	}

	cores := []zapcore.Core{
		zapcore.NewCore(zapcore.NewConsoleEncoder(encoderCfg), consoleSync, logLevel),
	}

	if logDir != "" {
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, err
		}

		mainLogPath := filepath.Join(logDir, "log.log")
		mainLogFile, err := os.OpenFile(mainLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		mainSync = zapcore.AddSync(mainLogFile)
		cores = append(cores, zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), mainSync, logLevel))

		appLogPath := filepath.Join(logDir, "app.log")
		appLogFile, err = os.OpenFile(appLogPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return nil, err
		}
		appSync = zapcore.AddSync(appLogFile)
		cores = append(cores, zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), appSync, logLevel))
	}

	core := zapcore.NewTee(cores...)

	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	zl := &ZapLogger{
		Logger: logger,
		logDir: logDir,
		retain: retainDays,
		appLog: appLogFile,
	}

	if logDir != "" {
		zl.startDailyRotation()
	}

	return zl, nil
}

func (zl *ZapLogger) startDailyRotation() {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	duration := next.Sub(now)
	zl.dayTimer = time.AfterFunc(duration, zl.rotateDaily)
}

func (zl *ZapLogger) rotateDaily() {
	zl.appLog.Close()
	dateStr := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
	dstPath := filepath.Join(zl.logDir, fmt.Sprintf("%s.log", dateStr))
	os.Rename(filepath.Join(zl.logDir, "app.log"), dstPath)
	newAppLog, _ := os.OpenFile(filepath.Join(zl.logDir, "app.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	zl.appLog = newAppLog

	files, _ := os.ReadDir(zl.logDir)
	cutoff := time.Now().AddDate(0, 0, -zl.retain)
	for _, f := range files {
		if f.Name() == "log.log" || f.Name() == "app.log" {
			continue
		}
		if t, err := time.Parse("2006-01-02.log", f.Name()); err == nil && t.Before(cutoff) {
			os.Remove(filepath.Join(zl.logDir, f.Name()))
		}
	}

	zl.startDailyRotation()
}

func (zl *ZapLogger) Sync() error {
	return zl.Logger.Sync()
}
