package orchestration

import (
	"encoding/json"
	"time"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/output"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
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

// FinishExecution sends the execution result of a step.
func FinishExecution(step *data.StepData, result constants.ExecutionResult) {
	instructions.PrintHintDetails(step.Ref, constants.InstructionExecution, result)
}

// FinishTimedOutExecution sends the execution result of a step that its timeout, or the timeout of a parent group, stopped.
// The timeout stops the process, or it ends before the process starts. In both cases the step gets the exit code of an
// aborted process, so an earlier attempt does not leave its exit code.
func FinishTimedOutExecution(step *data.StepData) {
	step.SetExitCode(constants.CodeAborted)
	FinishExecution(step, constants.ExecutionResult{
		ExitCode:  step.ExitCode,
		Details:   testkube.StopReasonStepTimeout.Sentence(),
		Iteration: int(step.Iteration),
	})
}

func End(step *data.StepData) {
	if !step.IsFinished() {
		v, e := json.Marshal(step)
		output.ExitErrorf(constants.CodeInternal, "cannot mark unfinished step as finished: %s, %v", string(v), e)
	}
	instructions.PrintHintDetails(step.Ref, constants.InstructionEnd, *step.Status)
}
