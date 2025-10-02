package v1

// RunningContext for test or test suite execution
type RunningContext struct {
	// One of possible context types
	Type_ RunningContextType `json:"type"`
	// Context value depending from its type
	Context string `json:"context,omitempty"`
}

type RunningContextType string

const (
	RunningContextTypeUserCLI     RunningContextType = "user-cli"
	RunningContextTypeUserUI      RunningContextType = "user-ui"
	RunningContextTypeTestSuite   RunningContextType = "testsuite"
	RunningContextTypeTestTrigger RunningContextType = "testtrigger"
	RunningContextTypeScheduler   RunningContextType = "scheduler"
	RunningContextTypeEmpty       RunningContextType = ""
)
