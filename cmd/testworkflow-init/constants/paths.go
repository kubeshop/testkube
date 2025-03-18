package constants

import "path/filepath"

const (
	InternalPath       = "/.tktw"
	TerminationLogPath = "/dev/termination-log"
)

var (
	InternalBinPath = filepath.Join(InternalPath, "bin")
	InitPath        = filepath.Join(InternalPath, "init")
	ToolkitPath     = filepath.Join(InternalPath, "toolkit")
	StatePath       = filepath.Join(InternalPath, "state")
)
