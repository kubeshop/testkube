// Package clusterdiscovery enumerates every GVK the cluster exposes and tags
// each with whether the agent's ServiceAccount holds list+watch on it.
package clusterdiscovery

import (
	"context"
	"fmt"
	"sort"
	"strings"

	authzv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type Discoverer struct {
	discovery discovery.DiscoveryInterface
	authz     kubernetes.Interface
}

func New(c kubernetes.Interface) *Discoverer {
	return &Discoverer{discovery: c.Discovery(), authz: c}
}

func (d *Discoverer) List(ctx context.Context) ([]testkube.ClusterResource, error) {
	// Tolerate partial discovery — one broken aggregated API shouldn't 500 the endpoint.
	apiResourceLists, err := d.discovery.ServerPreferredResources()
	if err != nil && !discovery.IsGroupDiscoveryFailedError(err) {
		return nil, fmt.Errorf("server preferred resources: %w", err)
	}

	rules, err := d.effectiveWatchRules(ctx)
	if err != nil {
		return nil, err
	}

	var out []testkube.ClusterResource
	for _, list := range apiResourceLists {
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			continue
		}
		for _, r := range list.APIResources {
			if strings.Contains(r.Name, "/") || !containsOrWildcard(r.Verbs, "watch") {
				continue
			}
			out = append(out, testkube.ClusterResource{
				Group:      gv.Group,
				Version:    gv.Version,
				Kind:       r.Kind,
				Resource:   r.Name,
				Namespaced: r.Namespaced,
				CanWatch:   canWatch(rules, gv.Group, r.Name),
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Group != b.Group {
			return a.Group < b.Group
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.Version < b.Version
	})
	return out, nil
}

func (d *Discoverer) effectiveWatchRules(ctx context.Context) ([]authzv1.ResourceRule, error) {
	review := &authzv1.SelfSubjectRulesReview{
		Spec: authzv1.SelfSubjectRulesReviewSpec{Namespace: metav1.NamespaceDefault},
	}
	result, err := d.authz.AuthorizationV1().SelfSubjectRulesReviews().Create(ctx, review, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("self subject rules review: %w", err)
	}
	return result.Status.ResourceRules, nil
}

func canWatch(rules []authzv1.ResourceRule, group, resource string) bool {
	for _, r := range rules {
		if containsOrWildcard(r.Verbs, "list") && containsOrWildcard(r.Verbs, "watch") && containsOrWildcard(r.APIGroups, group) && containsOrWildcard(r.Resources, resource) {
			return true
		}
	}
	return false
}

// containsOrWildcard reports whether items contains want or the wildcard "*".
func containsOrWildcard(items []string, want string) bool {
	for _, v := range items {
		if v == want || v == "*" {
			return true
		}
	}
	return false
}
