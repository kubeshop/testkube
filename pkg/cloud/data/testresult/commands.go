package testresult

import "github.com/kubeshop/testkube/pkg/cloud/data/executor"

const (
	CmdTestResultGet                   executor.Command = "test_result_get"
	CmdTestResultGetByNameAndTestSuite executor.Command = "test_result_get_by_name_and_test"
	CmdTestResultGetLatestByTestSuite  executor.Command = "test_result_get_latest_by_test_suite"
	CmdTestResultGetLatestByTestSuites executor.Command = "test_result_get_latest_by_test_suites"
	CmdTestResultGetExecutionsTotals   executor.Command = "test_result_get_executions_totals"
	CmdTestResultGetExecutions         executor.Command = "test_result_get_executions"
	CmdTestResultInsert                executor.Command = "test_result_insert"
	CmdTestResultUpdate                executor.Command = "test_result_update"
	CmdTestResultStartExecution        executor.Command = "test_result_start_execution"
	CmdTestResultEndExecution          executor.Command = "test_result_end_execution"
	CmdTestResultDeleteByTestSuite     executor.Command = "test_result_delete_by_test_suite"
	CmdTestResultDeleteAll             executor.Command = "test_result_delete_all"
	CmdTestResultDeleteByTestSuites    executor.Command = "test_result_delete_by_test_suites"
	CmdTestResultGetTestSuiteMetrics   executor.Command = "test_result_get_test_suite_metrics"
)
