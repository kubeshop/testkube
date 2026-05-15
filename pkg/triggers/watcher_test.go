package triggers

import (
	"context"
	"fmt"
	"sync/atomic"
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
	"github.com/kubeshop/testkube/pkg/newclients/workflowtriggerclient"
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

func TestService_startCloudTestTriggerWatch_UsesWatcherNamespaces(t *testing.T) {
	ctrl := gomock.NewController(t)
	testTriggersClient := testtriggerclient.NewMockTestTriggerClient(ctrl)

	testTriggersClient.EXPECT().
		List(gomock.Any(), "env-1", testtriggerclient.ListOptions{}, "team-a").
		Return([]testkube.TestTrigger{{
			Name:             "a",
			Namespace:        "team-a",
			Event:            "modified",
			ResourceSelector: &testkube.TestTriggerSelector{},
			TestSelector:     &testkube.TestTriggerSelector{},
		}}, nil).
		Times(1)
	testTriggersClient.EXPECT().
		List(gomock.Any(), "env-1", testtriggerclient.ListOptions{}, "team-b").
		Return([]testkube.TestTrigger{{
			Name:             "b",
			Namespace:        "team-b",
			Event:            "modified",
			ResourceSelector: &testkube.TestTriggerSelector{},
			TestSelector:     &testkube.TestTriggerSelector{},
		}}, nil).
		Times(1)

	service := &Service{
		triggerStatus:      make(map[statusKey]*triggerStatus),
		testTriggersClient: testTriggersClient,
		logger:             log.DefaultLogger,
		eventsBus:          &bus.EventBusMock{},
		metrics:            metrics.NewMetrics(),
		proContext:         &intconfig.ProContext{EnvID: "env-1"},
		scraperInterval:    time.Hour,
		watcherNamespaces:  []string{"team-a", "team-b"},
	}

	stop := make(chan struct{})
	defer close(stop)

	service.startCloudTestTriggerWatch(context.Background(), stop)

	require.Eventually(t, func() bool {
		service.triggerStatusMu.RLock()
		defer service.triggerStatusMu.RUnlock()
		_, a := service.triggerStatus[newStatusKey(triggerSourceV1, "team-a", "a")]
		_, b := service.triggerStatus[newStatusKey(triggerSourceV1, "team-b", "b")]
		return a && b
	}, time.Second, 10*time.Millisecond)
}

func TestService_startCloudTestTriggerWatch_PreservesContentSelector(t *testing.T) {
	ctrl := gomock.NewController(t)
	testTriggersClient := testtriggerclient.NewMockTestTriggerClient(ctrl)

	namespace := "team-a"
	testTriggersClient.EXPECT().
		List(gomock.Any(), "env-1", testtriggerclient.ListOptions{}, namespace).
		Return([]testkube.TestTrigger{{
			Name:      "git-trigger",
			Namespace: namespace,
			Event:     "modified",
			ContentSelector: &testkube.TestTriggerContentSelector{
				Git: &testkube.TestTriggerContentGit{
					Uri:      "https://github.com/kubeshop/testkube.git",
					Revision: "main",
					Paths:    []string{"pkg/triggers"},
				},
			},
		}}, nil).
		AnyTimes()

	service := &Service{
		triggerStatus:      make(map[statusKey]*triggerStatus),
		testTriggersClient: testTriggersClient,
		logger:             log.DefaultLogger,
		eventsBus:          &bus.EventBusMock{},
		metrics:            metrics.NewMetrics(),
		proContext:         &intconfig.ProContext{EnvID: "env-1"},
		scraperInterval:    time.Hour,
		watcherNamespaces:  []string{namespace},
	}

	stop := make(chan struct{})
	defer close(stop)

	service.startCloudTestTriggerWatch(context.Background(), stop)

	key := newStatusKey(triggerSourceV1, namespace, "git-trigger")
	require.Eventually(t, func() bool {
		service.triggerStatusMu.RLock()
		defer service.triggerStatusMu.RUnlock()
		return service.triggerStatus[key] != nil
	}, time.Second, 10*time.Millisecond)

	service.triggerStatusMu.RLock()
	internal := service.triggerStatus[key].trigger
	service.triggerStatusMu.RUnlock()

	require.NotNil(t, internal)
	require.NotNil(t, internal.ContentSelector)
	require.NotNil(t, internal.ContentSelector.Git)
	assert.Equal(t, "https://github.com/kubeshop/testkube.git", internal.ContentSelector.Git.Uri)
	assert.Equal(t, "main", internal.ContentSelector.Git.Revision)
	assert.Equal(t, []string{"pkg/triggers"}, internal.ContentSelector.Git.Paths)
}

func TestService_startCloudWorkflowTriggerWatch_UsesWatcherNamespaces(t *testing.T) {
	ctrl := gomock.NewController(t)
	workflowClient := workflowtriggerclient.NewMockWorkflowTriggerClient(ctrl)

	workflowClient.EXPECT().
		List(gomock.Any(), "env-1", workflowtriggerclient.ListOptions{}, "team-a").
		Return([]testkube.WorkflowTrigger{{Name: "wf-a", Namespace: "team-a"}}, nil).
		Times(1)
	workflowClient.EXPECT().
		List(gomock.Any(), "env-1", workflowtriggerclient.ListOptions{}, "team-b").
		Return([]testkube.WorkflowTrigger{{Name: "wf-b", Namespace: "team-b"}}, nil).
		Times(1)

	service := &Service{
		triggerStatus:          make(map[statusKey]*triggerStatus),
		workflowTriggersClient: workflowClient,
		logger:                 log.DefaultLogger,
		eventsBus:              &bus.EventBusMock{},
		metrics:                metrics.NewMetrics(),
		proContext:             &intconfig.ProContext{EnvID: "env-1"},
		scraperInterval:        time.Hour,
		watcherNamespaces:      []string{"team-a", "team-b"},
	}

	stop := make(chan struct{})
	defer close(stop)

	service.startCloudWorkflowTriggerWatch(context.Background(), stop)

	require.Eventually(t, func() bool {
		service.triggerStatusMu.RLock()
		defer service.triggerStatusMu.RUnlock()
		_, a := service.triggerStatus[newStatusKey(triggerSourceV2, "team-a", "wf-a")]
		_, b := service.triggerStatus[newStatusKey(triggerSourceV2, "team-b", "wf-b")]
		return a && b
	}, time.Second, 10*time.Millisecond)
}

func TestService_startCloudTestTriggerWatch_PreservesFailedNamespaceSnapshot(t *testing.T) {
	ctrl := gomock.NewController(t)
	testTriggersClient := testtriggerclient.NewMockTestTriggerClient(ctrl)

	var teamACalls int32
	testTriggersClient.EXPECT().
		List(gomock.Any(), "env-1", testtriggerclient.ListOptions{}, "team-a").
		DoAndReturn(func(context.Context, string, testtriggerclient.ListOptions, string) ([]testkube.TestTrigger, error) {
			call := atomic.AddInt32(&teamACalls, 1)
			if call == 1 {
				return []testkube.TestTrigger{{
					Name:             "a",
					Namespace:        "team-a",
					Event:            "modified",
					ResourceSelector: &testkube.TestTriggerSelector{},
					TestSelector:     &testkube.TestTriggerSelector{},
				}}, nil
			}
			return nil, assert.AnError
		}).
		AnyTimes()
	testTriggersClient.EXPECT().
		List(gomock.Any(), "env-1", testtriggerclient.ListOptions{}, "team-b").
		Return([]testkube.TestTrigger{{
			Name:             "b",
			Namespace:        "team-b",
			Event:            "modified",
			ResourceSelector: &testkube.TestTriggerSelector{},
			TestSelector:     &testkube.TestTriggerSelector{},
		}}, nil).
		AnyTimes()

	service := &Service{
		triggerStatus:      make(map[statusKey]*triggerStatus),
		testTriggersClient: testTriggersClient,
		logger:             log.DefaultLogger,
		eventsBus:          &bus.EventBusMock{},
		metrics:            metrics.NewMetrics(),
		proContext:         &intconfig.ProContext{EnvID: "env-1"},
		scraperInterval:    20 * time.Millisecond,
		watcherNamespaces:  []string{"team-a", "team-b"},
	}

	stop := make(chan struct{})
	defer close(stop)

	service.startCloudTestTriggerWatch(context.Background(), stop)

	require.Eventually(t, func() bool {
		service.triggerStatusMu.RLock()
		defer service.triggerStatusMu.RUnlock()
		_, a := service.triggerStatus[newStatusKey(triggerSourceV1, "team-a", "a")]
		_, b := service.triggerStatus[newStatusKey(triggerSourceV1, "team-b", "b")]
		return a && b
	}, time.Second, 10*time.Millisecond)

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&teamACalls) >= 2
	}, time.Second, 10*time.Millisecond)

	service.triggerStatusMu.RLock()
	_, a := service.triggerStatus[newStatusKey(triggerSourceV1, "team-a", "a")]
	_, b := service.triggerStatus[newStatusKey(triggerSourceV1, "team-b", "b")]
	service.triggerStatusMu.RUnlock()
	assert.True(t, a)
	assert.True(t, b)
}

