package executiondata

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// The resolver is shared by execution() and by the `fetch` block, so a reference means the
// same thing in both. Before it was shared, `fetch` looked only in the local registry and
// refused anything else - including the sibling id a parent hands down as configuration,
// which read_artifact() accepted from the same workflow.
func TestResolver(t *testing.T) {
	t.Run("prefers the registry, without asking the control plane", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		// No EXPECT: a call to the control plane fails the test.
		repository := NewMockExecutionRepository(ctrl)

		registry := NewRegistry()
		registry.Add(Execution{Id: "exec-1", Workflow: "producer", Alias: "p", Index: 0})
		resolver := Resolver{Registry: registry, Repository: repository}

		execution, err := resolver.Resolve(context.Background(), "p", 0)
		require.NoError(t, err)
		assert.Equal(t, "exec-1", execution.Id)
	})

	t.Run("falls back to the control plane for an id it did not schedule", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repository := NewMockExecutionRepository(ctrl)
		repository.EXPECT().Get(gomock.Any(), "sibling-1").
			Return(Execution{Id: "sibling-1", Workflow: "producer"}, nil)

		// An empty registry is the consumer's situation: it ran no test workflow itself,
		// and reaches its sibling by the id the suite passed down.
		resolver := Resolver{Registry: NewRegistry(), Repository: repository}

		execution, err := resolver.Resolve(context.Background(), "sibling-1", 0)
		require.NoError(t, err)
		assert.Equal(t, "sibling-1", execution.Id)
	})

	t.Run("resolves the parent reference to the closest ancestor", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repository := NewMockExecutionRepository(ctrl)
		repository.EXPECT().Get(gomock.Any(), "parent-2").
			Return(Execution{Id: "parent-2", Workflow: "suite"}, nil)

		resolver := Resolver{Repository: repository, ParentIds: []string{"parent-1", "parent-2"}}

		execution, err := resolver.Resolve(context.Background(), ParentRef, 0)
		require.NoError(t, err)
		assert.Equal(t, "parent-2", execution.Id)
		assert.Equal(t, ParentRef, execution.Alias, "the reference it was reached by is kept")
	})

	t.Run("passes the caller's context down", func(t *testing.T) {
		type key struct{}
		ctx := context.WithValue(context.Background(), key{}, "carried")

		ctrl := gomock.NewController(t)
		repository := NewMockExecutionRepository(ctrl)
		repository.EXPECT().Get(gomock.Any(), "exec-1").
			DoAndReturn(func(got context.Context, _ string) (Execution, error) {
				assert.Equal(t, "carried", got.Value(key{}), "a cancelled fetch must cancel the read")
				return Execution{Id: "exec-1"}, nil
			})

		resolver := Resolver{Repository: repository}
		_, err := resolver.Resolve(ctx, "exec-1", 0)
		require.NoError(t, err)
	})

	t.Run("reports what it knows when a reference resolves to nothing", func(t *testing.T) {
		registry := NewRegistry()
		registry.Add(Execution{Id: "exec-1", Workflow: "producer", Alias: "p"})

		t.Run("with no control plane to ask", func(t *testing.T) {
			resolver := Resolver{Registry: registry}
			_, err := resolver.Resolve(context.Background(), "nope", 0)
			require.Error(t, err)
			assert.Contains(t, err.Error(), `unknown execution "nope"`)
			assert.Contains(t, err.Error(), "available executions are p")
		})

		t.Run("when the control plane has no such execution", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repository := NewMockExecutionRepository(ctrl)
			repository.EXPECT().Get(gomock.Any(), "nope").Return(Execution{}, nil)

			resolver := Resolver{Registry: registry, Repository: repository}
			_, err := resolver.Resolve(context.Background(), "nope", 0)
			assert.ErrorContains(t, err, `unknown execution "nope"`)
		})

		t.Run("when the read fails", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repository := NewMockExecutionRepository(ctrl)
			repository.EXPECT().Get(gomock.Any(), "nope").Return(Execution{}, errors.New("unreachable"))

			resolver := Resolver{Registry: registry, Repository: repository}
			_, err := resolver.Resolve(context.Background(), "nope", 0)
			assert.ErrorContains(t, err, `reading execution "nope"`)
			assert.ErrorContains(t, err, "unreachable")
		})

		t.Run("when this execution has no parent", func(t *testing.T) {
			resolver := Resolver{Registry: registry}
			_, err := resolver.Resolve(context.Background(), ParentRef, 0)
			assert.ErrorContains(t, err, "this execution has no parent")
		})

		t.Run("when an index is asked of the control plane", func(t *testing.T) {
			// Positions exist only within the local registry; the control plane answers
			// about one execution addressed by its id.
			ctrl := gomock.NewController(t)
			repository := NewMockExecutionRepository(ctrl)

			resolver := Resolver{Registry: registry, Repository: repository}
			_, err := resolver.Resolve(context.Background(), "exec-9", 2)
			assert.ErrorContains(t, err, `unknown execution "exec-9" at index 2`)
		})
	})
}
