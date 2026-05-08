package triggers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	"github.com/kubeshop/testkube/internal/app/api/metrics"
	intconfig "github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/newclients/testtriggerclient"
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

	clientset := fake.NewClientset()
	testKubeClientset := faketestkube.NewSimpleClientset()
	namespace := "testkube"

	service := newWatcherTestService(clientset, testKubeClientset, namespace)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go service.runWatcher(ctx)

	require.Eventually(t, func() bool {
		service.informersMu.RLock()
		defer service.informersMu.RUnlock()
		return service.informers != nil
	}, time.Second, 10*time.Millisecond)

	trigger := &testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: "test-trigger"},
		Spec:       testtriggersv1.TestTriggerSpec{Event: "created"},
	}
	_, err := testKubeClientset.TestsV1().TestTriggers(namespace).Create(ctx, trigger, metav1.CreateOptions{})
	require.NoError(t, err)

	assert.Eventually(t, func() bool {
		service.triggerStatusMu.RLock()
		defer service.triggerStatusMu.RUnlock()
		_, ok := service.triggerStatus[newStatusKey(triggerSourceV1, namespace, "test-trigger")]
		return ok
	}, time.Second, 10*time.Millisecond)

	err = testKubeClientset.TestsV1().TestTriggers(namespace).Delete(ctx, "test-trigger", metav1.DeleteOptions{})
	require.NoError(t, err)

	assert.Eventually(t, func() bool {
		service.triggerStatusMu.RLock()
		defer service.triggerStatusMu.RUnlock()
		return len(service.triggerStatus) == 0
	}, time.Second, 10*time.Millisecond)
}

func TestService_runWatcher_stopsOnContextCancellation(t *testing.T) {

	clientset := fake.NewClientset()
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
		service.informersMu.RLock()
		defer service.informersMu.RUnlock()
		return service.informers != nil
	}, time.Second, 10*time.Millisecond)

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("expected watcher to stop after context cancellation")
	}

	require.Eventually(t, func() bool {
		service.informersMu.RLock()
		defer service.informersMu.RUnlock()
		return service.informers == nil
	}, time.Second, 10*time.Millisecond)
}

func TestService_startCloudTestTriggerWatchPreservesResourceRef(t *testing.T) {
	ctrl := gomock.NewController(t)
	testTriggersClient := testtriggerclient.NewMockTestTriggerClient(ctrl)

	namespace := "testkube"
	trigger := testkube.TestTrigger{
		Name:      "custom-resource-trigger",
		Namespace: namespace,
		ResourceRef: &testkube.TestTriggerResourceRef{
			Group:   "argoproj.io",
			Version: "v1alpha1",
			Kind:    "Rollout",
		},
		ResourceSelector: &testkube.TestTriggerSelector{},
		Event:            "modified",
		TestSelector:     &testkube.TestTriggerSelector{},
	}

	testTriggersClient.EXPECT().
		List(gomock.Any(), "env-1", testtriggerclient.ListOptions{}, namespace).
		Return([]testkube.TestTrigger{trigger}, nil).
		AnyTimes()

	service := &Service{
		triggerStatus:      make(map[statusKey]*triggerStatus),
		testTriggersClient: testTriggersClient,
		logger:             log.DefaultLogger,
		eventsBus:          &bus.EventBusMock{},
		metrics:            metrics.NewMetrics(),
		proContext:         &intconfig.ProContext{EnvID: "env-1"},
		testkubeNamespace:  namespace,
		scraperInterval:    time.Hour,
		watcherNamespaces:  []string{namespace},
	}

	stop := make(chan struct{})
	defer close(stop)

	service.startCloudTestTriggerWatch(context.Background(), stop)

	key := newStatusKey(triggerSourceV1, namespace, "custom-resource-trigger")
	require.Eventually(t, func() bool {
		service.triggerStatusMu.RLock()
		defer service.triggerStatusMu.RUnlock()
		return service.triggerStatus[key] != nil
	}, time.Second, 10*time.Millisecond)

	service.triggerStatusMu.RLock()
	internal := service.triggerStatus[key].trigger
	service.triggerStatusMu.RUnlock()

	require.NotNil(t, internal)
	assert.Equal(t, "argoproj.io", internal.ResourceGroup)
	assert.Equal(t, "v1alpha1", internal.ResourceVersion)
	assert.Equal(t, "Rollout", internal.ResourceKind)
}
