package triggers

import (
	"github.com/kubeshop/testkube-operator/pkg/validation/tests/v1/testtrigger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type watcherEvent struct {
	resource  testtrigger.ResourceType
	name      string
	namespace string
	labels    map[string]string
	object    metav1.Object
	eventType testtrigger.EventType
	causes    []testtrigger.Cause
}

func newWatcherEvent(eventType testtrigger.EventType, object metav1.Object, causes []testtrigger.Cause) *watcherEvent {
	return &watcherEvent{
		resource:  testtrigger.ResourcePod,
		name:      object.GetName(),
		namespace: object.GetNamespace(),
		labels:    object.GetLabels(),
		object:    object,
		eventType: eventType,
		causes: causes,
	}
}
