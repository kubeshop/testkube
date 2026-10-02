package executionworkertypes

import (
	"errors"
	"fmt"
	"testing"

	pkgerrors "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestStartReasonOf(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want testkube.StartReason
	}{
		{
			name: "reason of the failed phase",
			err:  WithStartReason(errors.New("job is invalid"), testkube.StartReasonJobCreateFailed),
			want: testkube.StartReasonJobCreateFailed,
		},
		{
			name: "inner phase keeps its reason when an outer phase sets another",
			err: WithStartReason(
				WithStartReason(errors.New("unauthorized"), testkube.StartReasonImagePullFailed),
				testkube.StartReasonDefinitionInvalid,
			),
			want: testkube.StartReasonImagePullFailed,
		},
		{
			name: "inner phase keeps its reason through a wrap",
			err: WithStartReason(
				pkgerrors.Wrap(WithStartReason(errors.New("unauthorized"), testkube.StartReasonImagePullFailed), "applying image data"),
				testkube.StartReasonDefinitionInvalid,
			),
			want: testkube.StartReasonImagePullFailed,
		},
		{
			name: "reason survives a plain wrap",
			err:  fmt.Errorf("start: %w", WithStartReason(errors.New("quota exceeded"), testkube.StartReasonResourceFailed)),
			want: testkube.StartReasonResourceFailed,
		},
		{
			name: "error without a reason",
			err:  errors.New("namespace foo not supported"),
			want: testkube.StartReasonUnknown,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, StartReasonOf(tt.err))
		})
	}
}

func TestWithStartReason_Nil(t *testing.T) {
	assert.NoError(t, WithStartReason(nil, testkube.StartReasonJobCreateFailed))
}
