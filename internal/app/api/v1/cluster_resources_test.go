package v1

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authzv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/clusterdiscovery"
	"github.com/kubeshop/testkube/pkg/log"
)

// preferredFakeDiscovery overrides ServerPreferredResources, which the
// upstream client-go fake stubs to (nil, nil) - and which is the only
// discovery method Discoverer consumes.
type preferredFakeDiscovery struct {
	*fakediscovery.FakeDiscovery
	resources    []*metav1.APIResourceList
	discoveryErr error
}

func (p *preferredFakeDiscovery) ServerPreferredResources() ([]*metav1.APIResourceList, error) {
	return p.resources, p.discoveryErr
}

type discovererOpts struct {
	resources    []*metav1.APIResourceList
	rules        []authzv1.ResourceRule
	discoveryErr error
}

func newFakeDiscoverer(t *testing.T, opts discovererOpts) *clusterdiscovery.Discoverer {
	t.Helper()
	cs := fake.NewSimpleClientset()
	cs.PrependReactor("create", "selfsubjectrulesreviews", func(ktesting.Action) (bool, runtime.Object, error) {
		return true, &authzv1.SelfSubjectRulesReview{
			Status: authzv1.SubjectRulesReviewStatus{ResourceRules: opts.rules},
		}, nil
	})
	disc := &preferredFakeDiscovery{
		FakeDiscovery: cs.Discovery().(*fakediscovery.FakeDiscovery),
		resources:     opts.resources,
		discoveryErr:  opts.discoveryErr,
	}
	return clusterdiscovery.NewFromInterfaces(discovery.DiscoveryInterface(disc), cs, "testkube")
}

func TestListClusterResourcesHandler(t *testing.T) {
	// Fixture: one core watchable resource, one CRD-style watchable resource,
	// plus a subresource that must be dropped by Discoverer.
	resources := []*metav1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{Name: "pods", Kind: "Pod", Namespaced: true, Verbs: []string{"get", "list", "watch"}},
				{Name: "pods/log", Kind: "Pod", Namespaced: true, Verbs: []string{"get"}},
			},
		},
		{
			GroupVersion: "argoproj.io/v1alpha1",
			APIResources: []metav1.APIResource{
				{Name: "rollouts", Kind: "Rollout", Namespaced: true, Verbs: []string{"get", "list", "watch"}},
			},
		},
	}
	podsOnly := []authzv1.ResourceRule{
		{Verbs: []string{"list", "watch"}, APIGroups: []string{""}, Resources: []string{"pods"}},
	}

	// A nil discoverer simulates "ClusterDiscoverer not configured" and must
	// short-circuit to 501 before List is ever called.
	tests := map[string]struct {
		discoverer func(t *testing.T) *clusterdiscovery.Discoverer
		query      string
		wantStatus int
		wantKinds  []string
	}{
		"501 when ClusterDiscoverer is not configured": {
			discoverer: func(*testing.T) *clusterdiscovery.Discoverer { return nil },
			wantStatus: http.StatusNotImplemented,
		},
		"502 when discovery fails": {
			discoverer: func(t *testing.T) *clusterdiscovery.Discoverer {
				return newFakeDiscoverer(t, discovererOpts{discoveryErr: errors.New("apiserver unreachable")})
			},
			wantStatus: http.StatusBadGateway,
		},
		"200 with full GVK list (no filter)": {
			discoverer: func(t *testing.T) *clusterdiscovery.Discoverer {
				return newFakeDiscoverer(t, discovererOpts{resources: resources, rules: podsOnly})
			},
			wantStatus: http.StatusOK,
			wantKinds:  []string{"Pod", "Rollout"},
		},
		"200 with ?watchable=true drops GVKs the agent cannot watch": {
			discoverer: func(t *testing.T) *clusterdiscovery.Discoverer {
				return newFakeDiscoverer(t, discovererOpts{resources: resources, rules: podsOnly})
			},
			query:      "?watchable=true",
			wantStatus: http.StatusOK,
			wantKinds:  []string{"Pod"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			testAPI := &TestkubeAPI{
				ClusterDiscoverer: tc.discoverer(t),
				Log:               log.DefaultLogger,
			}
			app := fiber.New()
			app.Get("/cluster-resources", testAPI.ListClusterResourcesHandler())

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/cluster-resources"+tc.query, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tc.wantStatus, resp.StatusCode)

			if tc.wantStatus != http.StatusOK {
				return
			}
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			var got []testkube.ClusterResource
			require.NoError(t, json.Unmarshal(body, &got))
			gotKinds := make([]string, 0, len(got))
			for _, r := range got {
				gotKinds = append(gotKinds, r.Kind)
			}
			assert.ElementsMatch(t, tc.wantKinds, gotKinds)
		})
	}
}
