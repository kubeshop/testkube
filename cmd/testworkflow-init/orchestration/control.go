package orchestration

import (
	"encoding/json"
	"time"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/output"
)

func Start(step *data.StepData) {
	state := data.GetState()
	state.CurrentRef = step.Ref
	startedAt := time.Now()
	step.StartedAt = &startedAt
	instructions.PrintHint(step.Ref, constants.InstructionStart)
}

func Pause(step *data.StepData, ts time.Time) {
	step.RegisterPauseStart(ts)
	instructions.PrintHintDetails(step.Ref, constants.InstructionPause, ts.UTC().Format(constants.PreciseTimeFormat))
}

func Resume(step *data.StepData, ts time.Time) {
	step.RegisterPauseEnd(ts)
	instructions.PrintHintDetails(step.Ref, constants.InstructionResume, ts.UTC().Format(constants.PreciseTimeFormat))
}

func FinishExecution(step *data.StepData, result constants.ExecutionResult) {
	instructions.PrintHintDetails(step.Ref, constants.InstructionExecution, result)
}

func End(step *data.StepData) {
	if !step.IsFinished() {
		v, e := json.Marshal(step)
		output.ExitErrorf(constants.CodeInternal, "cannot mark unfinished step as finished: %s, %v", string(v), e)
	}
	instructions.PrintHintDetails(step.Ref, constants.InstructionEnd, *step.Status)
}
