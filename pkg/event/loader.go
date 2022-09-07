package event

import (
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/log"
	"go.uber.org/zap"
)

func NewLoader() *Loader {
	return &Loader{
		Log: log.DefaultLogger,
	}
}

// Loader updates list of available listeners in the background as we don't want to load them on each event
type Loader struct {
	Log     *zap.SugaredLogger
	Loaders []common.ListenerLoader
}

// Register registers new listener reconciler
func (s *Loader) Register(reconciler common.ListenerLoader) {
	s.Loaders = append(s.Loaders, reconciler)
}

// Reconcile loop for reconciling listeners from different sources
func (s *Loader) Reconcile() (listeners common.Listeners) {
	listeners = make(common.Listeners, 0)
	for _, reconciler := range s.Loaders {
		l, err := reconciler.Load()
		if err != nil {
			s.Log.Errorw("error loading listeners", "kind", reconciler.Kind(), "error", err)
			continue
		}
		listeners = append(listeners, l...)
	}

	return listeners
}
