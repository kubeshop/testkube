package testkube

type RunningContextType string

const (
	RunningContextTypeUserCLI               RunningContextType = "user-cli"
	RunningContextTypeUserUI                RunningContextType = "user-ui"
	RunningContextTypeTestSuite             RunningContextType = "testsuite"
	RunningContextTypeTestWorkflow          RunningContextType = "testworkflow"
	RunningContextTypeTestTrigger           RunningContextType = "testtrigger"
	RunningContextTypeScheduler             RunningContextType = "scheduler"
	RunningContextTypeTestExecution         RunningContextType = "testexecution"
	RunningContextTypeTestSuiteExecution    RunningContextType = "testsuiteexecution"
	RunningContextTypeTestWorkflowExecution RunningContextType = "testworkflowexecution"
	RunningContextTypeEmpty                 RunningContextType = ""
)
