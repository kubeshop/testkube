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
	CodeAborted    uint8 = 137
	CodeInputError uint8 = 155
	CodeInternal   uint8 = 190
)
