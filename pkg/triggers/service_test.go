package triggers

import (
	"testing"

	v1 "github.com/kubeshop/testkube-operator/apis/testtriggers/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestService_addTrigger(t *testing.T) {
	t.Parallel()

	s := Service{triggerStatus: make(map[statusKey]*triggerStatus)}

	testTrigger := v1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Name: "test-trigger-1", Namespace: "testkube"},
	}
	s.addTrigger(&testTrigger)

	assert.Len(t, s.triggers, 1)
	assert.Len(t, s.triggerStatus, 1)
	assert.Equal(t, &testTrigger, s.triggers[0])
	key := newStatusKey("testkube", "test-trigger-1")
	assert.NotNil(t, s.triggerStatus[key])
}

func TestService_removeTrigger(t *testing.T) {
	t.Parallel()

	s := Service{triggerStatus: make(map[statusKey]*triggerStatus)}

	testTrigger1 := v1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Name: "test-trigger-1", Namespace: "testkube"},
	}
	testTrigger2 := v1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Name: "test-trigger-2", Namespace: "testkube"},
	}
	s.addTrigger(&testTrigger1)
	s.addTrigger(&testTrigger2)

	assert.Len(t, s.triggers, 2)
	assert.Len(t, s.triggerStatus, 2)

	s.removeTrigger(&testTrigger1)

	assert.Len(t, s.triggers, 1)
	assert.Len(t, s.triggerStatus, 1)
	assert.Equal(t, &testTrigger2, s.triggers[0])
	key := newStatusKey("testkube", "test-trigger-2")
	assert.NotNil(t, s.triggerStatus[key])
	deletedKey := newStatusKey("testkube", "test-trigger-1")
	assert.Nil(t, s.triggerStatus[deletedKey])
}

func TestService_updateTrigger(t *testing.T) {
	t.Parallel()

	s := Service{triggerStatus: make(map[statusKey]*triggerStatus)}

	oldTestTrigger := v1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "test-trigger-1"},
		Spec:       v1.TestTriggerSpec{Event: "created"},
	}
	s.addTrigger(&oldTestTrigger)

	newTestTrigger := v1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "test-trigger-1"},
		Spec:       v1.TestTriggerSpec{Event: "modified"},
	}

	s.updateTrigger(&newTestTrigger)

	assert.Len(t, s.triggers, 1)
	assert.Len(t, s.triggerStatus, 1)
	assert.Equal(t, "modified", s.triggers[0].Spec.Event)
	key := newStatusKey("testkube", "test-trigger-1")
	assert.NotNil(t, s.triggerStatus[key])
}
