package executionworkertypes

import (
	"errors"
	"fmt"
	"testing"

	pkgerrors "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestStartReasonOf(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want StartReason
	}{
		{
			name: "reason of the failed phase",
			err:  WithStartReason(errors.New("job is invalid"), StartReasonJobCreateFailed),
			want: StartReasonJobCreateFailed,
		},
		{
			name: "inner phase keeps its reason when an outer phase wraps it",
			err: WithStartReason(
				pkgerrors.Wrap(WithStartReason(errors.New("unauthorized"), StartReasonImagePullFailed), "failed to process test workflow"),
				StartReasonDefinitionInvalid,
			),
			want: StartReasonImagePullFailed,
		},
		{
			name: "reason survives a plain wrap",
			err:  fmt.Errorf("start: %w", WithStartReason(errors.New("quota exceeded"), StartReasonResourceFailed)),
			want: StartReasonResourceFailed,
		},
		{
			name: "error without a reason",
			err:  errors.New("namespace foo not supported"),
			want: StartReasonUnknown,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, StartReasonOf(tt.err))
		})
	}
}

func TestWithStartReason_Nil(t *testing.T) {
	assert.NoError(t, WithStartReason(nil, StartReasonJobCreateFailed))
}
