package log

import (
	"log"
	"os"
	"strconv"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func IsTrue(name string) (is bool) {
	var err error
	if val, ok := os.LookupEnv(name); ok {
		is, err = strconv.ParseBool(val)
		if err != nil {
			return false
		}
	}

	return is
}

// New returns new logger instance
func New() *zap.SugaredLogger {
	atomicLevel := zap.NewAtomicLevel()

	atomicLevel.SetLevel(zap.InfoLevel)
	if IsTrue("DEBUG") {
		atomicLevel.SetLevel(zap.DebugLevel)
	}

	zapCfg := zap.NewProductionConfig()
	zapCfg.Level = atomicLevel
	zapCfg.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder

	z, err := zapCfg.Build()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	logger := z.Sugar()
	return logger
}

func NewSilent() *zap.SugaredLogger {
	atomicLevel := zap.NewAtomicLevel()

	atomicLevel.SetLevel(zap.WarnLevel)
	if IsTrue("DEBUG") {
		atomicLevel.SetLevel(zap.DebugLevel)
	}

	zapCfg := zap.NewProductionConfig()
	zapCfg.Level = atomicLevel
	zapCfg.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder

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
