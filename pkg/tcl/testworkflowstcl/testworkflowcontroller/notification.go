// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowcontroller

import (
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type Notification struct {
	Timestamp time.Time                    `json:"ts"`
	Result    *testkube.TestWorkflowResult `json:"result,omitempty"`
	Ref       string                       `json:"ref,omitempty"`
	Log       string                       `json:"log,omitempty"`
	Output    *Instruction                 `json:"output,omitempty"`
}

func (n *Notification) ToInternal() testkube.TestWorkflowExecutionNotification {
	return testkube.TestWorkflowExecutionNotification{
		Ts:     n.Timestamp,
		Result: n.Result,
		Ref:    n.Ref,
		Log:    n.Log,
		Output: n.Output.ToInternal(),
	}
}
