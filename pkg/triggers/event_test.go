package triggers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
