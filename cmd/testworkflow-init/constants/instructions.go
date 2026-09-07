package constants

const (
	InstructionStart     = "start"
	InstructionEnd       = "end"
	InstructionExecution = "execution"
	InstructionPause     = "pause"
	InstructionResume    = "resume"
	InstructionIteration = "iteration"

	// InstructionTestResults carries what a step's test report said, when the
	// step declared a testCases policy. Its value is a
	// testkube.TestWorkflowStepTestResults.
	//
	// The counters travel separately from the human summary in ExecutionResult,
	// because they are two different audiences: the summary is a log line a
	// person reads, this is the structured record a dashboard filters on. A
	// control plane that does not know the name ignores it, so an older one
	// keeps working.
	InstructionTestResults = "testResults"
)

type ExecutionResult struct {
	ExitCode  uint8  `json:"code"`
	Details   string `json:"details,omitempty"`
	Iteration int    `json:"iteration,omitempty"`
}
