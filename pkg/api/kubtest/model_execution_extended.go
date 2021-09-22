package kubtest

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func NewExecutionWithID(id, scriptType, scriptName string) Execution {
	return Execution{
		Id:         id,
		Result:     &Result{},
		ScriptName: scriptName,
		ScriptType: scriptType,
	}
}

func NewScriptExecution(scriptName, name, scriptType string, execution Result, params map[string]string) Execution {
	return Execution{
		Id:         primitive.NewObjectID().Hex(),
		Name:       name,
		ScriptName: scriptName,
		Result:     &execution,
		ScriptType: scriptType,
		Params:     params,
	}
}

type ScriptExecutions []Execution

func (executions ScriptExecutions) Table() (header []string, output [][]string) {
	header = []string{"Script", "Type", "Name", "ID", "Status"}

	for _, e := range executions {
		status := "unknown"
		if e.Result != nil {
			status = e.Result.Status
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

func (e *Execution) WithContent(content string) *Execution {
	e.ScriptContent = content
	return e
}

func (e *Execution) WithRepository(repository *Repository) *Execution {
	e.Repository = repository
	return e
}

func (e *Execution) WithParams(params map[string]string) *Execution {
	e.Params = params
	return e
}

func (e *Execution) WithRepositoryData(uri, branch, path string) *Execution {
	e.Repository = &Repository{
		Uri:    uri,
		Branch: branch,
		Path:   path,
	}
	return e
}
