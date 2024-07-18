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

func (s StepStatus) Code() string {
	return string(s)[0:1]
}

func StepStatusFromCode(code string) StepStatus {
	switch code {
	case string(StepStatusPassed)[0:1]:
		return StepStatusPassed
	case string(StepStatusTimeout)[0:1]:
		return StepStatusTimeout
	case string(StepStatusFailed)[0:1]:
		return StepStatusFailed
	case string(StepStatusAborted)[0:1]:
		return StepStatusAborted
	case string(StepStatusSkipped)[0:1]:
		return StepStatusSkipped
	}
	return StepStatusAborted
}

const (
	CodeTimeout    uint8 = 124
	CodeAborted    uint8 = 137
	CodeInputError uint8 = 155
	CodeNoCommand  uint8 = 189
	CodeInternal   uint8 = 190
)
