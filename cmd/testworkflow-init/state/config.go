package state

import "path/filepath"

const (
	InitStepName       = "tktw-init"
	InternalPath       = "/.tktw"
	TerminationLogPath = "/dev/termination-log"
)

var (
	InternalBinPath = filepath.Join(InternalPath, "bin")
	ShellPath       = filepath.Join(InternalBinPath, "sh")
	InitPath        = filepath.Join(InternalPath, "init")
	StatePath       = filepath.Join(InternalPath, "state")
	TransferDirPath = filepath.Join(InternalPath, "transfer")
	TmpDirPath      = filepath.Join(InternalPath, "tmp")
)
