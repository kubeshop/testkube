package testkube

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestWorkflowStatusDetails_Clone(t *testing.T) {
	t.Run("nil stays nil", func(t *testing.T) {
		var details *TestWorkflowStatusDetails
		assert.Nil(t, details.Clone())
	})

	t.Run("the copy does not share the user", func(t *testing.T) {
		details := &TestWorkflowStatusDetails{
			Type_:  string(StatusDetailsTypeUserCancel),
			Reason: string(StopReasonUserCancel),
			Actor:  string(StopActorUser),
			User:   &TestWorkflowStatusDetailsUser{Name: "Ada", Email: "ada@example.com"},
		}

		clone := details.Clone()
		require.NotNil(t, clone.User)
		assert.Equal(t, *details, *clone)

		clone.User.Name = "Grace"
		assert.Equal(t, "Ada", details.User.Name)
	})
}
