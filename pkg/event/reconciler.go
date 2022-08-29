package event

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/log"
	"go.uber.org/zap"
)

func NewReconciler() *Reconciler {
	return &Reconciler{
		Log: log.DefaultLogger,
	}
}

// Reconciler updates list of available listeners in the background as we don't want to load them on each event
type Reconciler struct {
	Log         *zap.SugaredLogger
	Reconcilers []common.ListenerReconiler
}

// Register registers new listener reconciler
func (s *Reconciler) Register(reconciler common.ListenerReconiler) {
	s.Reconcilers = append(s.Reconcilers, reconciler)
}

func (s *Reconciler) Reconcile() (listeners []common.Listener) {
	listeners = make([]common.Listener, 0)
	for _, reconciler := range s.Reconcilers {
		l, err := reconciler.Load()
		if err != nil {
			fmt.Printf("%+v\n", err)
			fmt.Printf("%+v\n", reconciler)

			s.Log.Errorw("error loading listeners", "kind", reconciler.Kind(), "error", err)
			continue
		}
		listeners = append(listeners, l...)
	}

	return listeners
}
