package data

import (
	"path/filepath"
	"strings"

	"github.com/kubeshop/testkube/pkg/expressions"
)

const (
	stepPrefix      = "step."
	stepResultsBase = "/data/.steps/"
)

// StepResultsDir returns the results directory path for a step.
// Step IDs are validated by ValidateStepId (alphanumeric + underscores only),
// so path traversal is not possible.
func StepResultsDir(id string) string {
	return filepath.Join(stepResultsBase, id)
}

// StepMachine resolves step-scoped expressions:
//   - step.results -> current step's results directory
//   - step.<id>.results -> named step's results directory
var StepMachine = expressions.NewMachine().
	RegisterAccessor(func(name string) (interface{}, bool) {
		if !strings.HasPrefix(name, stepPrefix) {
			return nil, false
		}
		suffix := name[len(stepPrefix):]
		state := GetState()

		if suffix == "results" {
			currentStep := state.GetStep(state.CurrentRef)
			if currentStep.Id == "" {
				return nil, false
			}
			return StepResultsDir(currentStep.Id), true
		}

		parts := strings.SplitN(suffix, ".", 2)
		if len(parts) != 2 {
			return nil, false
		}
		stepId, property := parts[0], parts[1]

		if property == "results" {
			if state.GetStepByID(stepId) == nil {
				return nil, false
			}
			return StepResultsDir(stepId), true
		}

		return nil, false
	})