func TestService_startCloudWorkflowTriggerWatch_PreservesFailedNamespaceSnapshot(t *testing.T) {
	ctrl := gomock.NewController(t)
	workflowClient := workflowtriggerclient.NewMockWorkflowTriggerClient(ctrl)

	var teamACalls int32
	workflowClient.EXPECT().
		List(gomock.Any(), "env-1", workflowtriggerclient.ListOptions{}, "team-a").
		DoAndReturn(func(context.Context, string, workflowtriggerclient.ListOptions, string) ([]testkube.WorkflowTrigger, error) {
			call := atomic.AddInt32(&teamACalls, 1)
			if call == 1 {
				return []testkube.WorkflowTrigger{{Name: "wf-a", Namespace: "team-a"}}, nil
			}
			return nil, assert.AnError
		}).
		AnyTimes()
	workflowClient.EXPECT().
		List(gomock.Any(), "env-1", workflowtriggerclient.ListOptions{}, "team-b").
		Return([]testkube.WorkflowTrigger{{Name: "wf-b", Namespace: "team-b"}}, nil).
		AnyTimes()

	service := &Service{
		triggerStatus:          make(map[statusKey]*triggerStatus),
		workflowTriggersClient: workflowClient,
		logger:                 log.DefaultLogger,
		eventsBus:              &bus.EventBusMock{},
		metrics:                metrics.NewMetrics(),
		proContext:             &intconfig.ProContext{EnvID: "env-1"},
		scraperInterval:        20 * time.Millisecond,
		watcherNamespaces:      []string{"team-a", "team-b"},
	}

	stop := make(chan struct{})
	defer close(stop)

	service.startCloudWorkflowTriggerWatch(context.Background(), stop)

	require.Eventually(t, func() bool {
		service.triggerStatusMu.RLock()
		defer service.triggerStatusMu.RUnlock()
		_, a := service.triggerStatus[newStatusKey(triggerSourceV2, "team-a", "wf-a")]
		_, b := service.triggerStatus[newStatusKey(triggerSourceV2, "team-b", "wf-b")]
		return a && b
	}, time.Second, 10*time.Millisecond)

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&teamACalls) >= 2
	}, time.Second, 10*time.Millisecond)

	service.triggerStatusMu.RLock()
	_, a := service.triggerStatus[newStatusKey(triggerSourceV2, "team-a", "wf-a")]
	_, b := service.triggerStatus[newStatusKey(triggerSourceV2, "team-b", "wf-b")]
	service.triggerStatusMu.RUnlock()
	assert.True(t, a)
	assert.True(t, b)
}

