package orchestration

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
)

func Start(step *data.StepData) {
	state := data.GetState()
	state.CurrentRef = step.Ref
	startedAt := time.Now()
	step.StartedAt = &startedAt
	data.PrintHint(step.Ref, constants.InstructionStart)
}

func Pause(step *data.StepData) {
	//data.Step.Pause(time.Now())
}

func Resume(step *data.StepData) {
	//d
}

func FinishExecution(step *data.StepData, result constants.ExecutionResult) {
	data.PrintHintDetails(step.Ref, constants.InstructionExecution, result)
}

func End(step *data.StepData) {
	if !step.IsFinished() {
		v, e := json.Marshal(step)
		panic(fmt.Sprintf("cannot mark unfinished step as finished: %s, %v", string(v), e))
	}
	data.PrintHintDetails(step.Ref, constants.InstructionEnd, *step.Status)
}
