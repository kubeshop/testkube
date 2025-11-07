package triggers

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	"github.com/kubeshop/testkube/internal/app/api/metrics"
	intconfig "github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/log"
	faketestkube "github.com/kubeshop/testkube/pkg/operator/clientset/versioned/fake"
)

func newWatcherTestService(clientset *fake.Clientset, testKubeClientset *faketestkube.Clientset, namespace string) *Service {
	return &Service{
		triggerStatus:     make(map[statusKey]*triggerStatus),
		clientset:         clientset,
		testKubeClientset: testKubeClientset,
		logger:            log.DefaultLogger,
		eventsBus:         &bus.EventBusMock{},
		metrics:           metrics.NewMetrics(),
		proContext:        &intconfig.ProContext{},
		testkubeNamespace: namespace,
		watcherNamespaces: []string{namespace},
	}
}

func TestService_runWatcher_createsAndDeletesTrigger(t *testing.T) {
	t.Parallel()

	clientset := fake.NewSimpleClientset()
	testKubeClientset := faketestkube.NewSimpleClientset()
	namespace := "testkube"

	service := newWatcherTestService(clientset, testKubeClientset, namespace)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go service.runWatcher(ctx)

	require.Eventually(t, func() bool {
		return service.informers != nil
	}, time.Second, 10*time.Millisecond)

	trigger := &testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: "test-trigger"},
		Spec:       testtriggersv1.TestTriggerSpec{Event: "created"},
	}
	_, err := testKubeClientset.TestsV1().TestTriggers(namespace).Create(ctx, trigger, metav1.CreateOptions{})
	require.NoError(t, err)

	assert.Eventually(t, func() bool {
		_, ok := service.triggerStatus[newStatusKey(namespace, "test-trigger")]
		return ok
	}, time.Second, 10*time.Millisecond)

	err = testKubeClientset.TestsV1().TestTriggers(namespace).Delete(ctx, "test-trigger", metav1.DeleteOptions{})
	require.NoError(t, err)

	assert.Eventually(t, func() bool {
		return len(service.triggerStatus) == 0
	}, time.Second, 10*time.Millisecond)
}

func TestService_runWatcher_handlesPodEvents(t *testing.T) {
	t.Parallel()

	clientset := fake.NewSimpleClientset()
	testKubeClientset := faketestkube.NewSimpleClientset()
	namespace := "testkube"

	var triggered atomic.Bool
	service := newWatcherTestService(clientset, testKubeClientset, namespace)
	service.triggerExecutor = func(ctx context.Context, e *watcherEvent, trigger *testtriggersv1.TestTrigger) error {
		triggered.Store(true)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go service.runWatcher(ctx)

	require.Eventually(t, func() bool {
		return service.informers != nil
	}, time.Second, 10*time.Millisecond)

	trigger := &testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: "pod-trigger"},
		Spec: testtriggersv1.TestTriggerSpec{
			Resource:         "pod",
			ResourceSelector: testtriggersv1.TestTriggerSelector{Name: "watched-pod"},
			Event:            "created",
			Action:           "run",
			Execution:        "test",
		},
	}
	_, err := testKubeClientset.TestsV1().TestTriggers(namespace).Create(ctx, trigger, metav1.CreateOptions{})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		_, ok := service.triggerStatus[newStatusKey(namespace, "pod-trigger")]
		return ok
	}, 3*time.Second, 20*time.Millisecond)

	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: "watched-pod"}}
	_, err = clientset.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return triggered.Load()
	}, 3*time.Second, 20*time.Millisecond, "expected trigger executor to run")
}

func TestService_runWatcher_stopsOnContextCancellation(t *testing.T) {
	t.Parallel()

	clientset := fake.NewSimpleClientset()
	testKubeClientset := faketestkube.NewSimpleClientset()
	namespace := "testkube"

	service := newWatcherTestService(clientset, testKubeClientset, namespace)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		service.runWatcher(ctx)
		close(done)
	}()

	require.Eventually(t, func() bool {
		return service.informers != nil
	}, time.Second, 10*time.Millisecond)

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("expected watcher to stop after context cancellation")
	}

	require.Eventually(t, func() bool {
		return service.informers == nil
	}, time.Second, 10*time.Millisecond)
}
