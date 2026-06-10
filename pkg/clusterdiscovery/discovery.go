// Package clusterdiscovery enumerates every GVK the cluster exposes and tags
// each with whether the agent's ServiceAccount holds list+watch on it. For
// CustomResourceDefinitions, also extracts the openAPIV3Schema so the Control
// Plane / UI can offer schema-aware autocomplete on TestTrigger.match[] paths.
package clusterdiscovery

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	authzv1 "k8s.io/api/authorization/v1"
	apiextclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type Discoverer struct {
	discovery discovery.DiscoveryInterface
	authz     kubernetes.Interface
	apiext    apiextclient.Interface // optional; nil disables CRD-schema fetching
	// namespace scopes the SelfSubjectRulesReview backing canWatch. The review
	// only enumerates rules effective in that namespace, so a RoleBinding in
	// the agent's own namespace is invisible unless we pass that namespace
	// (NamespaceDefault would miss it). Cluster-scoped resources are covered
	// by ClusterRoleBindings regardless of namespace.
	namespace string
}

func New(c kubernetes.Interface, namespace string) *Discoverer {
	return NewFromInterfaces(c.Discovery(), c, namespace)
}

// NewFromInterfaces is a composition seam for tests in adjacent packages that
// need to swap the discovery implementation - client-go's fake stubs
// ServerPreferredResources to (nil, nil). Production wiring uses New.
func NewFromInterfaces(disc discovery.DiscoveryInterface, authz kubernetes.Interface, namespace string) *Discoverer {
	if namespace == "" {
		namespace = metav1.NamespaceDefault
	}
	return &Discoverer{discovery: disc, authz: authz, namespace: namespace}
}

// WithSchemas enables CRD openAPIV3Schema population on List(). Pass an
// apiextensions clientset built from the same rest.Config as the kubernetes
// client. Without this, ClusterResource.Schema is always empty.
func (d *Discoverer) WithSchemas(c apiextclient.Interface) *Discoverer {
	d.apiext = c
	return d
}

func (d *Discoverer) List(ctx context.Context) ([]testkube.ClusterResource, error) {
	// Tolerate partial discovery - one broken aggregated API shouldn't 500 the endpoint.
	apiResourceLists, err := d.discovery.ServerPreferredResources()
	if err != nil && !discovery.IsGroupDiscoveryFailedError(err) {
		return nil, fmt.Errorf("server preferred resources: %w", err)
	}

	rules, incomplete, err := d.effectiveWatchRules(ctx)
	if err != nil {
		return nil, err
	}

	crdSchemas := d.crdSchemas(ctx)

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
			// Permissive degradation: when the authorizer can't enumerate
			// every rule (webhook authorizers, Node authorizer combos), the
			// partial rules would yield false-negative canWatch values. We'd
			// rather over-offer in the UI than tell users to add RBAC they
			// already have.
			can := canWatch(rules, gv.Group, r.Name)
			if incomplete {
				can = true
			}
			cr := testkube.ClusterResource{
				Group:      gv.Group,
				Version:    gv.Version,
				Kind:       r.Kind,
				Resource:   r.Name,
				Namespaced: r.Namespaced,
				CanWatch:   can,
			}
			if s, ok := crdSchemas[gvkKey(gv.Group, gv.Version, r.Kind)]; ok {
				cr.Schema = s
			}
			out = append(out, cr)
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

// Watchable returns the subset of resources the agent can watch (CanWatch).
// It filters in place, reusing the input slice's backing array, so callers
// must not keep using the slice they passed in afterwards.
func Watchable(resources []testkube.ClusterResource) []testkube.ClusterResource {
	watchable := resources[:0]
	for _, r := range resources {
		if r.CanWatch {
			watchable = append(watchable, r)
		}
	}
	return watchable
}

// effectiveWatchRules returns the agent's SelfSubjectRulesReview rules and a
// flag indicating whether the authorizer enumerated all of them (when false,
// canWatch will have false negatives - see List for the degradation policy).
func (d *Discoverer) effectiveWatchRules(ctx context.Context) ([]authzv1.ResourceRule, bool, error) {
	review := &authzv1.SelfSubjectRulesReview{
		Spec: authzv1.SelfSubjectRulesReviewSpec{Namespace: d.namespace},
	}
	result, err := d.authz.AuthorizationV1().SelfSubjectRulesReviews().Create(ctx, review, metav1.CreateOptions{})
	if err != nil {
		return nil, false, fmt.Errorf("self subject rules review: %w", err)
	}
	return result.Status.ResourceRules, result.Status.Incomplete, nil
}

func canWatch(rules []authzv1.ResourceRule, group, resource string) bool {
	for _, r := range rules {
		if containsOrWildcard(r.Verbs, "list") && containsOrWildcard(r.Verbs, "watch") && containsOrWildcard(r.APIGroups, group) && containsOrWildcard(r.Resources, resource) {
			return true
		}
	}
	return false
}

func containsOrWildcard(items []string, want string) bool {
	for _, v := range items {
		if v == want || v == "*" {
			return true
		}
	}
	return false
}

func gvkKey(group, version, kind string) string {
	return group + "/" + version + "/" + kind
}

// crdSchemas is best-effort: any failure (apiext client missing, RBAC denial,
// transient API error) returns an empty map so the rest of discovery still
// completes. Built-in K8s types are not covered here - their schemas live in
// /openapi/v3 on the kube-apiserver.
func (d *Discoverer) crdSchemas(ctx context.Context) map[string]json.RawMessage {
	if d.apiext == nil {
		return nil
	}
	list, err := d.apiext.ApiextensionsV1().CustomResourceDefinitions().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil
	}
	out := make(map[string]json.RawMessage)
	for i := range list.Items {
		crd := &list.Items[i]
		for _, ver := range crd.Spec.Versions {
			if ver.Schema == nil || ver.Schema.OpenAPIV3Schema == nil {
				continue
			}
			raw, err := json.Marshal(ver.Schema.OpenAPIV3Schema)
			if err != nil {
				continue
			}
			out[gvkKey(crd.Spec.Group, ver.Name, crd.Spec.Names.Kind)] = raw
		}
	}
	return out
}
