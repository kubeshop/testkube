package constants

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
