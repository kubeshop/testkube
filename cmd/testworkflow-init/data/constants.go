package data

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

type StepStatus string

const (
	StepStatusPassed  StepStatus = "passed"
	StepStatusTimeout StepStatus = "timeout"
	StepStatusFailed  StepStatus = "failed"
	StepStatusAborted StepStatus = "aborted"
	StepStatusSkipped StepStatus = "skipped"
)

const (
	CodeTimeout    uint8 = 124
	CodeAborted    uint8 = 137
	CodeInputError uint8 = 155
	CodeNoCommand  uint8 = 189
	CodeInternal   uint8 = 190
)
