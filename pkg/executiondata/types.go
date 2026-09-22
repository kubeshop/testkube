// Package executiondata exposes data produced by other TestWorkflow executions
// to the expression language, so that workflows composed into a suite can
// exchange values and files.
//
// A parent workflow records every child it schedules through `execute.workflows`
// in a Registry; the `execution()` function resolves references against it, and
// falls back to the control plane for references the registry does not know
// (a raw execution id, or the special "parent" reference).
//
// # What outputs can carry
//
// Outputs cross the boundary between executions through the execution record, and
// they reach it by being printed to the log stream, which is obfuscated on its way
// out. That fixes the contract: an output whose value holds a sensitive word cannot
// be exchanged between executions. Such a value stays inside the workflow that
// produced it, where later steps still read it in full through
// `step.<id>.outputs.<key>`; what leaves is a marker naming what was withheld, and a
// consumer resolving that marker fails rather than acting on a value that isn't
// there. This is the intended contract, not a temporary limitation - publishing the
// value would either corrupt it (the obfuscator masks part of it) or leak it (the
// execution record is readable by everyone who can read the execution).
//
// A value that must genuinely cross executions has to travel outside the log stream:
// store it as an artifact and read it with `read_artifact()`, or hand it to both
// workflows as a secret.
package executiondata

import "github.com/kubeshop/testkube/pkg/api/v1/testkube"

const (
	// ParentRef is the reserved reference pointing at the execution that scheduled
	// the current one.
	ParentRef = "parent"

	// RerunRef is the reserved reference pointing at the execution this one is a
	// rerun of - the one whose results a rerun draws from.
	//
	// It is a reference rather than an id the workflow has to be handed, because
	// the workflow is written once and rerun many times: the author says "the
	// execution I am a rerun of" and the scheduler decides which that is.
	RerunRef = "rerun"

	// OutputsInstructionName is the name of the output instruction a step emits to
	// publish the values it left in the outputs directory. It makes them part of the
	// execution record, so a parent workflow can read them back with execution().
	//
	// The instruction is printed to the obfuscated log stream, so it carries only the
	// values that may leave the workflow: a value holding a sensitive word is replaced
	// by a WithheldMarker. See the package documentation for the contract.
	OutputsInstructionName = "outputs"

	// ExecutionInstructionPrefix prefixes the output instruction the parent emits for
	// each child it schedules. The suffix is the child's alias, so that every child
	// gets its own entry instead of overwriting the previous one.
	//
	// Deliberately not under "testworkflow-": the dashboard groups instruction names by
	// the pattern ^testworkflow(-.*)?$ and reads them as the status of a single child,
	// so a name in that family would be read as one - this one carries a list, and every
	// field the dashboard looked for would be missing.
	ExecutionInstructionPrefix = "executiondata."
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
	//
	// An output the execution produced but could not publish - its value holds a
	// sensitive word - is present here as a WithheldMarker rather than as its value.
	Outputs map[string]string `json:"outputs,omitempty"`
	// ErrorMessage is the message of the initialization step, empty when it has none.
	ErrorMessage string `json:"errorMessage,omitempty"`
	// StepErrors are the messages of the steps that have one, keyed by step ref.
	StepErrors map[string]string `json:"stepErrors,omitempty"`
	// StepAttempts are the numbers of attempts of the steps that report one, keyed by step ref.
	StepAttempts map[string]int64 `json:"stepAttempts,omitempty"`
	// ErrorReason is the reason code of the initialization step, empty when it has none.
	ErrorReason string `json:"errorReason,omitempty"`
	// StepReasons are the reason codes of the steps that have one, keyed by step ref.
	StepReasons map[string]string `json:"stepReasons,omitempty"`
	// StatusType is the layer that made the execution fail, empty when it passed.
	StatusType string `json:"statusType,omitempty"`
	// StatusReason is the code of the cause that made the execution fail.
	StatusReason string `json:"statusReason,omitempty"`
	// StatusStep is the ref of the step that holds the cause, empty when no step holds it.
	StatusStep string `json:"statusStep,omitempty"`
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
	return map[string]interface{}{
		"id":           e.Id,
		"name":         e.Name,
		"workflow":     e.Workflow,
		"alias":        e.Alias,
		"index":        e.Index,
		"status":       e.Status,
		"outputs":      toInterfaceMap(e.Outputs),
		"errorMessage": e.ErrorMessage,
		"stepErrors":   toInterfaceMap(e.StepErrors),
		"stepAttempts": toInterfaceMap(e.StepAttempts),
		"errorReason":  e.ErrorReason,
		"stepReasons":  toInterfaceMap(e.StepReasons),
		"statusType":   e.StatusType,
		"statusReason": e.StatusReason,
		"statusStep":   e.StatusStep,
	}
}

