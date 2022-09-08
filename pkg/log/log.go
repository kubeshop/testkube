package log

import (
	"fmt"
	"log"

	"github.com/kubeshop/testkube/pkg/envs"
	"go.uber.org/zap"
)

// New returns new logger instance
func New() *zap.SugaredLogger {
	atomicLevel := zap.NewAtomicLevel()

	atomicLevel.SetLevel(zap.InfoLevel)
	if envs.IsTrue("DEBUG") {
		fmt.Printf("%+v\n", "DEBUG=1")

		atomicLevel.SetLevel(zap.DebugLevel)
	} else {
		fmt.Printf("%+v\n", "DEBUG=1")

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
