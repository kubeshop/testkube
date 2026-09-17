package testkube

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTestWorkflowStatusDetails_Clone(t *testing.T) {
	tests := []struct {
		name    string
		details *TestWorkflowStatusDetails
	}{
		{
			name: "a nil object stays nil",
		},
		{
			name:    "an object without a user",
			details: &TestWorkflowStatusDetails{Type_: string(StatusDetailsTypeInitFailure), Reason: string(StopReasonUnschedulable)},
		},
		{
			name: "an object with a user",
			details: &TestWorkflowStatusDetails{
				Type_:  string(StatusDetailsTypeUserCancel),
				Reason: string(StopReasonUserCancel),
				Actor:  string(StopActorUser),
				User:   &TestWorkflowStatusDetailsUser{Name: "Ada", Email: "ada@example.com"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clone := tt.details.Clone()
			if tt.details == nil {
				assert.Nil(t, clone)
				return
			}
			assert.Equal(t, *tt.details, *clone)

			// A write to the copy must not reach the original, so the caller of Clone owns its copy.
			clone.Reason = string(StopReasonForceCancel)
			assert.NotEqual(t, clone.Reason, tt.details.Reason)
			if tt.details.User != nil {
				clone.User.Name = "Grace"
				assert.Equal(t, "Ada", tt.details.User.Name)
			}
		})
	}
}

func TestTestWorkflowStatusDetails_Label(t *testing.T) {
	tests := []struct {
		name    string
		details *TestWorkflowStatusDetails
		want    string
	}{
		{
			name:    "the layer and the code",
			details: &TestWorkflowStatusDetails{Type_: string(StatusDetailsTypeInitFailure), Reason: string(StopReasonUnschedulable)},
			want:    "init-failure: unschedulable",
		},
		{
			name:    "the layer alone when the code is empty",
			details: &TestWorkflowStatusDetails{Type_: string(StatusDetailsTypeUnknown)},
			want:    "unknown",
		},
		{
			name: "an execution that passed reads empty",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.details.Label())
		})
	}
}
