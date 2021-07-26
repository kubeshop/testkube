package kubetest

import "go.mongodb.org/mongo-driver/bson/primitive"

func NewScriptExecution(scriptName, name string, execution Execution) ScriptExecution {
	return ScriptExecution{
		Id:         primitive.NewObjectID().Hex(),
		Name:       name,
		ScriptName: scriptName,
		Execution:  &execution,
		ScriptType: "postman/collection", // TODO need to be passed from CRD type
	}
}

type ScriptExecutions []ScriptExecution

func (executions ScriptExecutions) ToArray() (header []string, output [][]string) {
	header = []string{"Script", "Type", "Execution ID", "Execution Name", "Status"}
	for _, e := range executions {
		output = append(output, []string{
			e.ScriptName,
			e.ScriptType,
			e.Id,
			e.Name,
			e.Execution.Status,
		})
	}

	return
}
