package triggers

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeshop/testkube/pkg/log"
)

var testGVR = schema.GroupVersionResource{Group: "argoproj.io", Version: "v1alpha1", Resource: "rollouts"}

func newTestDynamicInformerManager(t *testing.T, extraGVRs ...schema.GroupVersionResource) *dynamicInformerManager {
	t.Helper()
	gvrToListKind := map[schema.GroupVersionResource]string{
		testGVR: "RolloutList",
	}
	for _, gvr := range extraGVRs {
		gvrToListKind[gvr] = gvr.Resource + "List"
	}
	scheme := runtime.NewScheme()
	client := fake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind)
	return newDynamicInformerManager(client, nil, []string{"default"}, log.DefaultLogger)
}

func TestDynamicInformerManager_ensureInformer_startsOnce(t *testing.T) {
	m := newTestDynamicInformerManager(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m.ensureInformer(ctx, testGVR, "v1:default/my-trigger", cache.ResourceEventHandlerFuncs{})

	assert.Len(t, m.informers, 1, "informer entry should be created")
	entry := m.informers[testGVR.String()]
	assert.NotNil(t, entry)
	assert.Len(t, entry.refs, 1, "single trigger reference")
	assert.Contains(t, entry.refs, "v1:default/my-trigger")
}

func TestDynamicInformerManager_ensureInformer_idempotentPerTrigger(t *testing.T) {
	m := newTestDynamicInformerManager(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Same trigger key, called repeatedly — simulates informer resync
	// re-invoking AddFunc for an already-tracked trigger.
	for i := 0; i < 5; i++ {
		m.ensureInformer(ctx, testGVR, "v1:default/my-trigger", cache.ResourceEventHandlerFuncs{})
	}

	entry := m.informers[testGVR.String()]
	assert.Len(t, entry.refs, 1, "refs must not inflate on repeated ensures by same trigger")
}

func TestDynamicInformerManager_ensureInformer_multipleTriggersSameGVR(t *testing.T) {
	m := newTestDynamicInformerManager(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m.ensureInformer(ctx, testGVR, "v1:default/trigger-a", cache.ResourceEventHandlerFuncs{})
	m.ensureInformer(ctx, testGVR, "v1:default/trigger-b", cache.ResourceEventHandlerFuncs{})

	assert.Len(t, m.informers, 1, "shared informer for same GVR")
	entry := m.informers[testGVR.String()]
	assert.Len(t, entry.refs, 2)
	assert.Contains(t, entry.refs, "v1:default/trigger-a")
	assert.Contains(t, entry.refs, "v1:default/trigger-b")
}

func TestDynamicInformerManager_releaseInformer_stopsOnLastReference(t *testing.T) {
	m := newTestDynamicInformerManager(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m.ensureInformer(ctx, testGVR, "v1:default/trigger-a", cache.ResourceEventHandlerFuncs{})
	m.ensureInformer(ctx, testGVR, "v1:default/trigger-b", cache.ResourceEventHandlerFuncs{})

	stopCh := m.informers[testGVR.String()].stopCh

	m.releaseInformer(testGVR, "v1:default/trigger-a")
	assert.Len(t, m.informers, 1, "one reference remains, informer stays up")
	assert.Len(t, m.informers[testGVR.String()].refs, 1)

	m.releaseInformer(testGVR, "v1:default/trigger-b")
	assert.Len(t, m.informers, 0, "last reference released, entry removed")

	select {
	case <-stopCh:
	default:
		t.Fatal("stopCh should be closed after last reference release")
	}
}

func TestDynamicInformerManager_releaseInformer_ignoresUnknownTrigger(t *testing.T) {
	m := newTestDynamicInformerManager(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m.ensureInformer(ctx, testGVR, "v1:default/trigger-a", cache.ResourceEventHandlerFuncs{})

	// Release with a key that was never registered must not decrement/remove.
	m.releaseInformer(testGVR, "v1:default/phantom")

	entry, ok := m.informers[testGVR.String()]
	assert.True(t, ok)
	assert.Len(t, entry.refs, 1)
	assert.Contains(t, entry.refs, "v1:default/trigger-a")
}

func TestDynamicInformerManager_releaseInformer_ignoresUnknownGVR(t *testing.T) {
	m := newTestDynamicInformerManager(t)
	unknownGVR := schema.GroupVersionResource{Group: "unknown.io", Version: "v1", Resource: "things"}

	// No panic, no mutation.
	m.releaseInformer(unknownGVR, "v1:default/x")
	assert.Len(t, m.informers, 0)
}

func TestDynamicInformerManager_stopAll_cleansUpAllInformers(t *testing.T) {
	secondGVR := schema.GroupVersionResource{Group: "kafka.strimzi.io", Version: "v1beta2", Resource: "kafkatopics"}
	m := newTestDynamicInformerManager(t, secondGVR)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m.ensureInformer(ctx, testGVR, "t1", cache.ResourceEventHandlerFuncs{})
	m.ensureInformer(ctx, secondGVR, "t2", cache.ResourceEventHandlerFuncs{})

	stopChs := []chan struct{}{
		m.informers[testGVR.String()].stopCh,
		m.informers[secondGVR.String()].stopCh,
	}

	m.stopAll()

	assert.Len(t, m.informers, 0)
	for i, ch := range stopChs {
		select {
		case <-ch:
		default:
			t.Fatalf("stopCh[%d] should be closed after stopAll", i)
		}
	}
}

func TestDynamicInformerManager_concurrentEnsureRelease(t *testing.T) {
	m := newTestDynamicInformerManager(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 50 distinct trigger keys concurrently ensure+release on the same GVR.
	// Correct ref-counting means the final state has no informer entries.
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := triggerKeyForIndex(idx)
			m.ensureInformer(ctx, testGVR, key, cache.ResourceEventHandlerFuncs{})
			m.releaseInformer(testGVR, key)
		}(i)
	}
	wg.Wait()

	assert.Len(t, m.informers, 0, "all references released, no informer entries should remain")
}

func triggerKeyForIndex(i int) string {
	return "v1:default/trigger-" + string(rune('a'+i%26)) + "-" + string(rune('0'+i%10))
}

// --- resolveGVR builtin-path tests ---
// The discovery-cache recovery path (Reset + retry on NoMatch) requires a
// working REST mapper against a fake discovery client, which is exercised in
// integration tests. Here we cover the synchronous builtin-fast-path that
// does not require a mapper at all.

func TestResolveGVR_builtin_skipsMapperForBareKind(t *testing.T) {
	tests := map[string]struct {
		kind    string
		wantGVR schema.GroupVersionResource
	}{
		"deployment": {
			kind:    "Deployment",
			wantGVR: schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"},
		},
		"configmap lowercase": {
			kind:    "configmap",
			wantGVR: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"},
		},
		"ingress": {
			kind:    "Ingress",
			wantGVR: schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// A nil mapper is safe because the builtin path returns before calling it.
			gvr, err := resolveGVR(nil, "", "", tc.kind)
			assert.NoError(t, err)
			assert.Equal(t, tc.wantGVR, gvr)
		})
	}
}

func TestResolveGVR_builtin_honorsExplicitMatchingGroup(t *testing.T) {
	gvr, err := resolveGVR(nil, "apps", "v1", "Deployment")
	assert.NoError(t, err)
	assert.Equal(t, schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}, gvr)
}
