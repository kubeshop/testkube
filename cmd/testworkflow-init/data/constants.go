package data

import "path/filepath"

const (
	InitStepName       = "tktw-init"
	InternalPath       = "/.tktw"
	TerminationLogPath = "/dev/termination-log"
)

var (
	InternalBinPath = filepath.Join(InternalPath, "bin")
	InitPath        = filepath.Join(InternalPath, "init")
	StatePath       = filepath.Join(InternalPath, "state")
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
	if len(code) != 1 {
		return StepStatusAborted
	}
	switch code[0] {
	case StepStatusPassed[0]:
		return StepStatusPassed
	case StepStatusTimeout[0]:
		return StepStatusTimeout
	case StepStatusFailed[0]:
		return StepStatusFailed
	case StepStatusAborted[0]:
		return StepStatusAborted
	case StepStatusSkipped[0]:
		return StepStatusSkipped
	}
	return StepStatusAborted
}

const (
	CodeAborted    uint8 = 137
	CodeInputError uint8 = 155
	CodeInternal   uint8 = 190
)
