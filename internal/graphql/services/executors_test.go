package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	executorv1 "github.com/kubeshop/testkube-operator/api/executor/v1"
	executorsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/executors/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

var (
	srvMock    = NewMockService()
	k8sObjects = []k8sclient.Object{
		&executorv1.Executor{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Executor",
				APIVersion: "executor.testkube.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "default",
				Labels: map[string]string{
					"label-name": "label-value",
				},
			},
			Spec: executorv1.ExecutorSpec{
				Types:                []string{"curl/test"},
				ExecutorType:         "job",
				JobTemplate:          "",
				JobTemplateReference: "",
			},
			Status: executorv1.ExecutorStatus{},
		},
	}
	sample = testkube.ExecutorDetails{
		Name: "sample",
		Executor: &testkube.Executor{
			ExecutorType:         "job",
			Image:                "",
			ImagePullSecrets:     nil,
			Command:              nil,
			Args:                 nil,
			Types:                []string{"curl/test"},
			Uri:                  "",
			ContentTypes:         nil,
			JobTemplate:          "",
			JobTemplateReference: "",
			Labels:               map[string]string{"label-name": "label-value"},
			Features:             nil,
			Meta:                 nil,
		},
	}
	k8sObjects2 = []k8sclient.Object{
		&executorv1.Executor{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Executor",
				APIVersion: "executor.testkube.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "default",
				Labels: map[string]string{
					"label-name": "label-value",
				},
			},
			Spec: executorv1.ExecutorSpec{
				Types:                []string{"other/test"},
				ExecutorType:         "job",
				JobTemplate:          "",
				JobTemplateReference: "",
			},
			Status: executorv1.ExecutorStatus{},
		},
	}
	sample2 = testkube.ExecutorDetails{
		Name: "sample",
		Executor: &testkube.Executor{
			ExecutorType:         "job",
			Image:                "",
			ImagePullSecrets:     nil,
			Command:              nil,
			Args:                 nil,
			Types:                []string{"other/test"},
			Uri:                  "",
			ContentTypes:         nil,
			JobTemplate:          "",
			JobTemplateReference: "",
			Labels:               map[string]string{"label-name": "label-value"},
			Features:             nil,
			Meta:                 nil,
		},
	}
)

var (
	client *executorsclientv1.ExecutorsClient
	srv    ExecutorsService
)

func ResetMocks() {
	srvMock.Reset()
	client = getMockExecutorClient(k8sObjects)
	srv = NewExecutorsService(srvMock, client)
}

func TestExecutorsService_List(t *testing.T) {
	t.Run("should list all available executors when no selector passed", func(t *testing.T) {
		ResetMocks()
		result, err := srv.List("")
		assert.NoError(t, err)
		assert.Equal(t, result, []testkube.ExecutorDetails{sample})
	})

	t.Run("should list none executors when none matches selector", func(t *testing.T) {
		ResetMocks()
		result, err := srv.List("xyz=def")
		assert.NoError(t, err)
		assert.Equal(t, result, []testkube.ExecutorDetails{})
	})

	t.Run("should list executors matching the selector", func(t *testing.T) {
		ResetMocks()
		result, err := srv.List("label-name=label-value")
		assert.NoError(t, err)
		assert.Equal(t, result, []testkube.ExecutorDetails{sample})
	})
}

func TestExecutorsService_SubscribeList(t *testing.T) {
	t.Run("should cancel subscription when the context is canceled", func(t *testing.T) {
		ResetMocks()
		ctx, cancel := context.WithCancel(context.Background())
		ch, err := srv.SubscribeList(ctx, "")
		assert.NoError(t, err)
		<-ch
		cancel()
		_, opened := <-ch
		assert.False(t, opened)
		assert.Len(t, srvMock.BusMock().ListQueues(), 0)
	})

	t.Run("should return initial list of executors", func(t *testing.T) {
		ResetMocks()
		ch, err := srv.SubscribeList(context.Background(), "")
		assert.NoError(t, err)
		result := <-ch
		assert.Equal(t, result, []testkube.ExecutorDetails{sample})
	})

	t.Run("should subscribe for new entries", func(t *testing.T) {
		ResetMocks()
		ch, err := srv.SubscribeList(context.Background(), "")
		assert.NoError(t, err)
		result := <-ch
		assert.Equal(t, result, []testkube.ExecutorDetails{sample})
		assert.Len(t, srvMock.BusMock().ListQueues(), 1)
	})

	t.Run("should return new list of executors after events are triggered", func(t *testing.T) {
		ResetMocks()
		ch, err := srv.SubscribeList(context.Background(), "")
		assert.NoError(t, err)
		<-ch
		client.Client = getMockExecutorClient(k8sObjects2).Client
		assert.NoError(t, srvMock.BusMock().PublishTopic("events.executor.create", testkube.Event{
			Type_:    testkube.EventCreated,
			Resource: testkube.EventResourceExecutor,
		}))
		assert.Equal(t, <-ch, []testkube.ExecutorDetails{sample2})
	})
}

func getMockExecutorClient(initObjects []k8sclient.Object) *executorsclientv1.ExecutorsClient {
	scheme := runtime.NewScheme()
	executorv1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(initObjects...).
		Build()
	return executorsclientv1.NewClient(fakeClient, "")
}
