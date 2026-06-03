package log

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestLevelFromEnv_Unset(t *testing.T) {
	// t.Setenv to empty exercises the unset/empty short-circuit and restores
	// any pre-existing value after the test.
	t.Setenv("LOG_LEVEL", "")

	if got := levelFromEnv(zap.InfoLevel); got != zapcore.InfoLevel {
		t.Errorf("levelFromEnv(InfoLevel) with empty LOG_LEVEL = %v, want %v", got, zapcore.InfoLevel)
	}
	if got := levelFromEnv(zap.WarnLevel); got != zapcore.WarnLevel {
		t.Errorf("levelFromEnv(WarnLevel) with empty LOG_LEVEL = %v, want %v", got, zapcore.WarnLevel)
	}
}

func TestLevelFromEnv_Error(t *testing.T) {
	t.Setenv("LOG_LEVEL", "error")

	if got := levelFromEnv(zap.InfoLevel); got != zapcore.ErrorLevel {
		t.Errorf("levelFromEnv(InfoLevel) with LOG_LEVEL=error = %v, want %v", got, zapcore.ErrorLevel)
	}
}

func TestLevelFromEnv_Debug(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")

	if got := levelFromEnv(zap.InfoLevel); got != zapcore.DebugLevel {
		t.Errorf("levelFromEnv(InfoLevel) with LOG_LEVEL=debug = %v, want %v", got, zapcore.DebugLevel)
	}
}

func TestLevelFromEnv_Garbage(t *testing.T) {
	t.Setenv("LOG_LEVEL", "garbage")

	if got := levelFromEnv(zap.InfoLevel); got != zapcore.InfoLevel {
		t.Errorf("levelFromEnv(InfoLevel) with LOG_LEVEL=garbage = %v, want %v (fallback)", got, zapcore.InfoLevel)
	}
	if got := levelFromEnv(zap.WarnLevel); got != zapcore.WarnLevel {
		t.Errorf("levelFromEnv(WarnLevel) with LOG_LEVEL=garbage = %v, want %v (fallback)", got, zapcore.WarnLevel)
	}
}

func TestNew_LogLevelError(t *testing.T) {
	t.Setenv("DEBUG", "")
	t.Setenv("LOG_LEVEL", "error")

	core := New().Desugar().Core()
	if !core.Enabled(zapcore.ErrorLevel) {
		t.Error("New() with LOG_LEVEL=error: expected ErrorLevel enabled")
	}
	if core.Enabled(zapcore.InfoLevel) {
		t.Error("New() with LOG_LEVEL=error: expected InfoLevel disabled")
	}
}

func TestNew_DebugOverridesLogLevel(t *testing.T) {
	// DEBUG must keep forcing Debug regardless of LOG_LEVEL (backward compat).
	t.Setenv("LOG_LEVEL", "error")
	t.Setenv("DEBUG", "true")

	core := New().Desugar().Core()
	if !core.Enabled(zapcore.DebugLevel) {
		t.Error("New() with DEBUG=true should enable DebugLevel regardless of LOG_LEVEL")
	}
}
