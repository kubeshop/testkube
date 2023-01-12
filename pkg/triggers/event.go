package triggers

import (
	"context"

	testtriggersv1 "github.com/kubeshop/testkube-operator/apis/testtriggers/v1"
	"github.com/kubeshop/testkube-operator/pkg/validation/tests/v1/testtrigger"
	"github.com/kubeshop/testkube/pkg/mapper/deployments"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

type getConditions func() ([]testtriggersv1.TestTriggerCondition, error)

type watcherEvent struct {
	resource        testtrigger.ResourceType
	name            string
	namespace       string
	labels          map[string]string
	object          runtime.Object
	eventType       testtrigger.EventType
	causes          []testtrigger.Cause
	fnGetConditions getConditions
}

func newPodEvent(eventType testtrigger.EventType, pod *corev1.Pod) *watcherEvent {
	return &watcherEvent{
		resource:  testtrigger.ResourcePod,
		name:      pod.Name,
		namespace: pod.Namespace,
		labels:    pod.Labels,
		object:    pod,
		eventType: eventType,
	}
}

func getDeploymentConditions(name, namespace string, clientset kubernetes.Interface) ([]testtriggersv1.TestTriggerCondition, error) {
	deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return deployments.MapCRDConditionsToAPI(deployment.Status.Conditions), nil
}

func newDeploymentEvent(eventType testtrigger.EventType, deployment *appsv1.Deployment,
	causes []testtrigger.Cause, clientset kubernetes.Interface) *watcherEvent {
	return &watcherEvent{
		resource:  testtrigger.ResourceDeployment,
		name:      deployment.Name,
		namespace: deployment.Namespace,
		labels:    deployment.Labels,
		object:    deployment,
		eventType: eventType,
		causes:    causes,
		fnGetConditions: func() ([]testtriggersv1.TestTriggerCondition, error) {
			return getDeploymentConditions(deployment.Name, deployment.Name, clientset)
		},
	}
}

func newDaemonSetEvent(eventType testtrigger.EventType, daemonset *appsv1.DaemonSet) *watcherEvent {
	return &watcherEvent{
		resource:  testtrigger.ResourceDaemonSet,
		name:      daemonset.Name,
		namespace: daemonset.Namespace,
		labels:    daemonset.Labels,
		object:    daemonset,
		eventType: eventType,
	}
}

func newStatefulSetEvent(eventType testtrigger.EventType, statefulset *appsv1.StatefulSet) *watcherEvent {
	return &watcherEvent{
		resource:  testtrigger.ResourceStatefulSet,
		name:      statefulset.Name,
		namespace: statefulset.Namespace,
		labels:    statefulset.Labels,
		object:    statefulset,
		eventType: eventType,
	}
}

func newServiceEvent(eventType testtrigger.EventType, service *corev1.Service) *watcherEvent {
	return &watcherEvent{
		resource:  testtrigger.ResourceService,
		name:      service.Name,
		namespace: service.Namespace,
		labels:    service.Labels,
		object:    service,
		eventType: eventType,
	}
}

func newIngressEvent(eventType testtrigger.EventType, ingress *networkingv1.Ingress) *watcherEvent {
	return &watcherEvent{
		resource:  testtrigger.ResourceIngress,
		name:      ingress.Name,
		namespace: ingress.Namespace,
		labels:    ingress.Labels,
		object:    ingress,
		eventType: eventType,
	}
}

func NewClusterEventEvent(eventType testtrigger.EventType, event *corev1.Event) *watcherEvent {
	return &watcherEvent{
		resource:  testtrigger.ResourceEvent,
		name:      event.Name,
		namespace: event.Namespace,
		labels:    event.Labels,
		object:    event,
		eventType: eventType,
	}
}

func newConfigMapEvent(eventType testtrigger.EventType, configMap *corev1.ConfigMap) *watcherEvent {
	return &watcherEvent{
		resource:  testtrigger.ResourceConfigMap,
		name:      configMap.Name,
		namespace: configMap.Namespace,
		labels:    configMap.Labels,
		object:    configMap,
		eventType: eventType,
	}
}
