package log

import (
	"log"

	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/envs"
)

// New returns new logger instance
func New() *zap.SugaredLogger {
	atomicLevel := zap.NewAtomicLevel()

	atomicLevel.SetLevel(zap.InfoLevel)
	if envs.IsTrue("DEBUG") {
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
