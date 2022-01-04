package testkube

import (
	"encoding/json"
)

func (r *TestStepExecutionResult) Err(err error) TestStepExecutionResult {
	if r.Execution == nil {
		execution := NewFailedExecution(err)
		r.Execution = &execution

	}
	e := r.Execution.Err(err)
	r.Execution = &e
	return *r
}

func (r *TestStepExecutionResult) IsFailed() bool {
	if r.Execution != nil {
		return r.Execution.IsFailed()
	}

	return true
}

func (result *TestStepExecutionResult) UnmarshalJSON(data []byte) error {
	var r struct {
		Step      *TestStepBase `json:"step,omitempty"`
		Script    *ObjectRef    `json:"script,omitempty"`
		Execution *Execution    `json:"execution,omitempty"`
	}

	err := json.Unmarshal(data, &r)
	if err != nil {
		return err
	}

	if s := TestStepBase(*r.Step).GetTestStep(); s != nil {
		result.Step = &s
	}

	result.Script = r.Script
	result.Execution = r.Execution

	return nil
}
