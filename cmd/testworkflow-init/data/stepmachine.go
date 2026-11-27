package data

import (
	"encoding/json"
	"os"
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
	stepAgentPrefix   = "agent."
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

		case strings.HasPrefix(rest, stepAgentPrefix):
			if state.GetStepByID(stepId) == nil {
				return nil, false, nil
			}
			value, ok := stepAgentValue(stepId, rest[len(stepAgentPrefix):])
			if !ok {
				return nil, false, nil
			}
			return value, true, nil

		case strings.HasPrefix(rest, stepOutputsPrefix):
			key := rest[len(stepOutputsPrefix):]
			if key == "" {
				return nil, false, nil
			}
			return state.GetStepOutput(stepId, key)
		}

		return nil, false, nil
	})

const (
	// StepAgentFile is the file in the results directory of an agent step that holds the data of
	// its turn. The toolkit writes it, and step.<id>.agent.<key> reads it.
	//
	// The data is not a step output. The outputs of all steps make one set of names for the
	// workflow, which a parent workflow reads, so a value that each agent step writes would take
	// a name from the author and the last step would win.
	StepAgentFile = "agent.json"
	// StepAgentSession is the key of the session of the AI Agent that holds the turn.
	StepAgentSession = "session"
	// StepAgentName is the key of the name of the AI Agent.
	StepAgentName = "name"
)

// stepAgentValue reads one value of the turn of an agent step. A step that is not an agent step,
// or that started no turn, has no file, so the expression stays unresolved.
func stepAgentValue(stepId, key string) (string, bool) {
	content, err := os.ReadFile(filepath.Join(StepResultsDir(stepId), StepAgentFile))
	if err != nil {
		return "", false
	}
	var values map[string]string
	if err := json.Unmarshal(content, &values); err != nil {
		return "", false
	}
	value, ok := values[key]
	return value, ok
}
