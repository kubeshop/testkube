package kubtest

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func NewScriptExecution(scriptName, name, scriptType string, execution Execution, params map[string]string) ScriptExecution {
	return ScriptExecution{
		Id:         primitive.NewObjectID().Hex(),
		Name:       name,
		ScriptName: scriptName,
		Execution:  &execution,
		ScriptType: scriptType,
		Params:     params,
	}
}

type ScriptExecutions []ScriptExecution

func (executions ScriptExecutions) Table() (header []string, output [][]string) {
	header = []string{"Script", "Type", "Name", "ID", "Status"}

	for _, e := range executions {
		status := "unknown"
		if e.Execution != nil {
			status = e.Execution.Status
		}

		output = append(output, []string{
			e.ScriptName,
			e.ScriptType,
			e.Name,
			e.Id,
			status,
		})
	}

	return
}
