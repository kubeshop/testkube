package controller

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestNotifier_Instruction(t *testing.T) {
	const ref = "rstep1"
	execution := func(iteration int) instructions.Instruction {
		return instructions.Instruction{Ref: ref, Name: constants.InstructionExecution, Value: constants.ExecutionResult{ExitCode: 1, Iteration: iteration}}
	}
	retry := func(iteration int) instructions.Instruction {
		return instructions.Instruction{Ref: ref, Name: constants.InstructionIteration, Value: iteration}
	}

	tests := []struct {
		name         string
		hints        []instructions.Instruction
		wantAttempts int32
	}{
		{
			name:         "reports 1 attempt for a step without retry",
			hints:        []instructions.Instruction{execution(0)},
			wantAttempts: 1,
		},
		{
			name:         "sets the attempts from the iteration of the last execution result",
			hints:        []instructions.Instruction{execution(0), retry(1), execution(1), retry(2), execution(2)},
			wantAttempts: 3,
		},
		{
			name:         "counts the attempts that time out and send no execution result",
			hints:        []instructions.Instruction{retry(1), retry(2)},
			wantAttempts: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			ch := make(chan ChannelMessage[Notification], 100)
			n := &notifier{
				ctx: ctx,
				ch:  ch,
				result: testkube.TestWorkflowResult{
					Initialization: &testkube.TestWorkflowStepResult{Status: common.Ptr(testkube.PASSED_TestWorkflowStepStatus)},
					Steps: map[string]testkube.TestWorkflowStepResult{
						ref: {Status: common.Ptr(testkube.RUNNING_TestWorkflowStepStatus)},
					},
				},
			}

			for _, hint := range tt.hints {
				n.Instruction(time.Now(), hint, "exec-1")
			}

			// Read the sent result, because the notifier sends a copy of its state.
			var last *testkube.TestWorkflowResult
			for len(ch) > 0 {
				if message := <-ch; message.Value.Result != nil {
					last = message.Value.Result
				}
			}
			require.NotNil(t, last)
			assert.Equal(t, tt.wantAttempts, last.Steps[ref].Attempts)
		})
	}
}
