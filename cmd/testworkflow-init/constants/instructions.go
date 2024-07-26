package constants

const (
	InstructionStart     = "start"
	InstructionEnd       = "end"
	InstructionExecution = "execution"
	InstructionPause     = "pause"
	InstructionResume    = "resume"
	InstructionIteration = "iteration"
)

type ExecutionResult struct {
	ExitCode  uint8  `json:"code"`
	Details   string `json:"details,omitempty"`
	Iteration int    `json:"iteration,omitempty"`
}
