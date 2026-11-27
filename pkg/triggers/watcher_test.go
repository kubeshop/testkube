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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	executorv1 "github.com/kubeshop/testkube/api/executor/v1"
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
			Event:     string(testtriggersv1.TestTriggerEventGitPush),
			ContentSelector: &testkube.TestTriggerContentSelector{
				Git: &testkube.TestTriggerContentGit{
					Uri:      "https://github.com/kubeshop/testkube.git",
					Branches: []string{"main"},
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
	assert.Equal(t, []string{"main"}, internal.ContentSelector.Git.Branches)
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

	t.Run("uses testkube namespace when watcher namespaces are empty", func(t *testing.T) {
		s := &Service{testkubeNamespace: "testkube"}
		assert.Equal(t, []string{"testkube"}, s.getWorkflowTriggerWatchNamespaces())
	})

	t.Run("normalizes wildcard and de-duplicates", func(t *testing.T) {
		s := &Service{watcherNamespaces: []string{"*", "team-a", "*", "team-a"}}
		assert.Equal(t, []string{metav1.NamespaceAll}, s.getWorkflowTriggerWatchNamespaces())
	})
}

func TestUnwrapDeletedObject(t *testing.T) {
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "pod-1"}}

	t.Run("returns direct object", func(t *testing.T) {
		got, ok := unwrapDeletedObject[*corev1.Pod](pod)
		require.True(t, ok)
		assert.Same(t, pod, got)
	})

	t.Run("unwraps DeletedFinalStateUnknown tombstone", func(t *testing.T) {
		got, ok := unwrapDeletedObject[*corev1.Pod](cache.DeletedFinalStateUnknown{
			Key: "ns/pod-1",
			Obj: pod,
		})
		require.True(t, ok)
		assert.Same(t, pod, got)
	})

	t.Run("rejects tombstone with unexpected object type", func(t *testing.T) {
		got, ok := unwrapDeletedObject[*corev1.Pod](cache.DeletedFinalStateUnknown{
			Key: "ns/pod-1",
			Obj: &corev1.Service{},
		})
		assert.False(t, ok)
		assert.Nil(t, got)
	})

	t.Run("rejects unexpected object type", func(t *testing.T) {
		got, ok := unwrapDeletedObject[*corev1.Pod]("not-a-pod")
		assert.False(t, ok)
		assert.Nil(t, got)
	})
}

func TestService_eventHandlers_deleteTombstone(t *testing.T) {
	namespace := "testkube"
	clientset := fake.NewClientset()
	testKubeClientset := faketestkube.NewSimpleClientset()
	eventsBus := bus.NewEventBusMock()
	published := make(chan testkube.Event, 10)
	require.NoError(t, eventsBus.Subscribe("test-queue", func(e testkube.Event) error {
		published <- e
		return nil
	}))

	service := newWatcherTestService(clientset, testKubeClientset, namespace)
	service.eventsBus = eventsBus
	ctx := context.Background()

	trigger := &testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: "trigger-1"},
		Spec:       testtriggersv1.TestTriggerSpec{Resource: "pod", Event: "created"},
	}
	service.addTrigger(ctx, trigger)
	require.Len(t, service.triggerStatus, 1)

	service.testTriggerEventHandler(ctx).DeleteFunc(cache.DeletedFinalStateUnknown{
		Key: namespace + "/trigger-1",
		Obj: trigger,
	})
	assert.Empty(t, service.triggerStatus)

	select {
	case ev := <-published:
		assert.Equal(t, testkube.EventDeleted, ev.Type_)
		assert.Equal(t, testkube.EventResourceTrigger, ev.Resource)
		assert.Equal(t, "trigger-1", ev.ResourceId)
	case <-time.After(time.Second):
		t.Fatal("expected deleted trigger event to be published")
	}

	webhook := &executorv1.Webhook{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: "webhook-1"}}
	service.webhookEventHandler().DeleteFunc(cache.DeletedFinalStateUnknown{
		Key: namespace + "/webhook-1",
		Obj: webhook,
	})
	select {
	case ev := <-published:
		assert.Equal(t, testkube.EventDeleted, ev.Type_)
		assert.Equal(t, testkube.EventResourceWebhook, ev.Resource)
		assert.Equal(t, "webhook-1", ev.ResourceId)
	case <-time.After(time.Second):
		t.Fatal("expected deleted webhook event to be published")
	}

	webhookTemplate := &executorv1.WebhookTemplate{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: "webhook-tpl-1"}}
	service.webhookTemplateEventHandler().DeleteFunc(cache.DeletedFinalStateUnknown{
		Key: namespace + "/webhook-tpl-1",
		Obj: webhookTemplate,
	})
	select {
	case ev := <-published:
		assert.Equal(t, testkube.EventDeleted, ev.Type_)
		assert.Equal(t, testkube.EventResourceWebhookTemplate, ev.Resource)
		assert.Equal(t, "webhook-tpl-1", ev.ResourceId)
	case <-time.After(time.Second):
		t.Fatal("expected deleted webhook template event to be published")
	}

	// Verify resource handlers unwrap tombstones without panicking or dropping events.
	metaObj := metav1.ObjectMeta{Namespace: namespace, Name: "res-1"}
	assert.NotPanics(t, func() {
		service.podEventHandler(ctx).DeleteFunc(cache.DeletedFinalStateUnknown{Key: namespace + "/res-1", Obj: &corev1.Pod{ObjectMeta: metaObj}})
		service.deploymentEventHandler(ctx).DeleteFunc(cache.DeletedFinalStateUnknown{Key: namespace + "/res-1", Obj: &appsv1.Deployment{ObjectMeta: metaObj}})
		service.statefulSetEventHandler(ctx).DeleteFunc(cache.DeletedFinalStateUnknown{Key: namespace + "/res-1", Obj: &appsv1.StatefulSet{ObjectMeta: metaObj}})
		service.daemonSetEventHandler(ctx).DeleteFunc(cache.DeletedFinalStateUnknown{Key: namespace + "/res-1", Obj: &appsv1.DaemonSet{ObjectMeta: metaObj}})
		service.serviceEventHandler(ctx).DeleteFunc(cache.DeletedFinalStateUnknown{Key: namespace + "/res-1", Obj: &corev1.Service{ObjectMeta: metaObj}})
		service.ingressEventHandler(ctx).DeleteFunc(cache.DeletedFinalStateUnknown{Key: namespace + "/res-1", Obj: &networkingv1.Ingress{ObjectMeta: metaObj}})
		service.configMapEventHandler(ctx).DeleteFunc(cache.DeletedFinalStateUnknown{Key: namespace + "/res-1", Obj: &corev1.ConfigMap{ObjectMeta: metaObj}})
	})
}
