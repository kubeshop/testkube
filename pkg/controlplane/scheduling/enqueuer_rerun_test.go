package scheduling

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/cloud"
)

// narrowsTestCases decides whether the workflow has to support narrowing. An
// execution id alone does not ask for it: it is recorded so the pod can resolve
// the "rerun" reference, which a workflow may or may not use.
func TestNarrowsTestCases(t *testing.T) {
	for _, tc := range []struct {
		name  string
		rerun *cloud.RerunPolicy
		want  bool
	}{
		{"no policy", nil, false},
		{"an execution id alone", &cloud.RerunPolicy{ExecutionId: "exec-1"}, false},
		{"only failed", &cloud.RerunPolicy{ExecutionId: "exec-1", OnlyFailed: true}, true},
		{"named test cases", &cloud.RerunPolicy{TestCases: []string{"a"}}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, narrowsTestCases(tc.rerun))
		})
	}
}
