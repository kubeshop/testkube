package data

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor"
)

type state struct {
	Actions [][]testworkflowprocessor.Action `json:"a,omitempty"`

	CurrentRef    string               `json:"c,omitempty"`
	CurrentStatus string               `json:"s,omitempty"`
	Output        map[string]string    `json:"o,omitempty"`
	Steps         map[string]*StepData `json:"S,omitempty"`
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

func (s *state) GetStep(ref string) *StepData {
	if s.Steps[ref] == nil {
		s.Steps[ref] = &StepData{}
	}
	if s.Steps[ref].Condition == "" {
		s.Steps[ref].Condition = "passed"
	}
	return s.Steps[ref]
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
		Failf(CodeInternal, "failed to mark as done: %s", err.Error())
	}
}
