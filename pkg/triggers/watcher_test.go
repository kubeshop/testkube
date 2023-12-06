package triggers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	testtriggersv1 "github.com/kubeshop/testkube-operator/api/testtriggers/v1"
	faketestkube "github.com/kubeshop/testkube-operator/pkg/clientset/versioned/fake"
	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/log"
)

func TestService_runWatcher_lease(t *testing.T) {
	t.Parallel()

	t.Run("create and delete a test trigger", func(t *testing.T) {
		t.Parallel()

		clientset := fake.NewSimpleClientset()
		testKubeClientset := faketestkube.NewSimpleClientset()

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		s := &Service{
			triggerStatus:     make(map[statusKey]*triggerStatus),
			clientset:         clientset,
			testKubeClientset: testKubeClientset,
			logger:            log.DefaultLogger,
			informers:         newK8sInformers(clientset, testKubeClientset, "", []string{}),
			eventsBus:         &bus.EventBusMock{},
		}

		leaseChan := make(chan bool)
		go func() { time.Sleep(50 * time.Millisecond); leaseChan <- true }()
		go s.runWatcher(ctx, leaseChan)

		time.Sleep(100 * time.Millisecond)

		testNamespace := "testkube"
		testTrigger := testtriggersv1.TestTrigger{
			ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace, Name: "test-trigger-1"},
			Spec:       testtriggersv1.TestTriggerSpec{Event: "created"},
		}
		createdTestTrigger, err := testKubeClientset.TestsV1().TestTriggers(testNamespace).Create(ctx, &testTrigger, metav1.CreateOptions{})
		assert.NotNil(t, createdTestTrigger)
		assert.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		assert.Len(t, s.triggerStatus, 1)
		key := newStatusKey(testNamespace, "test-trigger-1")
		assert.Contains(t, s.triggerStatus, key)

		err = testKubeClientset.TestsV1().TestTriggers(testNamespace).Delete(ctx, "test-trigger-1", metav1.DeleteOptions{})
		assert.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		assert.Len(t, s.triggerStatus, 0)
	})

	t.Run("create a test trigger for pod created and match event on pod creation", func(t *testing.T) {
		t.Parallel()

		clientset := fake.NewSimpleClientset()
		testKubeClientset := faketestkube.NewSimpleClientset()

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		testNamespace := "testkube"

		match := false
		testExecutorF := func(ctx context.Context, e *watcherEvent, trigger *testtriggersv1.TestTrigger) error {
			assert.Equal(t, testNamespace, trigger.Namespace)
			assert.Equal(t, "test-trigger-2", trigger.Name)
			match = true
			return nil
		}
		s := &Service{
			triggerExecutor:   testExecutorF,
			identifier:        "testkube-api",
			clusterID:         "testkube",
			triggerStatus:     make(map[statusKey]*triggerStatus),
			clientset:         clientset,
			testKubeClientset: testKubeClientset,
			logger:            log.DefaultLogger,
			informers:         newK8sInformers(clientset, testKubeClientset, "", []string{}),
			eventsBus:         &bus.EventBusMock{},
		}

		leaseChan := make(chan bool)
		go func() { time.Sleep(50 * time.Millisecond); leaseChan <- true }()
		go s.runWatcher(ctx, leaseChan)

		time.Sleep(50 * time.Millisecond)
		leaseChan <- true
		time.Sleep(50 * time.Millisecond)

		testTrigger := testtriggersv1.TestTrigger{
			ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace, Name: "test-trigger-2"},
			Spec: testtriggersv1.TestTriggerSpec{
				Resource:          "pod",
				ResourceSelector:  testtriggersv1.TestTriggerSelector{Name: "test-pod"},
				Event:             "created",
				Action:            "run",
				Execution:         "test",
				ConcurrencyPolicy: "allow",
				TestSelector:      testtriggersv1.TestTriggerSelector{Name: "some-test"},
			},
		}
		createdTestTrigger, err := testKubeClientset.TestsV1().TestTriggers(testNamespace).Create(ctx, &testTrigger, metav1.CreateOptions{})
		assert.NotNil(t, createdTestTrigger)
		assert.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		assert.Len(t, s.triggerStatus, 1)
		key := newStatusKey(testNamespace, "test-trigger-2")
		assert.Contains(t, s.triggerStatus, key)

		testPod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace, Name: "test-pod"}}
		_, err = clientset.CoreV1().Pods(testNamespace).Create(ctx, &testPod, metav1.CreateOptions{})
		assert.NoError(t, err)

		time.Sleep(100 * time.Millisecond)
		assert.True(t, match, "pod created event should match the test trigger condition")
	})
}

