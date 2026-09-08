package grpc

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStartErrorReason(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "image inspection error inside the processing wrap",
			err:  errors.New("failed to process test workflow: resolving image error: cypress/nope:1: inspecting image: UNAUTHORIZED"),
			want: StartReasonImagePullFailed,
		},
		{
			name: "expression or template error at processing",
			err:  errors.New("failed to process test workflow: unknown template"),
			want: StartReasonDefinitionInvalid,
		},
		{
			name: "kubernetes rejected the job",
			err:  errors.New("failed to deploy test workflow: Job.batch \"x\" is invalid: spec.template: ..."),
			want: StartReasonJobCreateFailed,
		},
		{
			name: "kubernetes rejected a secret before the job",
			err:  errors.New("failed to deploy test workflow: failed to deploy secrets: secrets \"x\" is forbidden"),
			want: StartReasonResourceFailed,
		},
		{
			name: "kubernetes rejected a volume claim before the job",
			err:  errors.New("failed to deploy test workflow: failed to deploy pvcs: quota exceeded"),
			want: StartReasonResourceFailed,
		},
		{
			name: "admission webhook denied the job",
			err:  errors.New("admission webhook \"policy.example\" denied the request: no securityContext"),
			want: StartReasonJobCreateFailed,
		},
		{
			name: "anything else",
			err:  errors.New("namespace foo not supported"),
			want: StartReasonUnknown,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, startErrorReason(tt.err))
		})
	}
}
