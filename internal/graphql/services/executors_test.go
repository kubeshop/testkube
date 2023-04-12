package services

import (
	"context"
	"testing"

	executorv1 "github.com/kubeshop/testkube-operator/apis/executor/v1"
	executorsclientv1 "github.com/kubeshop/testkube-operator/client/executors/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/log"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
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
				Types:        []string{"curl/test"},
				ExecutorType: "job",
				JobTemplate:  "",
			},
			Status: executorv1.ExecutorStatus{},
		},
	}
	sample = testkube.ExecutorDetails{
		Name: "sample",
		Executor: &testkube.Executor{
			ExecutorType:     "job",
			Image:            "",
			ImagePullSecrets: nil,
			Command:          nil,
			Args:             nil,
			Types:            []string{"curl/test"},
			Uri:              "",
			ContentTypes:     nil,
			JobTemplate:      "",
			Labels:           map[string]string{"label-name": "label-value"},
			Features:         nil,
			Meta:             nil,
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
				Types:        []string{"other/test"},
				ExecutorType: "job",
				JobTemplate:  "",
			},
			Status: executorv1.ExecutorStatus{},
		},
	}
	sample2 = testkube.ExecutorDetails{
		Name: "sample",
		Executor: &testkube.Executor{
			ExecutorType:     "job",
			Image:            "",
			ImagePullSecrets: nil,
			Command:          nil,
			Args:             nil,
			Types:            []string{"other/test"},
			Uri:              "",
			ContentTypes:     nil,
			JobTemplate:      "",
			Labels:           map[string]string{"label-name": "label-value"},
			Features:         nil,
			Meta:             nil,
		},
	}
)

var (
	busMock *bus.EventBusMock
	client  *executorsclientv1.ExecutorsClient
	service *ExecutorsService
)

func ResetMocks() {
	busMock = bus.NewEventBusMock()
	client = getMockExecutorClient(k8sObjects)
	service = &ExecutorsService{
		Service: &Service{
			Bus:    busMock,
			Logger: log.DefaultLogger,
		},
		Client: client,
	}
}

func TestExecutorsService_List(t *testing.T) {
	t.Run("should list all available executors when no selector passed", func(t *testing.T) {
		ResetMocks()
		result, err := service.List("")
		assert.NoError(t, err)
		assert.Equal(t, result, []testkube.ExecutorDetails{sample})
	})

	t.Run("should list none executors when none matches selector", func(t *testing.T) {
		ResetMocks()
		result, err := service.List("xyz=def")
		assert.NoError(t, err)
		assert.Equal(t, result, []testkube.ExecutorDetails{})
	})

	t.Run("should list executors matching the selector", func(t *testing.T) {
		ResetMocks()
		result, err := service.List("label-name=label-value")
		assert.NoError(t, err)
		assert.Equal(t, result, []testkube.ExecutorDetails{sample})
	})
}

func TestExecutorsService_SubscribeList(t *testing.T) {
	t.Run("should cancel subscription when the context is canceled", func(t *testing.T) {
		ResetMocks()
		ctx, cancel := context.WithCancel(context.Background())
		ch, err := service.SubscribeList(ctx, "")
		assert.NoError(t, err)
		<-ch
		cancel()
		_, opened := <-ch
		assert.False(t, opened)
		assert.Len(t, busMock.ListQueues(), 0)
	})

	t.Run("should return initial list of executors", func(t *testing.T) {
		ResetMocks()
		ch, err := service.SubscribeList(context.Background(), "")
		assert.NoError(t, err)
		result := <-ch
		assert.Equal(t, result, []testkube.ExecutorDetails{sample})
	})

	t.Run("should subscribe for new entries", func(t *testing.T) {
		ResetMocks()
		ch, err := service.SubscribeList(context.Background(), "")
		assert.NoError(t, err)
		result := <-ch
		assert.Equal(t, result, []testkube.ExecutorDetails{sample})
		assert.Len(t, busMock.ListQueues(), 1)
	})

	t.Run("should return new list of executors after events are triggered", func(t *testing.T) {
		ResetMocks()
		ch, err := service.SubscribeList(context.Background(), "")
		assert.NoError(t, err)
		<-ch
		client.Client = getMockExecutorClient(k8sObjects2).Client
		assert.NoError(t, busMock.PublishTopic("events.executor.create", testkube.Event{}))
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
