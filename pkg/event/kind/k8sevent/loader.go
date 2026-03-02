package k8sevent

import (
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/log"
)

var _ common.ListenerLoader = (*K8sEventLoader)(nil)

func NewK8sEventLoader(clientset kubernetes.Interface, defaultNamespace string, events []testkube.EventType) *K8sEventLoader {
	return &K8sEventLoader{
		Log:              log.DefaultLogger,
		events:           events,
		clientset:        clientset,
		defaultNamespace: defaultNamespace,
	}
}

// K8sEventLoader is a reconciler for k8s events for now it returns single listener for k8s events
type K8sEventLoader struct {
	Log              *zap.SugaredLogger
	events           []testkube.EventType
	clientset        kubernetes.Interface
	defaultNamespace string
}

func (r K8sEventLoader) Kind() string {
	return "k8sevent"
}

// Load returns single listener for k8s event
func (r *K8sEventLoader) Load() (listeners common.Listeners, err error) {
	// TODO(emil): this is a static list it does not seem to need a loader, it can just use a Register
	return common.Listeners{NewK8sEventListener("k8sevent", "", r.defaultNamespace, r.events, r.clientset)}, nil
}