// toInterfaceMap copies the map into the shape the expression language reads.
// It returns an empty map for a nil map.
func toInterfaceMap[V any](values map[string]V) map[string]interface{} {
	result := make(map[string]interface{}, len(values))
	for k, v := range values {
		result[k] = v
	}
	return result
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
	result.ErrorMessage, result.StepErrors = ErrorsOf(execution)
	result.StepAttempts = AttemptsOf(execution)
	result.ErrorReason, result.StepReasons = ReasonsOf(execution)
	result.SetStatusDetails(execution)
	return result
}

// SetStatusDetails copies the codes of the status details of the execution into the record. Every
// caller that builds a record from a finished execution needs it, because a workflow reads the codes
// of a child through the record and not through the stored execution.
//
// The message and the user of the object stay out of the record. A workflow asserts the codes, and
// the words are already available through ErrorMessage and StepErrors.
func (e *Execution) SetStatusDetails(execution *testkube.TestWorkflowExecution) {
	if execution == nil || execution.Result == nil || execution.Result.StatusDetails == nil {
		return
	}
	e.StatusType = execution.Result.StatusDetails.Type_
	e.StatusReason = execution.Result.StatusDetails.Reason
	e.StatusStep = execution.Result.StatusDetails.Step
}

// ErrorsOf collects the message of the initialization step and the messages of the
// steps, so a workflow can assert why another execution failed.
func ErrorsOf(execution *testkube.TestWorkflowExecution) (string, map[string]string) {
	return stepStrings(execution, func(step testkube.TestWorkflowStepResult) string { return step.ErrorMessage })
}

// ReasonsOf collects the reason code of the initialization step and the reason codes of the
// steps, so a workflow can assert the cause of a failure without the words of a message.
func ReasonsOf(execution *testkube.TestWorkflowExecution) (string, map[string]string) {
	return stepStrings(execution, func(step testkube.TestWorkflowStepResult) string { return step.ErrorReason })
}

// stepStrings returns the field of the initialization step and the same field of every step that
// holds a value, keyed by step ref. It returns a nil map when no step holds one.
func stepStrings(execution *testkube.TestWorkflowExecution, field func(step testkube.TestWorkflowStepResult) string) (string, map[string]string) {
	if execution == nil || execution.Result == nil {
		return "", nil
	}
	initialization := ""
	if execution.Result.Initialization != nil {
		initialization = field(*execution.Result.Initialization)
	}
	var steps map[string]string
	for ref, step := range execution.Result.Steps {
		value := field(step)
		if value == "" {
			continue
		}
		if steps == nil {
			steps = make(map[string]string)
		}
		steps[ref] = value
	}
	return initialization, steps
}

// AttemptsOf collects the number of attempts of each step, so a workflow can assert
// how many times another execution ran a step. It skips the steps without a number,
// because the init process sent no execution result for them.
func AttemptsOf(execution *testkube.TestWorkflowExecution) map[string]int64 {
	if execution == nil || execution.Result == nil {
		return nil
	}
	var stepAttempts map[string]int64
	for ref, step := range execution.Result.Steps {
		if step.Attempts == 0 {
			continue
		}
		if stepAttempts == nil {
			stepAttempts = make(map[string]int64)
		}
		stepAttempts[ref] = int64(step.Attempts)
	}
	return stepAttempts
}

// OutputsOf collects the values an execution published through its steps.
//
// Every step emits a single instruction carrying all the files it left in the
// outputs directory; they are flattened into one map here. When two steps use
// the same key, the later step wins - outputs are ordered as they were appended
// to the execution record.
//
// This is the only channel outputs cross executions through, so what it cannot
// carry cannot be exchanged - see the package documentation.
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

// IsReservedRef reports whether a reference has a meaning Testkube assigns,
// rather than naming something the workflow executed.
//
// Reserved references are resolved before the registry, so this is also the
// list of names an `as` alias cannot usefully take.
func IsReservedRef(ref string) bool {
	return ref == ParentRef || ref == RerunRef
}

// reservedRefMeaning describes what a reserved reference addresses, for an error
// that has to explain the collision to a workflow author.
func reservedRefMeaning(ref string) string {
	switch ref {
	case ParentRef:
		return "the execution that scheduled this one"
	case RerunRef:
		return "the execution this one is a rerun of"
	}
	return ref
}
