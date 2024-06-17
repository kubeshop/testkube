package testworkflow

import "github.com/kubeshop/testkube/pkg/cloud/data/executor"

const (
	CmdTestWorkflowExecutionGet                  executor.Command = "workflow_execution_get"
	CmdTestWorkflowExecutionGetByNameAndWorkflow executor.Command = "workflow_execution_get_by_name_and_workflow"
	CmdTestWorkflowExecutionGetLatestByWorkflow  executor.Command = "workflow_execution_get_latest_by_workflow"
	CmdTestWorkflowExecutionGetRunning           executor.Command = "workflow_execution_get_running"
	CmdTestWorkflowExecutionGetLatestByWorkflows executor.Command = "workflow_execution_get_latest_by_workflows"
	CmdTestWorkflowExecutionGetExecutionTotals   executor.Command = "workflow_execution_get_execution_totals"
	CmdTestWorkflowExecutionGetExecutions        executor.Command = "workflow_execution_get_executions"
	CmdTestWorkflowExecutionGetExecutionsSummary executor.Command = "workflow_execution_get_executions_summary"
	CmdTestWorkflowExecutionInsert               executor.Command = "workflow_execution_insert"
	CmdTestWorkflowExecutionUpdate               executor.Command = "workflow_execution_update"
	CmdTestWorkflowExecutionUpdateResult         executor.Command = "workflow_execution_update_result"
	CmdTestWorkflowExecutionAddReport            executor.Command = "workflow_execution_add_report"
	CmdTestWorkflowExecutionUpdateOutput         executor.Command = "workflow_execution_update_output"
	CmdTestWorkflowExecutionDeleteByWorkflow     executor.Command = "workflow_execution_delete_by_workflow"
	CmdTestWorkflowExecutionDeleteAll            executor.Command = "workflow_execution_delete_all"
	CmdTestWorkflowExecutionDeleteByWorkflows    executor.Command = "workflow_execution_delete_by_workflows"
	CmdTestWorkflowExecutionGetWorkflowMetrics   executor.Command = "workflow_execution_get_workflow_metrics"

	CmdTestWorkflowOutputPresignSaveLog         executor.Command = "workflow_output_presign_save_log"
	CmdTestWorkflowOutputPresignReadLog         executor.Command = "workflow_output_presign_read_log"
	CmdTestWorkflowOutputHasLog                 executor.Command = "workflow_output_has_log"
	CmdTestWorkflowOutputDeleteByTestWorkflow   executor.Command = "workflow_output_delete_by_test_workflow"
	CmdTestworkflowOutputDeleteForTestWorkflows executor.Command = "workflow_output_delete_for_test_workflows"
)

func command(v interface{}) executor.Command {
	switch v.(type) {
	case ExecutionGetRequest:
		return CmdTestWorkflowExecutionGet
	case ExecutionGetByNameAndWorkflowRequest:
		return CmdTestWorkflowExecutionGetByNameAndWorkflow
	case ExecutionGetLatestByWorkflowRequest:
		return CmdTestWorkflowExecutionGetLatestByWorkflow
	case ExecutionGetRunningRequest:
		return CmdTestWorkflowExecutionGetRunning
	case ExecutionGetLatestByWorkflowsRequest:
		return CmdTestWorkflowExecutionGetLatestByWorkflows
	case ExecutionGetExecutionTotalsRequest:
		return CmdTestWorkflowExecutionGetExecutionTotals
	case ExecutionGetExecutionsRequest:
		return CmdTestWorkflowExecutionGetExecutions
	case ExecutionGetExecutionsSummaryRequest:
		return CmdTestWorkflowExecutionGetExecutionsSummary
	case ExecutionInsertRequest:
		return CmdTestWorkflowExecutionInsert
	case ExecutionUpdateRequest:
		return CmdTestWorkflowExecutionUpdate
	case ExecutionUpdateResultRequest:
		return CmdTestWorkflowExecutionUpdateResult
	case ExecutionUpdateOutputRequest:
		return CmdTestWorkflowExecutionUpdateOutput
	case ExecutionDeleteByWorkflowRequest:
		return CmdTestWorkflowExecutionDeleteByWorkflow
	case ExecutionDeleteAllRequest:
		return CmdTestWorkflowExecutionDeleteAll
	case ExecutionDeleteByWorkflowsRequest:
		return CmdTestWorkflowExecutionDeleteByWorkflows
	case ExecutionGetWorkflowMetricsRequest:
		return CmdTestWorkflowExecutionGetWorkflowMetrics

	case OutputPresignSaveLogRequest:
		return CmdTestWorkflowOutputPresignSaveLog
	case OutputPresignReadLogRequest:
		return CmdTestWorkflowOutputPresignReadLog
	case OutputHasLogRequest:
		return CmdTestWorkflowOutputHasLog
	case ExecutionDeleteOutputByWorkflowRequest:
		return CmdTestWorkflowOutputDeleteByTestWorkflow
	case ExecutionDeleteOutputForTestWorkflowsRequest:
		return CmdTestworkflowOutputDeleteForTestWorkflows
	}
	panic("unknown test workflows Cloud request")
}
