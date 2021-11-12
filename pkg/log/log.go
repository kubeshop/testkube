package log

import (
	"log"
	"os"

	"go.uber.org/zap"
)

// TODO introduce interface for zap
// New returns new logger instance
func New() *zap.SugaredLogger {
	atomicLevel := zap.NewAtomicLevel()

	atomicLevel.SetLevel(zap.InfoLevel)
	if val, exists := os.LookupEnv("DEBUG"); exists && val != "" {
		atomicLevel.SetLevel(zap.DebugLevel)
	}

	zapCfg := zap.NewProductionConfig()
	zapCfg.Level = atomicLevel

	z, err := zapCfg.Build()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	logger := z.Sugar()
	return logger
}

// DefaultLogger initialized default logger
var DefaultLogger *zap.SugaredLogger

func init() {
	DefaultLogger = New()
}

// TODO decide if we should introduce some abstraction/facade over ZAP to simplify future refactor to other logging library?
// It could be functions? like Info("message", fields...), Error("message", fields...)
// or some interface on top of ZAP?
