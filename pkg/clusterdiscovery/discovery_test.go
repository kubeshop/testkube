package clusterdiscovery

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authzv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func TestCanWatch(t *testing.T) {
	rules := []authzv1.ResourceRule{
		{Verbs: []string{"list", "watch"}, APIGroups: []string{""}, Resources: []string{"pods"}},
		{Verbs: []string{"*"}, APIGroups: []string{"cert-manager.io"}, Resources: []string{"certificates"}},
		{Verbs: []string{"list", "watch"}, APIGroups: []string{"*"}, Resources: []string{"*"}},
	}

	t.Run("exact match", func(t *testing.T) {
		assert.True(t, canWatch(rules[:1], "", "pods"))
	})
	t.Run("wildcard verb", func(t *testing.T) {
		assert.True(t, canWatch(rules[1:2], "cert-manager.io", "certificates"))
	})
	t.Run("wildcard group and resource", func(t *testing.T) {
		assert.True(t, canWatch(rules[2:3], "kafka.strimzi.io", "kafkatopics"))
	})
	t.Run("no matching rule", func(t *testing.T) {
		assert.False(t, canWatch(rules[:2], "batch", "jobs"))
	})
	t.Run("list-only rule without watch", func(t *testing.T) {
		listOnly := []authzv1.ResourceRule{
			{Verbs: []string{"list"}, APIGroups: []string{""}, Resources: []string{"pods"}},
		}
		assert.False(t, canWatch(listOnly, "", "pods"))
	})
}

func TestContainsOrWildcard(t *testing.T) {
	assert.True(t, containsOrWildcard([]string{"get", "list", "watch"}, "watch"))
	assert.True(t, containsOrWildcard([]string{"*"}, "watch"))
	assert.False(t, containsOrWildcard([]string{"get", "list"}, "watch"))
	assert.False(t, containsOrWildcard(nil, "watch"))
}

// preferredFakeDiscovery overrides ServerPreferredResources, which the
// upstream client-go fake stubs to (nil, nil) - and which is the only
// discovery method Discoverer consumes.
type preferredFakeDiscovery struct {
	*fakediscovery.FakeDiscovery
	resources []*metav1.APIResourceList
}

func (p *preferredFakeDiscovery) ServerPreferredResources() ([]*metav1.APIResourceList, error) {
	return p.resources, nil
}

func newDiscovererWithFake(t *testing.T, lists []*metav1.APIResourceList, namespace string, rules []authzv1.ResourceRule, incomplete bool) (*Discoverer, *string) {
	t.Helper()
	cs := fake.NewSimpleClientset()
	var captured string
	cs.PrependReactor("create", "selfsubjectrulesreviews", func(action ktesting.Action) (bool, runtime.Object, error) {
		ca, ok := action.(ktesting.CreateAction)
		require.True(t, ok)
		ssrr, ok := ca.GetObject().(*authzv1.SelfSubjectRulesReview)
		require.True(t, ok)
		captured = ssrr.Spec.Namespace
		return true, &authzv1.SelfSubjectRulesReview{
			Status: authzv1.SubjectRulesReviewStatus{ResourceRules: rules, Incomplete: incomplete},
		}, nil
	})
	disc := &preferredFakeDiscovery{
		FakeDiscovery: cs.Discovery().(*fakediscovery.FakeDiscovery),
		resources:     lists,
	}
	d := NewFromInterfaces(discovery.DiscoveryInterface(disc), cs, namespace)
	return d, &captured
}

func TestDiscoverer_List(t *testing.T) {
	// Fixture: one core watchable resource, one CRD-style watchable resource,
	// plus a subresource and a non-watchable resource that must be dropped.
	lists := []*metav1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{Name: "pods", Kind: "Pod", Namespaced: true, Verbs: []string{"get", "list", "watch"}},
				{Name: "pods/log", Kind: "Pod", Namespaced: true, Verbs: []string{"get"}},
				{Name: "bindings", Kind: "Binding", Namespaced: true, Verbs: []string{"create"}},
			},
		},
		{
			GroupVersion: "argoproj.io/v1alpha1",
			APIResources: []metav1.APIResource{
				{Name: "rollouts", Kind: "Rollout", Namespaced: true, Verbs: []string{"get", "list", "watch"}},
			},
		},
	}
	podsRule := []authzv1.ResourceRule{
		{Verbs: []string{"list", "watch"}, APIGroups: []string{""}, Resources: []string{"pods"}},
	}

	tests := map[string]struct {
		namespace     string
		rules         []authzv1.ResourceRule
		incomplete    bool
		wantNamespace string
		// wantCanWatch keyed by Kind. nil = skip per-resource assertions.
		wantCanWatch map[string]bool
		// wantKinds asserts the post-filter Kind set. nil = skip.
		wantKinds []string
	}{
		"namespace plumbing: configured value is passed to SelfSubjectRulesReview": {
			namespace:     "testkube",
			wantNamespace: "testkube",
		},
		"namespace plumbing: empty falls back to NamespaceDefault": {
			namespace:     "",
			wantNamespace: metav1.NamespaceDefault,
		},
		"CanWatch reflects the authorizer's resource rules": {
			namespace:     "testkube",
			rules:         podsRule,
			wantNamespace: "testkube",
			wantKinds:     []string{"Pod", "Rollout"},
			wantCanWatch:  map[string]bool{"Pod": true, "Rollout": false},
		},
		"Incomplete authorizer result forces CanWatch=true for every resource": {
			namespace:     "testkube",
			rules:         nil,
			incomplete:    true,
			wantNamespace: "testkube",
			wantKinds:     []string{"Pod", "Rollout"},
			wantCanWatch:  map[string]bool{"Pod": true, "Rollout": true},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			d, capturedNs := newDiscovererWithFake(t, lists, tc.namespace, tc.rules, tc.incomplete)
			out, err := d.List(context.Background())
			require.NoError(t, err)
			assert.Equal(t, tc.wantNamespace, *capturedNs, "Discoverer must pass its namespace into SelfSubjectRulesReview")

			if tc.wantKinds != nil {
				gotKinds := make([]string, 0, len(out))
				for _, r := range out {
					gotKinds = append(gotKinds, r.Kind)
				}
				assert.ElementsMatch(t, tc.wantKinds, gotKinds, "post-filter Kind set mismatch (subresources + non-watchable resources should be dropped)")
			}
			if tc.wantCanWatch != nil {
				byKind := map[string]bool{}
				for _, r := range out {
					byKind[r.Kind] = r.CanWatch
				}
				for kind, want := range tc.wantCanWatch {
					assert.Equal(t, want, byKind[kind], "CanWatch for %s", kind)
				}
			}
		})
	}
}
