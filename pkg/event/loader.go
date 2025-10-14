package event

import (
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/log"
)

func NewLoader() *loader {
	return &loader{
		Log:     log.DefaultLogger,
		Loaders: make([]common.ListenerLoader, 0),
	}
}

// loader updates list of available listeners in the background as we don't want to load them on each event
type loader struct {
	Log     *zap.SugaredLogger
	Loaders []common.ListenerLoader
}

// RegisterLoader registers new loader
// TODO(emil): check usage of this, it seems there are cases where this is not needed as we are loading a static set in that case it is better to use register directly on the emitter
func (s *loader) RegisterLoader(loader common.ListenerLoader) {
	s.Loaders = append(s.Loaders, loader)
}

// Reconcile loop for reconciling listeners from different sources
func (s *loader) Reconcile() (listeners common.Listeners) {
	listeners = make(common.Listeners, 0)
	for _, loader := range s.Loaders {
		l, err := loader.Load()
		log.Tracef(s.Log, "Got listeners from loader %T %+v\n", loader, l)

		if err != nil {
			s.Log.Errorw("error loading listeners", "error", err)
			continue
		}
		listeners = append(listeners, l...)
	}

	return listeners
}
