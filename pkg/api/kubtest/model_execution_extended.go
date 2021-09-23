package kubtest

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func NewExecutionWithID(id, scriptType, scriptName string) Execution {
	return Execution{
		Id:              id,
		ExecutionResult: &ExecutionResult{},
		ScriptName:      scriptName,
		ScriptType:      scriptType,
	}
}

func NewExecution(scriptName, name, scriptType string, result ExecutionResult, params map[string]string) Execution {
	return Execution{
		Id:              primitive.NewObjectID().Hex(),
		ScriptName:      scriptName,
		Name:            name,
		ScriptType:      scriptType,
		ExecutionResult: &result,
		Params:          params,
	}
}

type Executions []Execution

func (executions Executions) Table() (header []string, output [][]string) {
	header = []string{"Script", "Type", "Name", "ID", "Status"}

	for _, e := range executions {
		status := "unknown"
		if e.ExecutionResult != nil {
			status = e.ExecutionResult.Status
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

func (e Execution) Err(err error) Execution {
	e.ExecutionResult.Err(err)
	return e
}
