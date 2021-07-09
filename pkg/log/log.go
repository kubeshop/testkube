package log

import (
	"go.uber.org/zap"
)

// TODO introduce interface for zap
// New returns new logger instance
func New() *zap.SugaredLogger {
	logger, _ := zap.NewProduction()
	return logger.Sugar()
}

// DefaultLogger initialized default logger
var DefaultLogger *zap.SugaredLogger

func init() {
	DefaultLogger = New()
}

// TODO decide if we should introduce some abstraction/facade over ZAP to simplify future refactor to other logging library?
// It could be functions? like Info("message", fields...), Error("message", fields...)
// or some interface on top of ZAP?
