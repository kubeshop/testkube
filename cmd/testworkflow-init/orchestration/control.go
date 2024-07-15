package orchestration

import (
	"time"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/internal/common"
)

func Start(ref string) {
	state := data.GetState()
	state.CurrentRef = ref
	state.GetStep(ref).StartedAt = common.Ptr(time.Now())
	data.PrintHint(ref, constants.InstructionStart)
}

func Pause(ref string) {
	//data.Step.Pause(time.Now())
}

func Resume(ref string) {
	//d
}

func FinishExecution(ref string, result constants.ExecutionResult) {
	data.PrintHintDetails(ref, constants.InstructionExecution, result)
}

func End(ref string, status data.StepStatus) {
	data.PrintHintDetails(ref, constants.InstructionEnd, status)
}
