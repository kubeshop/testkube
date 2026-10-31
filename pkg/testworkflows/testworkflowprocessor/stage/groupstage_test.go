package stage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestContainerStage(ref string) ContainerStage {
	return NewContainerStage(ref, NewContainer())
}

// A step's id decides which stage prepares and rescans the outputs directory, so it has
// to reach a stage even when the step expands into several of them.
func TestGroupStageFlattenPropagatesId(t *testing.T) {
	t.Run("single child, group merged away", func(t *testing.T) {
		group := NewGroupStage("group-ref", false)
		child := newTestContainerStage("child-ref")
		group.SetId("mystep")
		group.Add(child)

		flattened := group.Flatten()
		require.Len(t, flattened, 1)
		assert.Equal(t, "mystep", flattened[0].Id())
	})

	t.Run("operation plus artifacts, group kept", func(t *testing.T) {
		// This is the shell-plus-artifacts shape: two stages, so the group is not
		// merged. The id used to stay on the group, leaving both stages without one.
		group := NewGroupStage("group-ref", false)
		operation := newTestContainerStage("operation-ref")
		artifacts := newTestContainerStage("artifacts-ref")
		group.SetId("mystep")
		group.Add(operation, artifacts)

		flattened := group.Flatten()
		require.Len(t, flattened, 1, "the group survives rather than merging into a child")

		assert.Equal(t, "mystep", operation.Id(), "the operation owns the step's outputs")
		assert.Empty(t, artifacts.Id(),
			"a trailing stage must not share the id, or it would clear what the operation published")
	})

	t.Run("a child with its own id keeps it", func(t *testing.T) {
		group := NewGroupStage("group-ref", false)
		operation := newTestContainerStage("operation-ref")
		operation.SetId("own")
		artifacts := newTestContainerStage("artifacts-ref")
		group.SetId("mystep")
		group.Add(operation, artifacts)

		group.Flatten()
		assert.Equal(t, "own", operation.Id())
	})
}
