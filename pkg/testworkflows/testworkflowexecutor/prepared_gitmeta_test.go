package testworkflowexecutor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
)

// TestStoreGitMetadata pins that git provenance survives StoreConfig's filter.
//
// StoreConfig keeps only the parameters a workflow declared, which is the right rule for
// values the workflow asked for and the wrong one for this: the dependency cache reads
// TESTKUBE_GIT_PR_NUMBER to decide which namespace a run may write, and a missing marker
// means "trusted". Dropping it because nobody declared it would hand a pull request's run
// the namespace the default branch restores from.
func TestStoreGitMetadata(t *testing.T) {
	// A workflow that declares one ordinary parameter and none of the git keys, which is
	// what every workflow looks like - nothing declares them.
	newExecution := func() *IntermediateExecution {
		execution := NewIntermediateExecution()
		execution.cr = &testworkflowsv1.TestWorkflow{
			Spec: testworkflowsv1.TestWorkflowSpec{
				TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
					Config: map[string]testworkflowsv1.ParameterSchema{
						"suite": {Type: testworkflowsv1.ParameterTypeString},
					},
				},
			},
		}
		return execution
	}

	triggerConfig := map[string]string{
		"suite":                    "smoke",
		"TESTKUBE_GIT_PR_NUMBER":   "42",
		"TESTKUBE_GIT_PR_HEAD_REF": "feature/login",
		"TESTKUBE_GIT_COMMIT":      "abc123",
	}

	t.Run("an undeclared git key is kept", func(t *testing.T) {
		execution := newExecution()

		// The precondition the whole method exists for. If this ever stops holding, the
		// method is redundant rather than wrong - but silently keeping it would hide that.
		execution.StoreConfig(triggerConfig)
		require.NotContains(t, execution.execution.ConfigParams, "TESTKUBE_GIT_PR_NUMBER",
			"StoreConfig is expected to drop undeclared parameters")

		execution.StoreGitMetadata(triggerConfig)

		assert.Equal(t, "42", execution.execution.ConfigParams["TESTKUBE_GIT_PR_NUMBER"].Value)
		assert.Equal(t, "feature/login", execution.execution.ConfigParams["TESTKUBE_GIT_PR_HEAD_REF"].Value)
		assert.Equal(t, "abc123", execution.execution.ConfigParams["TESTKUBE_GIT_COMMIT"].Value)
	})

	t.Run("the declared configuration is left alone", func(t *testing.T) {
		execution := newExecution()
		execution.StoreConfig(triggerConfig)
		execution.StoreGitMetadata(triggerConfig)

		assert.Equal(t, "smoke", execution.execution.ConfigParams["suite"].Value,
			"storing git metadata must not drop what StoreConfig recorded")
	})

	t.Run("it works when the configuration was never stored", func(t *testing.T) {
		// StoreConfig is skipped above a size limit. That path must still record the
		// metadata, because skipping it reads as "trusted".
		execution := newExecution()
		execution.StoreGitMetadata(triggerConfig)

		require.NotNil(t, execution.execution.ConfigParams)
		assert.Equal(t, "42", execution.execution.ConfigParams["TESTKUBE_GIT_PR_NUMBER"].Value)
		assert.NotContains(t, execution.execution.ConfigParams, "suite")
	})

	t.Run("a run with no git metadata records none", func(t *testing.T) {
		execution := newExecution()
		execution.StoreGitMetadata(map[string]string{
			"suite":                  "smoke",
			"TESTKUBE_GIT_PR_NUMBER": "",
		})

		assert.Empty(t, execution.execution.ConfigParams,
			"an empty value is not a marker, and a non-git key is not one either")
	})
}
