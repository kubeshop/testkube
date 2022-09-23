package triggers

import (
	"context"
	"testing"
	"time"

	testtriggersv1 "github.com/kubeshop/testkube-operator/apis/testtriggers/v1"
	faketestkube "github.com/kubeshop/testkube-operator/pkg/clientset/versioned/fake"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestService_runWatcher(t *testing.T) {
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
		}

		s.runWatcher(ctx)

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

		assert.Len(t, s.triggers, 1)
		assert.Len(t, s.triggerStatus, 1)
		assert.Equal(t, &testTrigger, s.triggers[0])
		key := newStatusKey(testNamespace, "test-trigger-1")
		assert.NotNil(t, s.triggerStatus[key])

		err = testKubeClientset.TestsV1().TestTriggers(testNamespace).Delete(ctx, "test-trigger-1", metav1.DeleteOptions{})
		assert.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		assert.Len(t, s.triggers, 0)
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
		testExecutorF := func(ctx context.Context, trigger *testtriggersv1.TestTrigger) error {
			assert.Equal(t, testNamespace, trigger.Namespace)
			assert.Equal(t, "test-trigger-2", trigger.Name)
			match = true
			return nil
		}
		s := &Service{
			executor:          testExecutorF,
			triggerStatus:     make(map[statusKey]*triggerStatus),
			clientset:         clientset,
			testKubeClientset: testKubeClientset,
			logger:            log.DefaultLogger,
		}

		s.runWatcher(ctx)

		time.Sleep(100 * time.Millisecond)

		testTrigger := testtriggersv1.TestTrigger{
			ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace, Name: "test-trigger-2"},
			Spec: testtriggersv1.TestTriggerSpec{
				Resource:         "pod",
				ResourceSelector: testtriggersv1.TestTriggerSelector{Name: "test-pod"},
				Event:            "created",
				Action:           "run",
				Execution:        "test",
				TestSelector:     testtriggersv1.TestTriggerSelector{Name: "some-test"},
			},
		}
		createdTestTrigger, err := testKubeClientset.TestsV1().TestTriggers(testNamespace).Create(ctx, &testTrigger, metav1.CreateOptions{})
		assert.NotNil(t, createdTestTrigger)
		assert.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		assert.Len(t, s.triggers, 1)
		assert.Len(t, s.triggerStatus, 1)
		assert.Equal(t, &testTrigger, s.triggers[0])
		key := newStatusKey(testNamespace, "test-trigger-2")
		assert.NotNil(t, s.triggerStatus[key])

		testPod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace, Name: "test-pod"}}
		_, err = clientset.CoreV1().Pods(testNamespace).Create(ctx, &testPod, metav1.CreateOptions{})
		assert.NoError(t, err)

		time.Sleep(100 * time.Millisecond)
		assert.True(t, match, "pod created event should match the test trigger condition")
	})
}
