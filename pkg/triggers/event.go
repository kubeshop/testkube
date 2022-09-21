package triggers

import (
	"github.com/kubeshop/testkube-operator/pkg/validation/tests/v1/testtrigger"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type event struct {
	resource  testtrigger.ResourceType
	name      string
	namespace string
	labels    map[string]string
	object    runtime.Object
	eventType testtrigger.EventType
	causes    []testtrigger.Cause
}

func newPodEvent(eventType testtrigger.EventType, pod *corev1.Pod) *event {
	return &event{
		resource:  testtrigger.ResourcePod,
		name:      pod.Name,
		namespace: pod.Namespace,
		labels:    pod.Labels,
		object:    pod,
		eventType: eventType,
	}
}

func newDeploymentEvent(deployment *appsv1.Deployment, eventType testtrigger.EventType, causes []testtrigger.Cause) *event {
	return &event{
		resource:  testtrigger.ResourceDeployment,
		name:      deployment.Name,
		namespace: deployment.Namespace,
		labels:    deployment.Labels,
		object:    deployment,
		eventType: eventType,
		causes:    causes,
	}
}
