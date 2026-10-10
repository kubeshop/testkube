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
	ExitCode uint8  `json:"code"`
	Details  string `json:"details,omitempty"`
	// Reason is the code of the cause, empty when the init process has no code for it. A code with
	// empty Details means that the code alone says what happened.
	Reason    string `json:"reason,omitempty"`
	Iteration int    `json:"iteration,omitempty"`
}
