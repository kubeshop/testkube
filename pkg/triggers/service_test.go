package triggers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
)

func TestService_addTrigger(t *testing.T) {

	s := Service{triggerStatus: make(map[statusKey]*triggerStatus)}

	testTrigger := testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Name: "test-trigger-1", Namespace: "testkube"},
	}
	s.addTrigger(&testTrigger)

	assert.Len(t, s.triggerStatus, 1)
	key := newStatusKey("testkube", "test-trigger-1")
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
	s.addTrigger(&testTrigger1)
	s.addTrigger(&testTrigger2)

	assert.Len(t, s.triggerStatus, 2)

	s.removeTrigger(&testTrigger1)

	assert.Len(t, s.triggerStatus, 1)
	key := newStatusKey("testkube", "test-trigger-2")
	assert.NotNil(t, s.triggerStatus[key])
	deletedKey := newStatusKey("testkube", "test-trigger-1")
	assert.Nil(t, s.triggerStatus[deletedKey])
}

func TestService_updateTrigger(t *testing.T) {

	s := Service{triggerStatus: make(map[statusKey]*triggerStatus)}

	oldTestTrigger := testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "test-trigger-1"},
		Spec:       testtriggersv1.TestTriggerSpec{Event: "created"},
	}
	s.addTrigger(&oldTestTrigger)

	newTestTrigger := testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "test-trigger-1"},
		Spec:       testtriggersv1.TestTriggerSpec{Event: "modified"},
	}

	s.updateTrigger(&newTestTrigger)

	assert.Len(t, s.triggerStatus, 1)
	key := newStatusKey("testkube", "test-trigger-1")
	assert.NotNil(t, s.triggerStatus[key])
}
