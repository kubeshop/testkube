package triggers

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type event struct {
	resource  ResourceType
	name      string
	namespace string
	labels    map[string]string
	object    runtime.Object
	eventType EventType
	causes    []Cause
}

func newPodEvent(eventType EventType, pod *corev1.Pod) *event {
	return &event{
		resource:  ResourcePod,
		name:      pod.Name,
		namespace: pod.Namespace,
		labels:    pod.Labels,
		object:    pod,
		eventType: eventType,
	}
}

func newDeploymentEvent(deployment *appsv1.Deployment, eventType EventType, causes []Cause) *event {
	return &event{
		resource:  ResourceDeployment,
		name:      deployment.Name,
		namespace: deployment.Namespace,
		labels:    deployment.Labels,
		object:    deployment,
		eventType: eventType,
		causes:    causes,
	}
}
