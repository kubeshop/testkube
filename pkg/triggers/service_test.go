package triggers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	intconfig "github.com/kubeshop/testkube/internal/config"
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

// TestService_triggerTargetsThisAgent pins the listener-pinning semantics of
// the helper that gates addTrigger/updateTrigger. The OSS fallback (no
// proContext, or proContext with empty ID) intentionally degrades to
// "broadcast" so standalone agents don't silently drop CRDs they could
// otherwise serve.
func TestService_triggerTargetsThisAgent(t *testing.T) {
	withProContext := func(id string) *intconfig.ProContext {
		return &intconfig.ProContext{Agent: intconfig.ProContextAgent{ID: id}}
	}

	cases := map[string]struct {
		listenerIDs []string
		proContext  *intconfig.ProContext
		want        bool
	}{
		"empty list — broadcast to every listener": {
			listenerIDs: nil,
			proContext:  withProContext("tkcagent_a"),
			want:        true,
		},
		"list contains this agent's ID — register": {
			listenerIDs: []string{"tkcagent_a", "tkcagent_b"},
			proContext:  withProContext("tkcagent_a"),
			want:        true,
		},
		"list does not contain this agent's ID — skip": {
			listenerIDs: []string{"tkcagent_b", "tkcagent_c"},
			proContext:  withProContext("tkcagent_a"),
			want:        false,
		},
		"OSS / no proContext — fall back to broadcast": {
			listenerIDs: []string{"tkcagent_b"},
			proContext:  nil,
			want:        true,
		},
		"proContext present but agent ID empty — fall back to broadcast": {
			listenerIDs: []string{"tkcagent_b"},
			proContext:  withProContext(""),
			want:        true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			s := &Service{proContext: tc.proContext}
			got := s.triggerTargetsThisAgent(tc.listenerIDs)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestService_addTrigger_listenerPinning verifies that addTrigger respects
// listenerAgentIds: triggers pinned to other agents are silently dropped (no
// row in triggerStatus, no informer started) — broadcast and self-pinned
// triggers register normally.
func TestService_addTrigger_listenerPinning(t *testing.T) {
	const myID = "tkcagent_self"
	cases := map[string]struct {
		listenerIDs    []string
		wantRegistered bool
	}{
		"empty — broadcast registers": {nil, true},
		"includes self — registers":   {[]string{myID, "tkcagent_other"}, true},
		"excludes self — skipped":     {[]string{"tkcagent_other"}, false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			s := &Service{
				triggerStatus: make(map[statusKey]*triggerStatus),
				proContext:    &intconfig.ProContext{Agent: intconfig.ProContextAgent{ID: myID}},
				logger:        log.DefaultLogger,
			}
			trig := &testtriggersv1.TestTrigger{
				ObjectMeta: metav1.ObjectMeta{Namespace: "tk-dev", Name: "t1"},
				Spec:       testtriggersv1.TestTriggerSpec{ListenerAgentIds: tc.listenerIDs},
			}
			s.addTrigger(context.Background(), trig)
			key := newStatusKey(triggerSourceV1, "tk-dev", "t1")
			if tc.wantRegistered {
				assert.NotNil(t, s.triggerStatus[key], "expected trigger registered")
			} else {
				assert.Nil(t, s.triggerStatus[key], "expected trigger skipped")
				assert.Len(t, s.triggerStatus, 0)
			}
		})
	}
}

// TestService_updateTrigger_listenerPinning verifies the transition matrix
// when an existing trigger's listenerAgentIds changes (or doesn't), covering
// the four core registered/targeted permutations plus two broadcast edges.
// Broadcast (nil/empty listenerAgentIds) means "every listener fires" — so
// transitioning broadcast → other-pin should drop us, and self-pin → broadcast
// should keep us registered.
func TestService_updateTrigger_listenerPinning(t *testing.T) {
	const myID = "tkcagent_self"
	const otherID = "tkcagent_other"
	mkSvc := func() *Service {
		return &Service{
			triggerStatus: make(map[statusKey]*triggerStatus),
			proContext:    &intconfig.ProContext{Agent: intconfig.ProContextAgent{ID: myID}},
			logger:        log.DefaultLogger,
		}
	}
	mkTrigger := func(listenerIDs []string, event string) *testtriggersv1.TestTrigger {
		return &testtriggersv1.TestTrigger{
			ObjectMeta: metav1.ObjectMeta{Namespace: "tk-dev", Name: "t1"},
			Spec: testtriggersv1.TestTriggerSpec{
				ListenerAgentIds: listenerIDs,
				Event:            testtriggersv1.TestTriggerEvent(event),
			},
		}
	}
	key := newStatusKey(triggerSourceV1, "tk-dev", "t1")

	cases := []struct {
		name           string
		preRegister    bool     // call addTrigger before update
		preRegisterAs  []string // listenerAgentIds for the pre-register call
		updatePin      []string // listenerAgentIds on the update payload
		wantRegistered bool
	}{
		{
			name:        "self-pin → self-pin: in-place update",
			preRegister: true, preRegisterAs: []string{myID},
			updatePin: []string{myID}, wantRegistered: true,
		},
		{
			name:        "self-pin → other-pin: drop",
			preRegister: true, preRegisterAs: []string{myID},
			updatePin: []string{otherID}, wantRegistered: false,
		},
		{
			name:        "unregistered → self-pin: register fresh",
			preRegister: false,
			updatePin:   []string{myID}, wantRegistered: true,
		},
		{
			name:        "unregistered → other-pin: no-op",
			preRegister: false,
			updatePin:   []string{otherID}, wantRegistered: false,
		},
		{
			name:        "self-pin → broadcast: still registered",
			preRegister: true, preRegisterAs: []string{myID},
			updatePin: nil, wantRegistered: true,
		},
		{
			name:        "broadcast → other-pin: drop",
			preRegister: true, preRegisterAs: nil,
			updatePin: []string{otherID}, wantRegistered: false,
		},
	}

	// The update always writes event="modified"; pre-register (when used)
	// always writes event="created". When the trigger ends up registered,
	// the post-update event must reflect the in-place write — confirms we
	// didn't just leave a stale entry from the addTrigger call.
	const updateEvent = "modified"
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := mkSvc()
			if tc.preRegister {
				s.addTrigger(context.Background(), mkTrigger(tc.preRegisterAs, "created"))
				assert.NotNil(t, s.triggerStatus[key], "pre-condition: addTrigger should have registered")
			}
			s.updateTrigger(context.Background(), mkTrigger(tc.updatePin, updateEvent))
			if tc.wantRegistered {
				assert.NotNil(t, s.triggerStatus[key], "expected trigger registered")
				assert.Equal(t, updateEvent, s.triggerStatus[key].trigger.Event,
					"expected in-place update to replace event")
			} else {
				assert.Nil(t, s.triggerStatus[key], "expected trigger absent")
			}
		})
	}
}
