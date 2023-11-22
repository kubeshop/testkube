package result

import "github.com/kubeshop/testkube/pkg/cloud/data/executor"

const (
	CmdResultGetNextExecutionNumber executor.Command = "result_get_next_execution_number"
	CmdResultGet                    executor.Command = "result_get"
	CmdResultGetByNameAndTest       executor.Command = "result_get_by_name_and_test"
	CmdResultGetLatestByTest        executor.Command = "result_get_latest_by_test"
	CmdResultGetLatestByTests       executor.Command = "result_get_latest_by_tests"
	CmdResultGetExecutions          executor.Command = "result_get_executions"
	CmdResultGetExecutionTotals     executor.Command = "result_get_execution_totals"
	CmdResultInsert                 executor.Command = "result_insert"
	CmdResultUpdate                 executor.Command = "result_update"
	CmdResultUpdateResult           executor.Command = "result_update_result"
	CmdResultStartExecution         executor.Command = "result_start_execution"
	CmdResultEndExecution           executor.Command = "result_end_execution"
	CmdResultGetLabels              executor.Command = "result_get_labels"
	CmdResultDeleteByTest           executor.Command = "result_delete_by_test"
	CmdResultDeleteByTestSuite      executor.Command = "result_delete_by_test_suite"
	CmdResultDeleteAll              executor.Command = "result_delete_all"
	CmdResultDeleteByTests          executor.Command = "result_delete_by_tests"
	CmdResultDeleteByTestSuites     executor.Command = "result_delete_by_test_suites"
	CmdResultDeleteForAllTestSuites executor.Command = "result_delete_for_all_test_suites"
	CmdResultGetTestMetrics         executor.Command = "result_get_test_metrics"
)