func TestService_startCloudTestTriggerWatch_DoesNotOverwriteWildcardDataOnNamespaceFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	testTriggersClient := testtriggerclient.NewMockTestTriggerClient(ctrl)

	var wildcardCalls int32
	testTriggersClient.EXPECT().
		List(gomock.Any(), "env-1", testtriggerclient.ListOptions{}, "*").
		DoAndReturn(func(context.Context, string, testtriggerclient.ListOptions, string) ([]testkube.TestTrigger, error) {
			call := atomic.AddInt32(&wildcardCalls, 1)
			disabled := call >= 2
			return []testkube.TestTrigger{{
				Name:             "a",
				Namespace:        "team-a",
				Event:            "modified",
				ResourceSelector: &testkube.TestTriggerSelector{},
				TestSelector:     &testkube.TestTriggerSelector{},
				Disabled:         disabled,
			}}, nil
		}).
		AnyTimes()
	testTriggersClient.EXPECT().
		List(gomock.Any(), "env-1", testtriggerclient.ListOptions{}, "team-a").
		DoAndReturn(func(context.Context, string, testtriggerclient.ListOptions, string) ([]testkube.TestTrigger, error) {
			if atomic.LoadInt32(&wildcardCalls) < 2 {
				return []testkube.TestTrigger{{
					Name:             "a",
					Namespace:        "team-a",
					Event:            "modified",
					ResourceSelector: &testkube.TestTriggerSelector{},
					TestSelector:     &testkube.TestTriggerSelector{},
				}}, nil
			}
			return nil, assert.AnError
		}).
		AnyTimes()

	service := &Service{
		triggerStatus:      make(map[statusKey]*triggerStatus),
		testTriggersClient: testTriggersClient,
		logger:             log.DefaultLogger,
		eventsBus:          &bus.EventBusMock{},
		metrics:            metrics.NewMetrics(),
		proContext:         &intconfig.ProContext{EnvID: "env-1"},
		scraperInterval:    20 * time.Millisecond,
		watcherNamespaces:  []string{"*", "team-a"},
	}

	stop := make(chan struct{})
	defer close(stop)

	service.startCloudTestTriggerWatch(context.Background(), stop)

	require.Eventually(t, func() bool {
		service.triggerStatusMu.RLock()
		defer service.triggerStatusMu.RUnlock()
		st := service.triggerStatus[newStatusKey(triggerSourceV1, "team-a", "a")]
		return st != nil && st.trigger != nil && st.trigger.Disabled
	}, time.Second, 10*time.Millisecond)
}

