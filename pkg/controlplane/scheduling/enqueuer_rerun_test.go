package scheduling

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/cloud"
)

func TestValidateRerunPolicy(t *testing.T) {
	t.Run("an ordinary selection is carried", func(t *testing.T) {
		require.NoError(t, validateRerunPolicy(&cloud.RerunPolicy{
			ExecutionId: "exec-1",
			OnlyFailed:  true,
			TestCases:   []string{"tests.a::test_one", "tests.b::test_two"},
		}))
	})

	t.Run("a reference without names is fine", func(t *testing.T) {
		// The selection is resolved in the pod from the referenced execution's
		// report, so naming nothing here is the normal case.
		require.NoError(t, validateRerunPolicy(&cloud.RerunPolicy{ExecutionId: "exec-1", OnlyFailed: true}))
	})

	t.Run("too many test cases is refused, not truncated", func(t *testing.T) {
		// Truncating would be the dangerous option: a caller who asked for 5000
		// specific tests and silently got 1000 would read the result as those
		// tests having passed.
		cases := make([]string, MaxRerunTestCases+1)
		for i := range cases {
			cases[i] = "t"
		}

		err := validateRerunPolicy(&cloud.RerunPolicy{TestCases: cases})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "more than the 1000 that can be carried")
		assert.Contains(t, err.Error(), "testCases.select",
			"the error has to name the way that does work")
	})

	t.Run("too many bytes is refused", func(t *testing.T) {
		// Few enough names to pass the count check, but far too long: the list
		// is inlined into the ConfigMap the pod reads, which has its own limit.
		cases := []string{strings.Repeat("x", MaxRerunTestCaseBytes+1)}

		err := validateRerunPolicy(&cloud.RerunPolicy{TestCases: cases})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bytes of test case names")
	})

	t.Run("exactly at the limits is allowed", func(t *testing.T) {
		cases := make([]string, MaxRerunTestCases)
		for i := range cases {
			cases[i] = "t"
		}
		require.NoError(t, validateRerunPolicy(&cloud.RerunPolicy{TestCases: cases}))
	})
}
