package triggers

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Event struct {
	Resource  ResourceType
	Name      string
	Namespace string
	Labels    map[string]string
	Object    runtime.Object
	Type      EventType
	Causes    []Cause
}

func newPodEvent(eventType EventType, pod *corev1.Pod) *Event {
	return &Event{
		Resource:  ResourcePod,
		Name:      pod.Name,
		Namespace: pod.Namespace,
		Labels:    pod.Labels,
		Object:    pod,
		Type:      eventType,
	}
}

func newDeploymentEvent(deployment *appsv1.Deployment, eventType EventType, causes []Cause) *Event {
	return &Event{
		Resource:  ResourceDeployment,
		Name:      deployment.Name,
		Namespace: deployment.Namespace,
		Labels:    deployment.Labels,
		Object:    deployment,
		Type:      eventType,
		Causes:    causes,
	}
}
