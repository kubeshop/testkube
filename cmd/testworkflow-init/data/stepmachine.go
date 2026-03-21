package data

import (
	"path/filepath"
	"strings"

	"github.com/kubeshop/testkube/pkg/expressions"
)

const (
	stepPrefix      = "step."
	stepResultsBase = "/data/.steps"
)

// StepResultsDir returns the results directory path for a step.
// Step IDs are validated by ValidateStepId (alphanumeric + underscores only),
// so path traversal is not possible.
func StepResultsDir(id string) string {
	return filepath.Join(stepResultsBase, id)
}

// StepMachine resolves step-scoped expressions like step.results,
// step.<id>.results, and step.<id>.outputs.<key>.
var StepMachine = expressions.NewMachine().
	RegisterAccessorExt(func(name string) (interface{}, bool, error) {
		if !strings.HasPrefix(name, stepPrefix) {
			return nil, false, nil
		}
		suffix := name[len(stepPrefix):]
		state := GetState()

		if suffix == "results" {
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
		case rest == "results":
			if state.GetStepByID(stepId) == nil {
				return nil, false, nil
			}
			return StepResultsDir(stepId), true, nil

		case strings.HasPrefix(rest, "outputs."):
			key := rest[len("outputs."):]
			if key == "" {
				return nil, false, nil
			}
			return state.GetStepOutput(stepId, key)
		}

		return nil, false, nil
	})
