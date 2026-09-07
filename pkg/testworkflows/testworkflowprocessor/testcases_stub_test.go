package testworkflowprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

func TestStubTestCases(t *testing.T) {
	t.Run("refuses a step that declares a policy", func(t *testing.T) {
		// This is the whole cloud-only gate. It refuses rather than ignoring the
		// block: silently dropping a mute policy keeps a pipeline red that its
		// author declared green, which is a wrong answer delivered confidently.
		stage, err := StubTestCases(nil, nil, nil, testworkflowsv1.Step{
			TestCases: &testworkflowsv1.StepTestCases{
				Report: &testworkflowsv1.TestCaseReport{Paths: []string{"junit.xml"}},
			},
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, constants.ErrOpenSourceTestCasesOperationIsNotAvailable)
		assert.Contains(t, err.Error(), `"testCases"`)
		assert.Nil(t, stage)
	})

	t.Run("passes through a step without one", func(t *testing.T) {
		stage, err := StubTestCases(nil, nil, nil, testworkflowsv1.Step{})
		require.NoError(t, err)
		assert.Nil(t, stage, "the stub contributes no stage of its own")
	})
}
