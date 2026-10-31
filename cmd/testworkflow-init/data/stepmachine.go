package data

import (
	"path/filepath"
	"strings"

	"github.com/kubeshop/testkube/pkg/expressions"
)

const (
	stepPrefix = "step."
	// The words that follow the step id in an expression.
	stepResultsKey    = "results"
	stepStatusKey     = "status"
	stepExitCodeKey   = "exitCode"
	stepOutputsPrefix = "outputs."
)

var (
	stepResultsBase = "/data/.steps"
)

func GetStepResultsBase() string {
	return stepResultsBase
}

func SetStepResultsBase(base string) {
	stepResultsBase = base
}

// StepResultsDir returns the results directory path for a step.
// Step IDs are validated by ValidateStepId (alphanumeric + underscores only),
// so path traversal is not possible.
func StepResultsDir(id string) string {
	return filepath.Join(stepResultsBase, id)
}

// StepMachine resolves step-scoped expressions like step.results,
// step.<id>.results, step.<id>.outputs.<key>, step.<id>.status, and step.<id>.exitCode.
// The status of a skipped step resolves, and its exit code does not, because it ran no command.
var StepMachine = expressions.NewMachine().
	RegisterAccessorExt(func(name string) (interface{}, bool, error) {
		if !strings.HasPrefix(name, stepPrefix) {
			return nil, false, nil
		}
		suffix := name[len(stepPrefix):]
		state := GetState()

		if suffix == stepResultsKey {
			currentStep := state.GetStep(state.CurrentRef)
			if currentStep.Id == "" {
				return nil, false, nil
			}
			return StepResultsDir(currentStep.Id), true, nil
		}

		parts := strings.SplitN(suffix, ".", 2)
		if len(parts) != 2 {
			return nil, false, nil
		}
		stepId, rest := parts[0], parts[1]

		switch {
		case rest == stepResultsKey:
			if state.GetStepByID(stepId) == nil {
				return nil, false, nil
			}
			return StepResultsDir(stepId), true, nil

		case rest == stepStatusKey:
			// A step that did not run has no status, so the expression stays unresolved.
			step := state.GetStepByID(stepId)
			if step == nil || step.Status == nil {
				return nil, false, nil
			}
			return string(*step.Status), true, nil

		case rest == stepExitCodeKey:
			// Only a step that started has an exit code. A step that a condition skipped would
			// otherwise report the zero value, which reads as a success.
			step := state.GetStepByID(stepId)
			if step == nil || step.Status == nil || !step.IsStarted() {
				return nil, false, nil
			}
			return int64(step.ExitCode), true, nil

		case strings.HasPrefix(rest, stepOutputsPrefix):
			key := rest[len(stepOutputsPrefix):]
			if key == "" {
				return nil, false, nil
			}
			return state.GetStepOutput(stepId, key)
		}

		return nil, false, nil
	})
