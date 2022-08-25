package event

import (
	"testing"

	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/stretchr/testify/assert"
)

type DummyReconciler struct {
}

func (r DummyReconciler) Kind() string {
	return "dummy"
}

func (r DummyReconciler) Load() ([]common.Listener, error) {
	return []common.Listener{
		&DummyListener{},
		&DummyListener{},
	}, nil
}

func TestReconciler_Reconcile(t *testing.T) {

	t.Run("reconcile updates listeners list based on registered reconcilers", func(t *testing.T) {
		// given reconciler with two registered reconcilers that return two listeners each
		reconciler := Reconciler{}
		reconciler.Register(&DummyReconciler{})
		reconciler.Register(&DummyReconciler{})

		// when
		listeners := reconciler.Reconcile()

		// then there should be 4 listeners
		assert.Len(t, listeners, 4)
	})

}
