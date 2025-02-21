package controller

import (
	"encoding/json"
	"time"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
)

type Notification struct {
	Timestamp time.Time                    `json:"ts"`
	Result    *testkube.TestWorkflowResult `json:"result,omitempty"`
	Ref       string                       `json:"ref,omitempty"`
	Log       string                       `json:"log,omitempty"`
	Output    *instructions.Instruction    `json:"output,omitempty"`
	Temporary bool                         `json:"temporary,omitempty"`
}

func (n *Notification) ToInternal() testkube.TestWorkflowExecutionNotification {
	return testkube.TestWorkflowExecutionNotification{
		Ts:        n.Timestamp,
		Result:    n.Result,
		Ref:       n.Ref,
		Log:       n.Log,
		Output:    InstructionToInternal(n.Output),
		Temporary: n.Temporary,
	}
}

func InstructionToInternal(instruction *instructions.Instruction) *testkube.TestWorkflowOutput {
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
