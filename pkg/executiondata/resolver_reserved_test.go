package executiondata

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// A reserved reference must not be shadowed by something the workflow executed.
//
// Resolving the registry first would hand the step the child instead of the
// execution the reserved word names, and nothing would say so - the workflow
// would read the wrong execution's outputs and carry on.
func TestResolverReservedRefsAreNotShadowed(t *testing.T) {
	for _, tc := range []struct {
		name     string
		ref      string
		resolver func(repository ExecutionRepository, registry *Registry) Resolver
	}{
		{
			name: "rerun shadowed by an alias",
			ref:  RerunRef,
			resolver: func(repository ExecutionRepository, registry *Registry) Resolver {
				return Resolver{Registry: registry, Repository: repository, RerunId: "base-1"}
			},
		},
		{
			name: "parent shadowed by an alias",
			ref:  ParentRef,
			resolver: func(repository ExecutionRepository, registry *Registry) Resolver {
				return Resolver{Registry: registry, Repository: repository, ParentIds: []string{"parent-1"}}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			// No EXPECT: the collision is refused before anything is fetched.
			repository := NewMockExecutionRepository(ctrl)

			registry := NewRegistry()
			registry.Add(Execution{Id: "child-1", Workflow: "some-workflow", Alias: tc.ref})

			_, err := tc.resolver(repository, registry).Resolve(context.Background(), tc.ref, 0)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "reserved execution reference")
			assert.Contains(t, err.Error(), "different 'as' alias")
		})
	}
}

// The same holds when the collision comes from the workflow's own name rather
// than an alias the author chose - they may not control that name at all.
func TestResolverReservedRefShadowedByWorkflowName(t *testing.T) {
	ctrl := gomock.NewController(t)
	repository := NewMockExecutionRepository(ctrl)

	registry := NewRegistry()
	registry.Add(Execution{Id: "child-1", Workflow: RerunRef})

	resolver := Resolver{Registry: registry, Repository: repository, RerunId: "base-1"}
	_, err := resolver.Resolve(context.Background(), RerunRef, 0)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "reserved execution reference")
}

// Without a collision the reserved reference resolves through the control plane
// as before, and is labelled with the reserved name rather than the child's.
func TestResolverReservedRefResolvesWithoutACollision(t *testing.T) {
	ctrl := gomock.NewController(t)
	repository := NewMockExecutionRepository(ctrl)
	repository.EXPECT().Get(gomock.Any(), "base-1").
		Return(Execution{Id: "base-1", Workflow: "suite"}, nil)

	registry := NewRegistry()
	registry.Add(Execution{Id: "child-1", Workflow: "producer", Alias: "p"})

	resolver := Resolver{Registry: registry, Repository: repository, RerunId: "base-1"}
	execution, err := resolver.Resolve(context.Background(), RerunRef, 0)

	require.NoError(t, err)
	assert.Equal(t, "base-1", execution.Id)
	assert.Equal(t, RerunRef, execution.Alias)
}

// An ordinary reference still prefers the registry - the reserved-first rule
// must not cost the no-network path for everything else.
func TestResolverOrdinaryRefStillPrefersTheRegistry(t *testing.T) {
	ctrl := gomock.NewController(t)
	// No EXPECT: a call to the control plane fails the test.
	repository := NewMockExecutionRepository(ctrl)

	registry := NewRegistry()
	registry.Add(Execution{Id: "child-1", Workflow: "producer", Alias: "p"})

	resolver := Resolver{Registry: registry, Repository: repository, RerunId: "base-1"}
	execution, err := resolver.Resolve(context.Background(), "p", 0)

	require.NoError(t, err)
	assert.Equal(t, "child-1", execution.Id)
}

func TestIsReservedRef(t *testing.T) {
	assert.True(t, IsReservedRef(ParentRef))
	assert.True(t, IsReservedRef(RerunRef))
	assert.False(t, IsReservedRef("p"))
	assert.False(t, IsReservedRef(""))
}
