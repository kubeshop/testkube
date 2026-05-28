package triggers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	"github.com/kubeshop/testkube/pkg/log"
)

func TestService_addTrigger(t *testing.T) {

	s := Service{triggerStatus: make(map[statusKey]*triggerStatus)}

	testTrigger := testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Name: "test-trigger-1", Namespace: "testkube"},
	}
	s.addTrigger(context.Background(), &testTrigger)

	assert.Len(t, s.triggerStatus, 1)
	key := newStatusKey(triggerSourceV1, "testkube", "test-trigger-1")
	assert.NotNil(t, s.triggerStatus[key])
}

func TestService_removeTrigger(t *testing.T) {

	s := Service{triggerStatus: make(map[statusKey]*triggerStatus)}

	testTrigger1 := testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Name: "test-trigger-1", Namespace: "testkube"},
	}
	testTrigger2 := testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Name: "test-trigger-2", Namespace: "testkube"},
	}
	s.addTrigger(context.Background(), &testTrigger1)
	s.addTrigger(context.Background(), &testTrigger2)

	assert.Len(t, s.triggerStatus, 2)

	s.removeTrigger(&testTrigger1)

	assert.Len(t, s.triggerStatus, 1)
	key := newStatusKey(triggerSourceV1, "testkube", "test-trigger-2")
	assert.NotNil(t, s.triggerStatus[key])
	deletedKey := newStatusKey(triggerSourceV1, "testkube", "test-trigger-1")
	assert.Nil(t, s.triggerStatus[deletedKey])
}

func TestService_updateTrigger(t *testing.T) {

	s := Service{triggerStatus: make(map[statusKey]*triggerStatus)}

	oldTestTrigger := testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "test-trigger-1"},
		Spec:       testtriggersv1.TestTriggerSpec{Event: "created"},
	}
	s.addTrigger(context.Background(), &oldTestTrigger)

	newTestTrigger := testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "test-trigger-1"},
		Spec:       testtriggersv1.TestTriggerSpec{Event: "modified"},
	}

	s.updateTrigger(context.Background(), &newTestTrigger)

	assert.Len(t, s.triggerStatus, 1)
	key := newStatusKey(triggerSourceV1, "testkube", "test-trigger-1")
	assert.NotNil(t, s.triggerStatus[key])
}

func TestWithClusterID(t *testing.T) {
	t.Run("non-empty overrides default", func(t *testing.T) {
		s := &Service{clusterID: DefaultClusterID}
		opt := WithClusterID("my-custom-id")
		opt(s)
		assert.Equal(t, "my-custom-id", s.clusterID)
	})

	t.Run("empty string does not override", func(t *testing.T) {
		s := &Service{clusterID: DefaultClusterID}
		opt := WithClusterID("")
		opt(s)
		assert.Equal(t, DefaultClusterID, s.clusterID)
	})
}

func TestService_ensureDynamicInformerForTrigger_SkipsContentResourceRef(t *testing.T) {
	s := Service{
		dynamicManager: newTestDynamicInformerManager(t),
		logger:         log.DefaultLogger,
	}

	testTrigger := testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Name: "test-trigger-1", Namespace: "testkube"},
		Spec: testtriggersv1.TestTriggerSpec{
			ResourceRef: &testtriggersv1.TestTriggerResourceRef{Kind: string(testtriggersv1.TestTriggerResourceContent)},
		},
	}

	assert.NotPanics(t, func() {
		s.ensureDynamicInformerForTrigger(context.Background(), &testTrigger, newStatusKey(triggerSourceV1, "testkube", "test-trigger-1"))
	})
	assert.Empty(t, s.dynamicManager.informers)
}
