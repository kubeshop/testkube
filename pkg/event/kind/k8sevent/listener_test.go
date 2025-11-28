package k8sevent

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestK8sEventListenerCreatesEvent(t *testing.T) {
	t.Parallel()

	clientset := fake.NewSimpleClientset()
	listener := NewK8sEventListener("k8s", "", "tk-dev",
		[]testkube.EventType{*testkube.EventEndTestWorkflowSuccess}, clientset)

	event := testkube.Event{
		Id:    "event-123",
		Type_: testkube.EventEndTestWorkflowSuccess,
		TestWorkflowExecution: &testkube.TestWorkflowExecution{
			Id: "exec-1",
			Workflow: &testkube.TestWorkflow{
				Name:   "sample-workflow",
				Labels: map[string]string{"docs": "example"},
			},
		},
	}

	result := listener.Notify(event)
	assert.Empty(t, result.Error())

	created, err := clientset.CoreV1().Events("tk-dev").Get(context.Background(),
		"testkube-event-event-123", metav1.GetOptions{})
	require.NoError(t, err)

	assert.Equal(t, "event-123", result.Id)
	assert.Equal(t, "end-testworkflow-success", created.Reason)
	assert.Equal(t, "succeed", created.Action)
	assert.Equal(t, "sample-workflow", created.InvolvedObject.Name)
	assert.Equal(t, "testworkflows.testkube.io/v1", created.InvolvedObject.APIVersion)
	assert.Equal(t, "TestWorkflow", created.InvolvedObject.Kind)
	assert.Equal(t, "example", created.Labels["docs"])
}

func TestK8sEventListenerCreateError(t *testing.T) {
	t.Parallel()

	clientset := fake.NewSimpleClientset()
	// Force an error on event creation.
	clientset.PrependReactor("create", "events", func(action clienttesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, errors.New("create failed")
	})

	listener := NewK8sEventListener("k8s", "", "tk-dev",
		[]testkube.EventType{*testkube.EventEndTestWorkflowSuccess}, clientset)

	event := testkube.Event{
		Id:    "event-err",
		Type_: testkube.EventEndTestWorkflowSuccess,
		TestWorkflowExecution: &testkube.TestWorkflowExecution{
			Id:       "exec-err",
			Workflow: &testkube.TestWorkflow{Name: "sample-workflow"},
		},
	}

	result := listener.Notify(event)
	assert.Equal(t, "create failed", result.Error())

	events, err := clientset.CoreV1().Events("tk-dev").List(context.Background(), metav1.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, events.Items, 0)
}
