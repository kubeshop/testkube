package data

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

// TestRerunExecutionId covers the reference every rerun feature hangs off.
//
// The case that matters most is the original run: lineage is recorded on every
// execution, so a naive "lineage is present, use it" would hand back the chain
// root and make execution("rerun") resolve to the run itself instead of failing.
func TestRerunExecutionId(t *testing.T) {
	tests := []struct {
		name     string
		lineage  *testworkflowconfig.LineageConfig
		expected string
	}{
		{
			name:     "an original run is not a rerun of anything",
			lineage:  &testworkflowconfig.LineageConfig{RootId: "exec-1", Attempt: 1},
			expected: "",
		},
		{
			name:     "a rerun resolves to its base",
			lineage:  &testworkflowconfig.LineageConfig{BaseId: "exec-1", RootId: "exec-1", Attempt: 2},
			expected: "exec-1",
		},
		{
			name:     "a rerun of a rerun resolves to its immediate base",
			lineage:  &testworkflowconfig.LineageConfig{BaseId: "exec-2", RootId: "exec-1", Attempt: 3},
			expected: "exec-2",
		},
		{
			name:     "no lineage recorded",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := GetState()
			state.InternalConfig.Execution.Lineage = tt.lineage
			t.Cleanup(func() { state.InternalConfig.Execution.Lineage = nil })

			assert.Equal(t, tt.expected, RerunExecutionId())
		})
	}
}
