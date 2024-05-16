package k8sevent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

var testEventTypes = []testkube.EventType{*testkube.EventStartTest}

func TestK8sEventListener_Notify(t *testing.T) {
	t.Parallel()

	t.Run("send event success response", func(t *testing.T) {
		t.Parallel()

		// given
		clientset := fake.NewSimpleClientset()

		l := NewK8sEventListener("k8seli", "", "", testEventTypes, clientset)

		// when
		r := l.Notify(testkube.Event{
			Type_:         testkube.EventStartTest,
			TestExecution: exampleExecution(),
		})

		assert.Equal(t, "", r.Error())
	})

}

func exampleExecution() *testkube.Execution {
	execution := testkube.NewQueuedExecution()
	execution.Id = "1"
	execution.Name = "test-1"
	execution.TestName = "test"
	execution.TestNamespace = "testkube"
	return execution
}
