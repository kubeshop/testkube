package context

import (
	"errors"
	"testing"

	pkgerrors "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	cloudclient "github.com/kubeshop/testkube/pkg/cloud/client"
)

func TestOrgEnvLookupHint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "a refused credential points at logging in again",
			err:  &cloudclient.StatusError{StatusCode: 401},
			want: "refused the stored credential",
		},
		{
			name: "a forbidden credential points at logging in again",
			err:  &cloudclient.StatusError{StatusCode: 403},
			want: "refused the stored credential",
		},
		{
			name: "a missing record points at the ids",
			err:  &cloudclient.StatusError{StatusCode: 404},
			want: "exist on this Control Plane",
		},
		{
			// The lookup wraps the client error before it reaches the hint.
			name: "the status is read through a wrapped error",
			err:  pkgerrors.Wrap(&cloudclient.StatusError{StatusCode: 401}, "error getting organization"),
			want: "refused the stored credential",
		},
		{
			name: "an unreachable host lists both causes",
			err:  errors.New(`Get "https://api.testkube.io/organizations/tkcorg_1": dial tcp: no such host`),
			want: "set and correct for this Control Plane",
		},
		{
			name: "an unrecognised status lists both causes",
			err:  &cloudclient.StatusError{StatusCode: 500},
			want: "set and correct for this Control Plane",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Contains(t, orgEnvLookupHint(tt.err), tt.want)
		})
	}
}
