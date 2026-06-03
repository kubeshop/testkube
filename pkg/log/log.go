package log

import (
	"log"
	"os"
	"strconv"
	"strings"

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

// levelFromEnv reads the optional LOG_LEVEL environment variable and parses it
// into a zapcore.Level. Accepted values follow zap semantics: debug, info,
// warn (the issue's "Warning"), error, dpanic, panic, fatal. zap has no Trace
// level, so debug is the finest granularity. If LOG_LEVEL is unset or empty the
// provided defaultLevel is returned. On parse error a non-fatal warning is
// written to stderr and defaultLevel is returned.
func levelFromEnv(defaultLevel zapcore.Level) zapcore.Level {
	val, ok := os.LookupEnv("LOG_LEVEL")
	if !ok || val == "" {
		return defaultLevel
	}

	lvl, err := zapcore.ParseLevel(strings.ToLower(val))
	if err != nil {
		log.Printf("invalid LOG_LEVEL %q: %v; falling back to %s", val, err, defaultLevel)
		return defaultLevel
	}

	return lvl
}

// New returns new logger instance
func New() *zap.SugaredLogger {
	atomicLevel := zap.NewAtomicLevel()

	atomicLevel.SetLevel(levelFromEnv(zap.InfoLevel))
	if IsTrue("DEBUG") {
		atomicLevel.SetLevel(zap.DebugLevel)
	}

	zapCfg := zap.NewProductionConfig()
	if loggerJsonStr := os.Getenv("LOGGER_JSON"); loggerJsonStr == "true" {
		zapCfg = zap.NewDevelopmentConfig()
	}

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

	atomicLevel.SetLevel(levelFromEnv(zap.WarnLevel))
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
