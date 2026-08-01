// Package executiondata exposes data produced by other TestWorkflow executions
// to the expression language, so that workflows composed into a suite can
// exchange values and files.
//
// A parent workflow records every child it schedules through `execute.workflows`
// in a Registry; the `execution()` function resolves references against it, and
// falls back to the control plane for references the registry does not know
// (a raw execution id, or the special "parent" reference).
package executiondata

import "github.com/kubeshop/testkube/pkg/api/v1/testkube"

const (
	// ParentRef is the reserved reference pointing at the execution that scheduled
	// the current one.
	ParentRef = "parent"

	// OutputsInstructionName is the name of the output instruction a step emits to
	// publish the values it left in the outputs directory. It makes them part of the
	// execution record, so a parent workflow can read them back with execution().
	OutputsInstructionName = "outputs"

	// ExecutionInstructionPrefix prefixes the output instruction the parent emits for
	// each child it schedules. The suffix is the child's alias, so that every child
	// gets its own entry instead of overwriting the previous one.
	ExecutionInstructionPrefix = "testworkflow-execution."
)

// Execution is the data a workflow may read about another execution.
type Execution struct {
	// Id is the execution id.
	Id string `json:"id"`
	// Name is the execution name.
	Name string `json:"name"`
	// Workflow is the name of the TestWorkflow that was executed.
	Workflow string `json:"workflow"`
	// Alias is the `as` value the parent gave this entry, empty when not aliased.
	Alias string `json:"alias,omitempty"`
	// Index is the position within a fan-out (matrix/shards/count), 0 when single.
	Index int64 `json:"index"`
	// Status is the final execution status.
	Status string `json:"status,omitempty"`
	// Outputs are the values the execution published in its outputs directory.
	Outputs map[string]string `json:"outputs,omitempty"`
}

// Key is the primary reference of the execution - its alias when the parent gave
// one, otherwise the name of the workflow it ran. Executions sharing a key form a
// single fan-out group addressed by index.
func (e Execution) Key() string {
	if e.Alias != "" {
		return e.Alias
	}
	return e.Workflow
}

// Refs are the names this execution may be addressed by, most specific first.
func (e Execution) Refs() []string {
	refs := make([]string, 0, 3)
	if e.Alias != "" {
		refs = append(refs, e.Alias)
	}
	if e.Workflow != "" {
		refs = append(refs, e.Workflow)
	}
	if e.Id != "" {
		refs = append(refs, e.Id)
	}
	return refs
}

// AsMap converts the execution into the shape the expression language sees.
func (e Execution) AsMap() map[string]interface{} {
	outputs := make(map[string]interface{}, len(e.Outputs))
	for k, v := range e.Outputs {
		outputs[k] = v
	}
	return map[string]interface{}{
		"id":       e.Id,
		"name":     e.Name,
		"workflow": e.Workflow,
		"alias":    e.Alias,
		"index":    e.Index,
		"status":   e.Status,
		"outputs":  outputs,
	}
}

// FromExecution converts a full execution record into the data workflows may read.
func FromExecution(execution *testkube.TestWorkflowExecution) Execution {
	if execution == nil {
		return Execution{}
	}
	result := Execution{
		Id:      execution.Id,
		Name:    execution.Name,
		Outputs: OutputsOf(execution),
	}
	if execution.Workflow != nil {
		result.Workflow = execution.Workflow.Name
	}
	if execution.Result != nil && execution.Result.Status != nil {
		result.Status = string(*execution.Result.Status)
	}
	return result
}

// OutputsOf collects the values an execution published through its steps.
//
// Every step emits a single instruction carrying all the files it left in the
// outputs directory; they are flattened into one map here. When two steps use
// the same key, the later step wins - outputs are ordered as they were appended
// to the execution record.
func OutputsOf(execution *testkube.TestWorkflowExecution) map[string]string {
	if execution == nil {
		return nil
	}
	values := make(map[string]string)
	for _, output := range execution.Output {
		if output.Name != OutputsInstructionName {
			continue
		}
		for key, value := range output.Value {
			if str, ok := value.(string); ok {
				values[key] = str
			}
		}
	}
	return values
}
