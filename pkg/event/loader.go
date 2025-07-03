package event

import (
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/log"
)

// NOTE: this is an event emitter loader which register "listener" loaders
func NewLoader() *Loader {
	return &Loader{
		Log:     log.DefaultLogger,
		Loaders: make([]common.ListenerLoader, 0),
	}
}

// Loader updates list of available listeners in the background as we don't want to load them on each event
type Loader struct {
	Log     *zap.SugaredLogger
	Loaders []common.ListenerLoader
}

// Register registers new listener reconciler
func (s *Loader) Register(loader common.ListenerLoader) {
	// NOTE: webhook loader registered here
	s.Loaders = append(s.Loaders, loader)
}

// Reconcile loop for reconciling listeners from different sources
func (s *Loader) Reconcile() (listeners common.Listeners) {
	listeners = make(common.Listeners, 0)
	for _, loader := range s.Loaders {
		// NOTE: this loads all the webhook cr
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
