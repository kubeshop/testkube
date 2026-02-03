package v1

func IsWorkflowSilent(workflow *TestWorkflow) bool {
	if workflow == nil || workflow.Spec.Execution == nil || workflow.Spec.Execution.Silent == nil {
		return false
	}
	return *workflow.Spec.Execution.Silent
}