func TestService_startCloudWorkflowTriggerWatch_DoesNotOverwriteWildcardDataOnNamespaceFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	workflowClient := workflowtriggerclient.NewMockWorkflowTriggerClient(ctrl)

	var wildcardCalls int32
	workflowClient.EXPECT().
		List(gomock.Any(), "env-1", workflowtriggerclient.ListOptions{}, "*").
		DoAndReturn(func(context.Context, string, workflowtriggerclient.ListOptions, string) ([]testkube.WorkflowTrigger, error) {
			call := atomic.AddInt32(&wildcardCalls, 1)
			return []testkube.WorkflowTrigger{{
				Name:      "wf-a",
				Namespace: "team-a",
				Labels: map[string]string{
					"version": fmt.Sprintf("%d", call),
				},
			}}, nil
		}).
		AnyTimes()
	workflowClient.EXPECT().
		List(gomock.Any(), "env-1", workflowtriggerclient.ListOptions{}, "team-a").
		DoAndReturn(func(context.Context, string, workflowtriggerclient.ListOptions, string) ([]testkube.WorkflowTrigger, error) {
			if atomic.LoadInt32(&wildcardCalls) < 2 {
				return []testkube.WorkflowTrigger{{Name: "wf-a", Namespace: "team-a"}}, nil
			}
			return nil, assert.AnError
		}).
		AnyTimes()

	service := &Service{
		triggerStatus:          make(map[statusKey]*triggerStatus),
		workflowTriggersClient: workflowClient,
		logger:                 log.DefaultLogger,
		eventsBus:              &bus.EventBusMock{},
		metrics:                metrics.NewMetrics(),
		proContext:             &intconfig.ProContext{EnvID: "env-1"},
		scraperInterval:        20 * time.Millisecond,
		watcherNamespaces:      []string{"*", "team-a"},
	}

	stop := make(chan struct{})
	defer close(stop)

	service.startCloudWorkflowTriggerWatch(context.Background(), stop)

	require.Eventually(t, func() bool {
		service.triggerStatusMu.RLock()
		defer service.triggerStatusMu.RUnlock()
		st := service.triggerStatus[newStatusKey(triggerSourceV2, "team-a", "wf-a")]
		return st != nil && st.trigger != nil && st.trigger.Labels["version"] == "2"
	}, time.Second, 10*time.Millisecond)
}

func TestService_getCloudWatchNamespaces(t *testing.T) {
	t.Run("uses watcher namespaces when configured", func(t *testing.T) {
		s := &Service{watcherNamespaces: []string{"a", "b"}}
		assert.Equal(t, []string{"a", "b"}, s.getCloudWatchNamespaces())
	})

	t.Run("uses wildcard when watcher namespaces are empty", func(t *testing.T) {
		s := &Service{}
		assert.Equal(t, []string{"*"}, s.getCloudWatchNamespaces())
	})

	t.Run("normalizes wildcard and de-duplicates", func(t *testing.T) {
		s := &Service{watcherNamespaces: []string{"*", "team-a", "*", "team-a"}}
		assert.Equal(t, []string{"*"}, s.getCloudWatchNamespaces())
	})
}

func TestService_getWorkflowTriggerWatchNamespaces(t *testing.T) {
	t.Run("uses watcher namespaces when configured", func(t *testing.T) {
		s := &Service{watcherNamespaces: []string{"team-a", "team-b"}}
		assert.Equal(t, []string{"team-a", "team-b"}, s.getWorkflowTriggerWatchNamespaces())
	})

	t.Run("uses all namespaces when watcher namespaces are empty", func(t *testing.T) {
		s := &Service{}
		assert.Equal(t, []string{metav1.NamespaceAll}, s.getWorkflowTriggerWatchNamespaces())
	})

	t.Run("normalizes wildcard and de-duplicates", func(t *testing.T) {
		s := &Service{watcherNamespaces: []string{"*", "team-a", "*", "team-a"}}
		assert.Equal(t, []string{metav1.NamespaceAll}, s.getWorkflowTriggerWatchNamespaces())
	})
}
