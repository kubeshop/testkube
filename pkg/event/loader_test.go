package event

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/event/kind/dummy"
)

func TestLoader_UpdateListeners(t *testing.T) {

	t.Run("reconcile updates listeners list based on registered reconcilers", func(t *testing.T) {
		// given reconciler with two registered reconcilers that return two listeners each
		reconciler := NewLoader()
		reconciler.RegisterLoader(&dummy.DummyLoader{IdPrefix: "dummy1"})
		reconciler.RegisterLoader(&dummy.DummyLoader{IdPrefix: "dummy2"})

		// when
		listeners := reconciler.Reconcile()

		// then there should be 4 listeners
		assert.Len(t, listeners, 4)
	})

	t.Run("reconcile updates listeners list based on registered reconcilers thread safe", func(t *testing.T) {
		// given reconciler with two registered reconcilers that return two listeners each
		reconciler := NewLoader()
		reconciler.RegisterLoader(&dummy.DummyLoader{})
		reconciler.RegisterLoader(&dummy.DummyLoader{})

		// when
		listeners := reconciler.Reconcile()

		// then there should be 4 listeners
		assert.Len(t, listeners, 4)
	})

	t.Run("failed loaders are omited", func(t *testing.T) {
		// given reconciler with two registered reconcilers that return two listeners each
		reconciler := NewLoader()
		reconciler.RegisterLoader(&dummy.DummyLoader{Err: fmt.Errorf("loader error")})
		reconciler.RegisterLoader(&dummy.DummyLoader{})

		// when
		listeners := reconciler.Reconcile()

		// then there should be 2 listeners
		assert.Len(t, listeners, 2)
	})

}
