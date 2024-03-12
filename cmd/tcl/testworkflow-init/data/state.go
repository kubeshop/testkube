// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package data

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
)

const (
	defaultInternalPath       = "/.tktw"
	defaultTerminationLogPath = "/dev/termination-log"
)

type state struct {
	Status TestWorkflowStatus   `json:"status"`
	Steps  map[string]*StepInfo `json:"steps"`
	Output map[string]string    `json:"output"`
}

var State = &state{
	Steps:  map[string]*StepInfo{},
	Output: map[string]string{},
}

func (s *state) GetStep(ref string) *StepInfo {
	_, ok := State.Steps[ref]
	if !ok {
		State.Steps[ref] = &StepInfo{Ref: ref}
	}
	return State.Steps[ref]
}

func (s *state) GetOutput(name string) (expressionstcl.Expression, bool, error) {
	v, ok := s.Output[name]
	if !ok {
		return expressionstcl.None, false, nil
	}
	expr, err := expressionstcl.Compile(v)
	return expr, true, err
}

func (s *state) GetSelfStatus() string {
	if Step.Executed {
		return string(Step.Status)
	}
	v := s.GetStep(Step.Ref)
	if v.Status != StepStatusPassed {
		return string(v.Status)
	}
	return string(Step.Status)
}

func (s *state) GetStatus() string {
	if Step.Executed {
		return string(Step.Status)
	}
	if Step.InitStatus == "" {
		return string(s.Status)
	}
	v, err := RefStatusExpression(Step.InitStatus)
	if err != nil {
		return string(s.Status)
	}
	str, _ := v.Static().StringValue()
	if str == "" {
		return string(s.Status)
	}
	return str
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
	err = gob.NewDecoder(bytes.NewBuffer(b)).Decode(&State)
	if err != nil {
		panic(err)
	}
}

func persistState(filePath string) {
	b := bytes.Buffer{}
	err := gob.NewEncoder(&b).Encode(State)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(filePath, b.Bytes(), 0777)
	if err != nil {
		panic(err)
	}
}

func recomputeStatuses() {
	// Read current status
	status := StepStatus(State.GetSelfStatus())

	// Update own status
	State.GetStep(Step.Ref).SetStatus(status)

	// Update expected failure statuses
	Iterate(Config.Resulting, func(r Rule) bool {
		v, err := RefSuccessExpression(r.Expr)
		if err != nil {
			return false
		}
		vv, _ := v.Static().BoolValue()
		if !vv {
			for _, ref := range r.Refs {
				if ref == "" {
					State.Status = TestWorkflowStatusFailed
				} else {
					State.GetStep(ref).SetStatus(StepStatusFailed)
				}
			}
		}
		return true
	})
}

func persistStatus(filePath string) {
	// Persist container termination result
	res := fmt.Sprintf(`%s,%d`, State.GetStep(Step.Ref).Status, Step.ExitCode)
	err := os.WriteFile(filePath, []byte(res), 0755)
	if err != nil {
		panic(err)
	}
}

func LoadState() {
	readState(filepath.Join(defaultInternalPath, "state"))
}

func Finish() {
	// Persist step information and shared data
	recomputeStatuses()
	persistStatus(defaultTerminationLogPath)
	persistState(filepath.Join(defaultInternalPath, "state"))

	// Kill the sub-process
	if Step.Cmd != nil && Step.Cmd.Process != nil {
		_ = Step.Cmd.Process.Kill()
	}

	// Emit end hint to allow exporting the timestamp
	PrintHint(Step.Ref, "end")

	// The init process needs to finish with zero exit code,
	// to continue with the next container.
	os.Exit(0)
}
