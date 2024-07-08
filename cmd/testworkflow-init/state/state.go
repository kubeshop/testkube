package state

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/output"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor"
)

type state struct {
	Actions    [][]testworkflowprocessor.Action       `json:"actions"`
	Conditions map[string]string                      `json:"conditions"`
	Parents    map[string][]string                    `json:"parents"`
	Timeouts   map[string]string                      `json:"timeouts"`
	Pauses     map[string]bool                        `json:"pauses"`
	Retries    map[string]testworkflowsv1.RetryPolicy `json:"retries"`
	Results    map[string]string                      `json:"results"`

	Status string                    `json:"status"`
	Output map[string]string         `json:"output"`
	Steps  map[string]*data.StepInfo `json:"steps"`
}

func (s *state) GetActions(groupIndex int) []testworkflowprocessor.Action {
	if groupIndex < 0 || groupIndex >= len(s.Actions) {
		panic("unknown actions group")
	}
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

func (s *state) SetCondition(ref, expression string) {
	s.Conditions[ref] = expression
}

func (s *state) SetParents(ref string, parents []string) {
	s.Parents[ref] = parents
}

func (s *state) SetPause(ref string, pause bool) {
	s.Pauses[ref] = pause
}

func (s *state) SetTimeout(ref string, timeout string) {
	s.Timeouts[ref] = timeout
}

func (s *state) SetResult(ref, expression string) {
	s.Results[ref] = expression
}

func (s *state) SetCurrentStatus(expression string) {
	s.Status = expression
}

func (s *state) SetRetryPolicy(ref string, policy testworkflowsv1.RetryPolicy) {
	s.Retries[ref] = policy
}

var currentState = &state{
	Conditions: map[string]string{},
	Parents:    map[string][]string{},
	Timeouts:   map[string]string{},
	Pauses:     map[string]bool{},
	Retries:    map[string]testworkflowsv1.RetryPolicy{},
	Results:    map[string]string{},
	Output:     map[string]string{},
	Steps:      map[string]*data.StepInfo{},
}

func (s *state) SetStepStatus(ref string, status data.StepStatus) {
	if _, ok := s.Steps[ref]; !ok {
		s.Steps[ref] = &data.StepInfo{}
	}
	s.Steps[ref].Status = status
}

func (s *state) GetStep(ref string) *data.StepInfo {
	return s.Steps[ref]
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
}

func SaveTerminationLog() {
	// Write the termination log TODO: do that generically
	err := os.WriteFile(TerminationLogPath, []byte(",0"), 0)
	if err != nil {
		output.Failf(output.CodeInternal, "failed to mark as done: %s", err.Error())
	}
}
