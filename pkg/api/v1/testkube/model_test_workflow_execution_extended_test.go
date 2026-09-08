package testkube

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTestWorkflowExecution_InitializationError(t *testing.T) {
	tests := []struct {
		name             string
		header           string
		err              error
		wantErrorMessage string
	}{
		{
			name:             "puts the plain header above the error",
			header:           "Failed to run execution",
			err:              errors.New("image not found"),
			wantErrorMessage: "Failed to run execution\nimage not found",
		},
		{
			name:             "stores only the error when the header is empty",
			header:           "",
			err:              errors.New("image not found"),
			wantErrorMessage: "image not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &TestWorkflowExecution{
				Result: &TestWorkflowResult{
					Initialization: &TestWorkflowStepResult{},
					Steps:          map[string]TestWorkflowStepResult{},
				},
			}

			e.InitializationError(tt.header, tt.err)

			assert.Equal(t, tt.wantErrorMessage, e.Result.Initialization.ErrorMessage)
		})
	}
}
