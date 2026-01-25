package k8sevent

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/mapper/k8sevents"
)

var _ common.Listener = (*K8sEventListener)(nil)

func NewK8sEventListener(name, selector, defaultNamespace string, events []testkube.EventType, clientset kubernetes.Interface) *K8sEventListener {
	return &K8sEventListener{
		name:             name,
		Log:              log.DefaultLogger,
		selector:         selector,
		events:           events,
		clientset:        clientset,
		defaultNamespace: defaultNamespace,
	}
}

type K8sEventListener struct {
	name             string
	Log              *zap.SugaredLogger
	events           []testkube.EventType
	selector         string
	clientset        kubernetes.Interface
	defaultNamespace string
}

func (l *K8sEventListener) Name() string {
	return l.name
}

func (l *K8sEventListener) Selector() string {
	return l.selector
}

func (l *K8sEventListener) Events() []testkube.EventType {
	return l.events
}
func (l *K8sEventListener) Metadata() map[string]string {
	return map[string]string{
		"name":     l.Name(),
		"events":   fmt.Sprintf("%v", l.Events()),
		"selector": l.Selector(),
	}
}

func (l *K8sEventListener) Match(event testkube.Event) bool {
	_, valid := event.Valid(l.Group(), l.Selector(), l.Events())
	return valid
}

func (l *K8sEventListener) Notify(event testkube.Event) (result testkube.EventResult) {
	ev := k8sevents.MapAPIToCRD(event, l.defaultNamespace, time.Now())
	eventsClient := l.clientset.CoreV1().Events(l.defaultNamespace)
	if _, err := eventsClient.Create(context.Background(), &ev, metav1.CreateOptions{}); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return testkube.NewSuccessEventResult(event.Id, "event already exists")
		}
		return testkube.NewFailedEventResult(event.Id, err)
	}

	return testkube.NewSuccessEventResult(event.Id, "event sent to K8s")
}

func (l *K8sEventListener) Kind() string {
	return "k8sevent"
}

func (l *K8sEventListener) Group() string {
	return ""
}
