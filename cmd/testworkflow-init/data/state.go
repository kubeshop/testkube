package data

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/output"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
)

type state struct {
	Actions           [][]lite.LiteAction               `json:"a,omitempty"`
	InternalConfig    testworkflowconfig.InternalConfig `json:"C,omitempty"`
	CurrentGroupIndex int                               `json:"g,omitempty"`

	CurrentRef    string               `json:"c,omitempty"`
	CurrentStatus string               `json:"s,omitempty"`
	Output        map[string]string    `json:"o,omitempty"`
	Steps         map[string]*StepData `json:"S,omitempty"`

	Signature          []testworkflowconfig.SignatureConfig       `json:"G,omitempty"`
	ContainerResources testworkflowconfig.ContainerResourceConfig `json:"R,omitempty"`
}

func (s *state) GetActions(groupIndex int) ([]lite.LiteAction, error) {
	stateMu.Lock()
	defer stateMu.Unlock()

	if groupIndex < 0 || groupIndex >= len(s.Actions) {
		return nil, fmt.Errorf("unknown actions group %d (available: 0-%d)", groupIndex, len(s.Actions)-1)
	}
	s.CurrentGroupIndex = groupIndex
	return s.Actions[groupIndex], nil
}

func (s *state) GetOutput(name string) (expressions.Expression, bool, error) {
	stateMu.RLock()
	v, ok := s.Output[name]
	stateMu.RUnlock()

	if !ok {
		return expressions.None, false, nil
	}
	expr, err := expressions.Compile(v)
	return expr, true, err
}

func (s *state) SetOutput(ref, name string, value interface{}) {
	stateMu.Lock()
	defer stateMu.Unlock()

	if s.Output == nil {
		s.Output = make(map[string]string)
	}
	v, err := json.Marshal(value)
	if err == nil {
		s.Output[name] = string(v)
	} else {
		output.Std.Warnf("warn: couldn't save '%s' (%s) output: %s\n", name, ref, err.Error())
	}
}

func (s *state) GetStep(ref string) *StepData {
	stateMu.Lock()
	defer stateMu.Unlock()

	if s.Steps[ref] == nil {
		s.Steps[ref] = &StepData{Ref: ref}
	}
	if s.Steps[ref].Condition == "" {
		s.Steps[ref].Condition = "passed"
	}
	return s.Steps[ref]
}

func (s *state) getSubSteps(ref string, visited *map[*StepData]struct{}) {
	// Ignore already visited node
	if _, ok := (*visited)[s.Steps[ref]]; ok {
		return
	}

	// Append the node
	(*visited)[s.Steps[ref]] = struct{}{}

	// Visit its children
	for _, sub := range s.Steps {
		if slices.Contains(sub.Parents, ref) {
			s.getSubSteps(sub.Ref, visited)
		}
	}
}

func (s *state) GetSubSteps(ref string) []*StepData {
	stateMu.RLock()
	defer stateMu.RUnlock()

	visited := map[*StepData]struct{}{}
	s.getSubSteps(ref, &visited)
	result := make([]*StepData, 0, len(visited))
	for r := range visited {
		result = append(result, r)
	}
	return result
}

func (s *state) SetCurrentStatus(expression string) {
	stateMu.Lock()
	defer stateMu.Unlock()

	s.CurrentStatus = expression
}

var currentState = &state{
	Output: map[string]string{},
	Steps:  map[string]*StepData{},
}

func readState(filePath string) error {
	b, err := os.ReadFile(filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to read state file: %w", err)
		}
		return nil
	}
	if len(b) == 0 {
		return nil
	}
	err = json.NewDecoder(bytes.NewBuffer(b)).Decode(&currentState)
	if err != nil {
		return fmt.Errorf("failed to decode state file: %w", err)
	}
	return nil
}

func persistState(filePath string) error {
	stateMu.RLock()
	b := bytes.Buffer{}
	err := json.NewEncoder(&b).Encode(currentState)
	stateMu.RUnlock()

	if err != nil {
		return fmt.Errorf("failed to encode state: %w", err)
	}

	// Write with 0777 permissions - REQUIRED for Kubernetes shared volumes
	// where containers may run with different UIDs. See state_manager.go
	// for detailed explanation. DO NOT CHANGE!
	err = os.WriteFile(filePath, b.Bytes(), 0777)
	if err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}
	return nil
}

var (
	prevTerminationLog []string
)

func persistTerminationLog() {
	// Read the state
	s := GetState()

	// Get list of statuses
	actions, err := s.GetActions(s.CurrentGroupIndex)
	if err != nil {
		// Log error but continue with empty statuses to avoid crash
		output.Std.Warnf("failed to get actions for termination log: %v\n", err)
		return
	}
	statuses := make([]string, 0)
	for i := range actions {
		ref := ""
		if actions[i].Type() == lite.ActionTypeEnd {
			ref = *actions[i].End
		}
		if actions[i].Type() == lite.ActionTypeSetup {
			ref = constants.InitStepName
		}
		if ref == "" {
			continue
		}
		step := s.GetStep(ref)
		if step.Status == nil {
			statuses = append(statuses, fmt.Sprintf("%s,%d", constants.StepStatusAborted, constants.CodeAborted))
		} else {
			statuses = append(statuses, fmt.Sprintf("%s,%d", (*step.Status).Code(), step.ExitCode))
		}
	}

	// Avoid using FS when it is not necessary
	if slices.Equal(prevTerminationLog, statuses) {
		return
	}
	prevTerminationLog = statuses

	// Write the termination log
	err = os.WriteFile(constants.TerminationLogPath, []byte(strings.Join(statuses, "/")), 0)
	if err != nil {
		output.UnsafeExitErrorf(constants.CodeInternal, "failed to save the termination log: %s", err.Error())
	}
}

var (
	loadStateMu sync.Mutex
	loadedState bool

	// stateMu protects all state mutations to prevent race conditions
	// between the control server and main execution
	stateMu sync.RWMutex
)

func GetState() *state {
	defer loadStateMu.Unlock()
	loadStateMu.Lock()
	if !loadedState {
		if err := readState(constants.StatePath); err != nil {
			// Log error but continue with empty state
			output.UnsafeExitErrorf(constants.CodeInternal, "failed to load state: %s", err.Error())
		}
		loadedState = true
	}
	return currentState
}

func SaveTerminationLog() {
	persistTerminationLog()
}

func SaveState() {
	if err := persistState(constants.StatePath); err != nil {
		output.UnsafeExitErrorf(constants.CodeInternal, "failed to save state: %s", err.Error())
	}
	persistTerminationLog()
}

// ClearState clears the singleton state - for testing only
func ClearState() {
	loadStateMu.Lock()
	defer loadStateMu.Unlock()
	loadedState = false
	currentState = &state{
		Output: map[string]string{},
		Steps:  map[string]*StepData{},
	}
}
