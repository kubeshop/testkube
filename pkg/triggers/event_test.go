package triggers

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation"
	k8sscheme "k8s.io/client-go/kubernetes/scheme"

	"github.com/kubeshop/testkube/pkg/log"
)

func TestNewWatcherEvent(t *testing.T) {
	scheme := runtime.NewScheme()
	metav1.AddToGroupVersion(scheme, schema.GroupVersion{Version: "v1"})
	k8sscheme.AddToScheme(scheme)

	service := &Service{
		agentName:         "testkube-agent",
		testkubeNamespace: "testkube-ns",
		informers:         &k8sInformers{scheme: scheme},
		logger:            log.DefaultLogger,
	}

	deploymentLabels := map[string]string{"app": "nginx", "env": "prod"}
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-deployment",
			Namespace: "default",
			Labels:    deploymentLabels,
		},
	}
	event := service.newWatcherEvent(
		"created",
		&deployment.ObjectMeta,
		deployment,
		"deployment",
	)

	expectedEventLabels := map[string]string{
		"testkube.io/agent-name":         "testkube-agent",
		"testkube.io/agent-namespace":    "testkube-ns",
		"testkube.io/resource-name":      "nginx-deployment",
		"testkube.io/resource-namespace": "default",
		"testkube.io/resource-kind":      "Deployment",
		"testkube.io/resource-group":     "apps",
		"testkube.io/resource-version":   "v1",
	}

	assert.EqualValues(t, "deployment", event.resource, "resource should be correct")
	assert.Equal(t, "nginx-deployment", event.name, "name should be correct")
	assert.Equal(t, "default", event.Namespace, "namespace should be correct")
	assert.Equal(t, deploymentLabels, event.resourceLabels, "resourceLabels should be correct")
	assert.EqualValues(t, "created", event.eventType, "eventType should be correct")
	assert.Equal(t, expectedEventLabels, event.EventLabels, "EventLabels should be correct")
}

func TestNewWatcherEventSanitizesEventLabels(t *testing.T) {
	scheme := runtime.NewScheme()
	metav1.AddToGroupVersion(scheme, schema.GroupVersion{Version: "v1"})
	k8sscheme.AddToScheme(scheme)

	longName := "external-booking-retry-cancellation-failures-cronjob.187e03d261b2b5c0"
	longLabel := strings.Repeat("x", validation.LabelValueMaxLength+10)
	service := &Service{
		agentName:         "testkube-agent",
		testkubeNamespace: "testkube-ns",
		informers:         &k8sInformers{scheme: scheme},
		logger:            log.DefaultLogger,
		eventLabels: map[string]string{
			"custom": longLabel,
		},
	}

	kubeEvent := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      longName,
			Namespace: "default",
		},
	}
	event := service.newWatcherEvent(
		"created",
		&kubeEvent.ObjectMeta,
		kubeEvent,
		"event",
	)

	resourceNameLabel := event.EventLabels[eventLabelKeyResourceName]
	assert.Equal(t, longName[:validation.LabelValueMaxLength], resourceNameLabel)
	assert.Empty(t, validation.IsValidLabelValue(resourceNameLabel))
	assert.LessOrEqual(t, len(resourceNameLabel), validation.LabelValueMaxLength)

	customLabel := event.EventLabels["custom"]
	assert.Equal(t, longLabel[:validation.LabelValueMaxLength], customLabel)
	assert.Empty(t, validation.IsValidLabelValue(customLabel))
	assert.LessOrEqual(t, len(customLabel), validation.LabelValueMaxLength)
}
