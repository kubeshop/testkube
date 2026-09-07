package instructions

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// TestTestResultsInstructionRoundTrip covers the whole channel the test-case
// counters travel down: the init process prints them as a log line, and the
// controller watching the pod parses that line back into the same struct.
//
// It is worth testing end to end because the two halves live in different
// packages and only agree through JSON field names. A rename on either side
// would silently drop the counters rather than fail to compile.
func TestTestResultsInstructionRoundTrip(t *testing.T) {
	sent := testkube.TestWorkflowStepTestResults{
		Tests:                1043,
		Passed:               1035,
		Failed:               8,
		Muted:                8,
		Unexpected:           0,
		Tolerated:            true,
		RequirementApplied:   true,
		IdentitiesIncomplete: false,
		Unrepresented:        0,
		UnusedMutePatterns:   []string{"test_retired_*", "Tests.Old/**"},
	}

	line := SprintHintDetails("step-ref", constants.InstructionTestResults, sent)

	// The emitted text is a log line, so it has to survive being read as one.
	trimmed := strings.TrimSpace(line)
	require.True(t, MayBeInstruction([]byte(trimmed)), "the line must be recognised as an instruction")

	instruction, isHint, err := DetectInstruction([]byte(trimmed))
	require.NoError(t, err)
	require.True(t, isHint, "details are emitted as a hint")
	require.NotNil(t, instruction)
	assert.Equal(t, "step-ref", instruction.Ref)
	assert.Equal(t, constants.InstructionTestResults, instruction.Name)

	// This mirrors exactly what the notifier does with hint.Value.
	serialized, err := json.Marshal(instruction.Value)
	require.NoError(t, err)
	var received testkube.TestWorkflowStepTestResults
	require.NoError(t, json.Unmarshal(serialized, &received))

	assert.Equal(t, sent, received, "every counter has to survive the round trip")
}

func TestTestResultsInstructionCarriesZeroValues(t *testing.T) {
	// A step where nothing failed still reports, because muted must never mean
	// silent - and omitempty means the zeroes vanish from the wire. What must
	// survive is that the struct is present at all, with the totals intact.
	sent := testkube.TestWorkflowStepTestResults{Tests: 12, Passed: 12}

	line := strings.TrimSpace(SprintHintDetails("r", constants.InstructionTestResults, sent))
	instruction, _, err := DetectInstruction([]byte(line))
	require.NoError(t, err)

	serialized, err := json.Marshal(instruction.Value)
	require.NoError(t, err)
	var received testkube.TestWorkflowStepTestResults
	require.NoError(t, json.Unmarshal(serialized, &received))

	assert.Equal(t, int32(12), received.Tests)
	assert.Equal(t, int32(12), received.Passed)
	assert.Zero(t, received.Muted)
	assert.False(t, received.Tolerated)
}
