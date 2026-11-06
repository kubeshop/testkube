package credentials

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/expressions"
)

type fakeIdentityTokenRepository struct {
	tokens    map[string]string
	err       error
	audiences []string
}

func (f *fakeIdentityTokenRepository) Get(_ context.Context, audience string) (string, error) {
	f.audiences = append(f.audiences, audience)
	if f.err != nil {
		return "", f.err
	}
	return f.tokens[audience], nil
}

func TestIdentityTokenMachine(t *testing.T) {
	tests := []struct {
		name          string
		template      string
		err           error
		want          string
		wantErr       string
		wantAudiences []string
	}{
		{
			name:          "returns the token for the audience",
			template:      `{{ identity_token("sts.amazonaws.com") }}`,
			want:          "aws-token",
			wantAudiences: []string{"sts.amazonaws.com"},
		},
		{
			name:     "requires one argument",
			template: `{{ identity_token() }}`,
			wantErr:  "expects 1 argument, 0 provided",
		},
		{
			name:     "requires a string audience",
			template: `{{ identity_token(5) }}`,
			wantErr:  "expects the audience to be a string",
		},
		{
			name:     "requires a non-empty audience",
			template: `{{ identity_token("") }}`,
			wantErr:  "expects a non-empty audience",
		},
		{
			name:          "returns the Control Plane error",
			template:      `{{ identity_token("sts.amazonaws.com") }}`,
			err:           errors.New("not supported"),
			wantErr:       "not supported",
			wantAudiences: []string{"sts.amazonaws.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &fakeIdentityTokenRepository{
				tokens: map[string]string{"sts.amazonaws.com": "aws-token"},
				err:    tt.err,
			}
			var observed []string
			machine := NewIdentityTokenMachine(repository, func(_ string, token string) {
				observed = append(observed, token)
			})

			got, err := expressions.EvalTemplate(tt.template, machine)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Empty(t, observed)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
				assert.Equal(t, []string{tt.want}, observed)
			}
			assert.Equal(t, tt.wantAudiences, repository.audiences)
		})
	}
}
