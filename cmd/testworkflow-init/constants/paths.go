package constants

import (
	"os"
	"path/filepath"
)

const (
	defaultInternalPath       = "/.tktw"
	defaultTerminationLogPath = "/dev/termination-log"
)

var (
	InternalPath       = getEnvOrDefault("TESTKUBE_TW_INTERNAL_PATH", defaultInternalPath)
	TerminationLogPath = getEnvOrDefault("TESTKUBE_TW_TERMINATION_LOG_PATH", defaultTerminationLogPath)
	InternalBinPath    = filepath.Join(InternalPath, "bin")
	InitPath           = filepath.Join(InternalPath, "init")
	ToolkitPath        = filepath.Join(InternalPath, "toolkit")
	StatePath          = getEnvOrDefault("TESTKUBE_TW_STATE_PATH", filepath.Join(InternalPath, "state"))
	// StepErrorPath is the file where a toolkit step writes the cause of its failure.
	StepErrorPath = getEnvOrDefault("TESTKUBE_TW_STEP_ERROR_PATH", filepath.Join(InternalPath, "error"))
)

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