func TestService_runWatcher_noLease(t *testing.T) {
	t.Parallel()

	t.Run("watcher will not start if lease is not acquired", func(t *testing.T) {
		t.Parallel()

		clientset := fake.NewSimpleClientset()
		testKubeClientset := faketestkube.NewSimpleClientset()

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		s := &Service{
			triggerStatus:     make(map[statusKey]*triggerStatus),
			identifier:        "testkube-api",
			clusterID:         "testkube",
			clientset:         clientset,
			testKubeClientset: testKubeClientset,
			logger:            log.DefaultLogger,
			informers:         newK8sInformers(clientset, testKubeClientset, "", []string{}),
			eventsBus:         &bus.EventBusMock{},
		}

		leaseChan := make(chan bool)
		go s.runWatcher(ctx, leaseChan)

		time.Sleep(50 * time.Millisecond)
		leaseChan <- false
		time.Sleep(50 * time.Millisecond)

		testNamespace := "testkube"
		testTrigger := testtriggersv1.TestTrigger{
			ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace, Name: "test-trigger-1"},
			Spec:       testtriggersv1.TestTriggerSpec{Event: "created"},
		}
		createdTestTrigger, err := testKubeClientset.TestsV1().TestTriggers(testNamespace).Create(ctx, &testTrigger, metav1.CreateOptions{})
		assert.NotNil(t, createdTestTrigger)
		assert.NoError(t, err)

		time.Sleep(50 * time.Millisecond)

		assert.Len(t, s.triggerStatus, 0)
	})

	t.Run("watcher should stop when lease is lost", func(t *testing.T) {
		t.Parallel()

		clientset := fake.NewSimpleClientset()
		testKubeClientset := faketestkube.NewSimpleClientset()

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		s := &Service{
			triggerStatus:     make(map[statusKey]*triggerStatus),
			identifier:        "testkube-api",
			clusterID:         "testkube",
			clientset:         clientset,
			testKubeClientset: testKubeClientset,
			logger:            log.DefaultLogger,
			informers:         newK8sInformers(clientset, testKubeClientset, "", []string{}),
			eventsBus:         &bus.EventBusMock{},
		}

		leaseChan := make(chan bool)
		go s.runWatcher(ctx, leaseChan)

		time.Sleep(50 * time.Millisecond)
		leaseChan <- true
		time.Sleep(50 * time.Millisecond)
		leaseChan <- false
		time.Sleep(50 * time.Millisecond)

		testNamespace := "testkube"
		testTrigger := testtriggersv1.TestTrigger{
			ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace, Name: "test-trigger-1"},
			Spec:       testtriggersv1.TestTriggerSpec{Event: "created"},
		}
		createdTestTrigger, err := testKubeClientset.TestsV1().TestTriggers(testNamespace).Create(ctx, &testTrigger, metav1.CreateOptions{})
		assert.NotNil(t, createdTestTrigger)
		assert.NoError(t, err)

		time.Sleep(50 * time.Millisecond)

		assert.Len(t, s.triggerStatus, 0)
	})

	t.Run("watcher should successfully restart on a newly acquired lease", func(t *testing.T) {
		t.Parallel()

		clientset := fake.NewSimpleClientset()
		testKubeClientset := faketestkube.NewSimpleClientset()

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		s := &Service{
			triggerStatus:     make(map[statusKey]*triggerStatus),
			identifier:        "testkube-api",
			clusterID:         "testkube",
			clientset:         clientset,
			testKubeClientset: testKubeClientset,
			logger:            log.DefaultLogger,
			informers:         newK8sInformers(clientset, testKubeClientset, "", []string{}),
			eventsBus:         &bus.EventBusMock{},
		}

		leaseChan := make(chan bool)
		go s.runWatcher(ctx, leaseChan)

		time.Sleep(50 * time.Millisecond)
		leaseChan <- true
		time.Sleep(50 * time.Millisecond)
		leaseChan <- false
		time.Sleep(50 * time.Millisecond)
		leaseChan <- true
		time.Sleep(50 * time.Millisecond)

		testNamespace := "testkube"
		testTrigger := testtriggersv1.TestTrigger{
			ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace, Name: "test-trigger-1"},
			Spec:       testtriggersv1.TestTriggerSpec{Event: "created"},
		}
		createdTestTrigger, err := testKubeClientset.TestsV1().TestTriggers(testNamespace).Create(ctx, &testTrigger, metav1.CreateOptions{})
		assert.NotNil(t, createdTestTrigger)
		assert.NoError(t, err)

		time.Sleep(50 * time.Millisecond)

		assert.Len(t, s.triggerStatus, 1)
		key := newStatusKey(testNamespace, "test-trigger-1")
		assert.Contains(t, s.triggerStatus, key)
	})
}
