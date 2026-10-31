package executiondata

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestTranslateUnsupported(t *testing.T) {
	t.Run("names the missing capability", func(t *testing.T) {
		// A control plane that predates the RPC answers Unimplemented, which on its own
		// says nothing about what the workflow was trying to do.
		err := translateUnsupported(status.Error(codes.Unimplemented, "unknown method ListExecutionArtifactsPresigned"))
		assert.ErrorContains(t, err, "cannot grant access to another execution's artifacts")
		assert.ErrorContains(t, err, "tw-artifact-read")
		assert.ErrorContains(t, err, "exchange the data as outputs instead")
		assert.Equal(t, codes.Unimplemented, status.Code(err), "the original status stays wrapped for callers that check it")
	})

	t.Run("leaves other failures alone", func(t *testing.T) {
		original := status.Error(codes.PermissionDenied, "execution unreachable")
		assert.Equal(t, original, translateUnsupported(original))

		plain := errors.New("connection refused")
		assert.Equal(t, plain, translateUnsupported(plain))
	})

	t.Run("passes nil through", func(t *testing.T) {
		assert.NoError(t, translateUnsupported(nil))
	})
}
