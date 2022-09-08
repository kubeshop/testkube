package event

import (
	"fmt"
	"testing"

	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/stretchr/testify/assert"
)

type DummyLoader struct {
	Err error
}

func (r DummyLoader) Kind() string {
	return "dummy"
}

func (r *DummyLoader) Load() (common.Listeners, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	return common.Listeners{
		&DummyListener{},
		&DummyListener{},
	}, nil
}

func TestLoader_UpdateListeners(t *testing.T) {

	t.Run("reconcile updates listeners list based on registered reconcilers", func(t *testing.T) {
		// given reconciler with two registered reconcilers that return two listeners each
		reconciler := NewLoader()
		reconciler.Register(&DummyLoader{})
		reconciler.Register(&DummyLoader{})

		// when
		listeners := reconciler.Reconcile()

		// then there should be 4 listeners
		assert.Len(t, listeners, 4)
	})

	t.Run("reconcile updates listeners list based on registered reconcilers thread safe", func(t *testing.T) {
		// given reconciler with two registered reconcilers that return two listeners each
		reconciler := NewLoader()
		reconciler.Register(&DummyLoader{})
		reconciler.Register(&DummyLoader{})

		// when
		listeners := reconciler.Reconcile()

		// then there should be 4 listeners
		assert.Len(t, listeners, 4)
	})

	t.Run("failed loaders are omited", func(t *testing.T) {
		// given reconciler with two registered reconcilers that return two listeners each
		reconciler := NewLoader()
		reconciler.Register(&DummyLoader{Err: fmt.Errorf("loader error")})
		reconciler.Register(&DummyLoader{})

		// when
		listeners := reconciler.Reconcile()

		// then there should be 2 listeners
		assert.Len(t, listeners, 2)
	})

}
