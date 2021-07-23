package kubetest

func NewScriptExecution(id string, name string, execution Execution) ScriptExecution {
	return ScriptExecution{
		Id:        id,
		Name:      name,
		Execution: &execution,
	}
}
