package data

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
)

type state struct {
	Actions           [][]lite.LiteAction `json:"a,omitempty"`
	CurrentGroupIndex int                 `json:"g,omitempty"`

	CurrentRef    string               `json:"c,omitempty"`
	CurrentStatus string               `json:"s,omitempty"`
	Output        map[string]string    `json:"o,omitempty"`
	Steps         map[string]*StepData `json:"S,omitempty"`
}

func (s *state) GetActions(groupIndex int) []lite.LiteAction {
	if groupIndex < 0 || groupIndex >= len(s.Actions) {
		panic("unknown actions group")
	}
	s.CurrentGroupIndex = groupIndex
	return s.Actions[groupIndex]
}

func (s *state) GetOutput(name string) (expressions.Expression, bool, error) {
	v, ok := s.Output[name]
	if !ok {
		return expressions.None, false, nil
	}
	expr, err := expressions.Compile(v)
	return expr, true, err
}

func (s *state) SetOutput(ref, name string, value interface{}) {
	if s.Output == nil {
		s.Output = make(map[string]string)
	}
	v, err := json.Marshal(value)
	if err == nil {
		s.Output[name] = string(v)
	} else {
		fmt.Printf("Warning: couldn't save '%s' (%s) output: %s\n", name, ref, err.Error())
	}
}

func (s *state) GetStep(ref string) *StepData {
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
	visited := map[*StepData]struct{}{}
	s.getSubSteps(ref, &visited)
	result := make([]*StepData, 0, len(visited))
	for r := range visited {
		result = append(result, r)
	}
	return result
}

func (s *state) SetCurrentStatus(expression string) {
	s.CurrentStatus = expression
}

var currentState = &state{
	Output: map[string]string{},
	Steps:  map[string]*StepData{},
}

func readState(filePath string) {
	b, err := os.ReadFile(filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			panic(err)
		}
		return
	}
	if len(b) == 0 {
		return
	}
	err = json.NewDecoder(bytes.NewBuffer(b)).Decode(&currentState)
	if err != nil {
		panic(err)
	}
}

func persistState(filePath string) {
	b := bytes.Buffer{}
	err := json.NewEncoder(&b).Encode(currentState)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(filePath, b.Bytes(), 0777)
	if err != nil {
		panic(err)
	}
}

func persistTerminationLog() {
	// Read the state
	s := GetState()

	// Get list of statuses
	actions := s.GetActions(s.CurrentGroupIndex)
	statuses := make([]string, 0)
	for i := range actions {
		ref := ""
		if actions[i].Type() == lite.ActionTypeEnd {
			ref = *actions[i].End
		}
		if actions[i].Type() == lite.ActionTypeSetup {
			ref = InitStepName
		}
		if ref == "" {
			continue
		}
		step := s.GetStep(ref)
		if step.Status == nil {
			statuses = append(statuses, fmt.Sprintf("%s,%d", StepStatusAborted, CodeAborted))
		} else {
			statuses = append(statuses, fmt.Sprintf("%s,%d", (*step.Status).Code(), step.ExitCode))
		}
	}

	// Write the termination log
	err := os.WriteFile(TerminationLogPath, []byte(strings.Join(statuses, "/")), 0)
	if err != nil {
		Failf(CodeInternal, "failed to save the termination log: %s", err.Error())
	}
}

var loadStateMu sync.Mutex
var loadedState bool

func GetState() *state {
	defer loadStateMu.Unlock()
	loadStateMu.Lock()
	if !loadedState {
		readState(StatePath)
		loadedState = true
	}
	return currentState
}

func SaveState() {
	persistState(StatePath)
	persistTerminationLog()
}
