// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowcontroller

import (
	"encoding/json"
	"time"

	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-init/data"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
)

type Notification struct {
	Timestamp time.Time                    `json:"ts"`
	Result    *testkube.TestWorkflowResult `json:"result,omitempty"`
	Ref       string                       `json:"ref,omitempty"`
	Log       string                       `json:"log,omitempty"`
	Output    *data.Instruction            `json:"output,omitempty"`
}

func (n *Notification) ToInternal() testkube.TestWorkflowExecutionNotification {
	return testkube.TestWorkflowExecutionNotification{
		Ts:     n.Timestamp,
		Result: n.Result,
		Ref:    n.Ref,
		Log:    n.Log,
		Output: InstructionToInternal(n.Output),
	}
}

func InstructionToInternal(instruction *data.Instruction) *testkube.TestWorkflowOutput {
	if instruction == nil {
		return nil
	}
	value := map[string]interface{}(nil)
	if instruction.Value != nil {
		v, _ := json.Marshal(instruction.Value)
		e := json.Unmarshal(v, &value)
		if e != nil {
			log.DefaultLogger.Warnf("invalid output passed from TestWorkflow - %v", instruction.Value)
		}
	}
	if v, ok := instruction.Value.(map[string]interface{}); ok {
		value = v
	}
	return &testkube.TestWorkflowOutput{
		Ref:   instruction.Ref,
		Name:  instruction.Name,
		Value: value,
	}
}
