package controller

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"go.uber.org/zap"
	authorizationv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	aiv1 "github.com/kubeshop/testkube/api/ai/v1alpha1"
	insightsv1 "github.com/kubeshop/testkube/api/insights/v1alpha1"
	testbundlesv1 "github.com/kubeshop/testkube/api/testbundles/v1alpha1"
	"github.com/kubeshop/testkube/pkg/capabilities"
	"github.com/kubeshop/testkube/pkg/controlplane/declarative"
)

// ControlPlaneResourceStore receives the Control Plane resources this agent syncs.
type ControlPlaneResourceStore interface {
	UpdateOrCreateControlPlaneResource(ctx context.Context, obj declarative.Object) error
	DeleteControlPlaneResource(ctx context.Context, kind declarative.Kind, name string) error
}

// syncedControlPlaneResource is a kind of the declarative registry that is synced, by resource,
// with the capability a Control Plane advertises when it takes that kind through the SyncService.
type syncedControlPlaneResource struct {
	resource   string
	capability capabilities.Capability
}

// syncedControlPlaneResources are the kinds a GitOps agent syncs. A kind is added here when the
// Control Plane can take it through the SyncService. A kind that refers to another comes after it
// (an AI Agent after the MCP servers it links, a Test Bundle after its AI Agents), so on a start
// the dependency is usually there first; when it is not, the Control Plane answers with a
// retryable error and the reference is resolved on a later attempt.
var syncedControlPlaneResources = []syncedControlPlaneResource{
	{resource: insightsv1.BoardResource, capability: capabilities.CapabilityControlPlaneResources},
	{resource: aiv1.MCPServerResource, capability: capabilities.CapabilityControlPlaneResourcesAI},
	{resource: aiv1.LLMProviderResource, capability: capabilities.CapabilityControlPlaneResourcesAI},
	{resource: aiv1.AIAgentResource, capability: capabilities.CapabilityControlPlaneResourcesAI},
	{resource: testbundlesv1.TestBundleResource, capability: capabilities.CapabilityControlPlaneResourcesTestBundles},
}

// SyncedControlPlaneKinds returns the kinds of the declarative registry that a GitOps agent syncs
// to a Control Plane advertising the capabilities given, in the order of
// syncedControlPlaneResources.
func SyncedControlPlaneKinds(advertised []capabilities.Capability) []declarative.Kind {
	var out []declarative.Kind
	for _, synced := range syncedControlPlaneResources {
		if !slices.Contains(advertised, synced.capability) {
			continue
		}
		for _, k := range declarative.Kinds() {
			if k.Resource == synced.resource {
				out = append(out, k)
			}
		}
	}
	return out
}

// NewControlPlaneResourceSyncControllers registers a sync controller for each synced kind that the
// Control Plane takes (it advertises the kind's capability) and whose custom resource the agent
// can watch in namespace. A kind whose CRD is not installed, that the
// agent's role does not let it list and watch, or whose access the agent may not check (a cluster
// that forbids SelfSubjectAccessReviews), is skipped with a warning rather than failing: these
// kinds are optional, and a controller that cannot sync its cache stops the manager, and with it
// the agent.
func NewControlPlaneResourceSyncControllers(ctx context.Context, mgr ctrl.Manager, store ControlPlaneResourceStore, namespace string, advertised []capabilities.Capability, logger *zap.SugaredLogger) error {
	clientset, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return fmt.Errorf("create client for access reviews: %w", err)
	}
	reviews := clientset.AuthorizationV1().SelfSubjectAccessReviews()
	for _, kind := range SyncedControlPlaneKinds(advertised) {
		mapping, err := mgr.GetRESTMapper().RESTMapping(kind.GroupVersionKind().GroupKind(), kind.GroupVersion.Version)
		if meta.IsNoMatchError(err) {
			logger.Warnw("not syncing a Control Plane resource kind: its CRD is not installed", "kind", kind.Kind, "group", kind.GroupVersion.Group)
			continue
		}
		if err != nil {
			return fmt.Errorf("look up the %s CRD: %w", kind.Kind, err)
		}
		allowed, err := canListAndWatch(ctx, reviews, namespace, mapping.Resource.Group, mapping.Resource.Resource)
		switch {
		case errors.Is(err, errAccessReviewForbidden):
			logger.Warnw("not syncing a Control Plane resource kind: the agent may not check its access to it", "kind", kind.Kind, "group", kind.GroupVersion.Group, "namespace", namespace, "error", err)
			continue
		case err != nil:
			return fmt.Errorf("check access to %s: %w", kind.Resource, err)
		case !allowed:
			logger.Warnw("not syncing a Control Plane resource kind: the agent may not list and watch it", "kind", kind.Kind, "group", kind.GroupVersion.Group, "namespace", namespace)
			continue
		}
		if err := NewControlPlaneResourceSyncController(mgr, store, kind); err != nil {
			return err
		}
	}
	return nil
}

// accessReviews creates SelfSubjectAccessReviews, as the client-go client of the authorization API
// does.
type accessReviews interface {
	Create(ctx context.Context, review *authorizationv1.SelfSubjectAccessReview, opts metav1.CreateOptions) (*authorizationv1.SelfSubjectAccessReview, error)
}

// errAccessReviewForbidden is returned by canListAndWatch when the cluster refuses the agent's
// access review itself, so whether it may read the resource is unknown.
var errAccessReviewForbidden = errors.New("the agent may not create SelfSubjectAccessReviews")

// canListAndWatch reports whether the agent may list and watch the resource in namespace. A
// cluster that forbids the review itself is errAccessReviewForbidden.
func canListAndWatch(ctx context.Context, reviews accessReviews, namespace, group, resource string) (bool, error) {
	for _, verb := range []string{"list", "watch"} {
		review := &authorizationv1.SelfSubjectAccessReview{
			Spec: authorizationv1.SelfSubjectAccessReviewSpec{
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Namespace: namespace,
					Verb:      verb,
					Group:     group,
					Resource:  resource,
				},
			},
		}
		result, err := reviews.Create(ctx, review, metav1.CreateOptions{})
		if apierrors.IsForbidden(err) {
			return false, fmt.Errorf("%w: %w", errAccessReviewForbidden, err)
		}
		if err != nil {
			return false, err
		}
		if !result.Status.Allowed {
			return false, nil
		}
	}
	return true, nil
}

// NewControlPlaneResourceSyncController syncs the custom resources of one kind of the declarative
// registry to the Control Plane.
func NewControlPlaneResourceSyncController(mgr ctrl.Manager, store ControlPlaneResourceStore, kind declarative.Kind) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(kind.New()).
		Complete(controlPlaneResourceSyncReconciler(mgr.GetClient(), store, kind)); err != nil {
		return fmt.Errorf("create new sync controller for %s: %w", kind.Kind, err)
	}
	return nil
}

func controlPlaneResourceSyncReconciler(client client.Reader, store ControlPlaneResourceStore, kind declarative.Kind) reconcile.Reconciler {
	return reconcile.Func(func(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
		obj := kind.New()
		err := client.Get(ctx, req.NamespacedName, obj)
		switch {
		case apierrors.IsNotFound(err):
			// Deleted. Resources are named by their name alone in the Control Plane.
			return ctrl.Result{}, deleteControlPlaneResource(ctx, store, kind, req.Name)
		case err != nil:
			return ctrl.Result{}, fmt.Errorf("retrieve %s %q from Kubernetes: %w", kind.Kind, req.NamespacedName, err)
		}

		// Marked for deletion: there may be no event when it finally goes, so this is when the
		// Control Plane is told.
		if !obj.GetDeletionTimestamp().IsZero() {
			return ctrl.Result{}, deleteControlPlaneResource(ctx, store, kind, req.Name)
		}

		// Reading it back may leave the type unset, which the shared rules read.
		obj.GetObjectKind().SetGroupVersionKind(kind.GroupVersionKind())
		// Checked here with the rules the Control Plane applies, so an invalid resource is reported
		// where it is declared and never sent. No retry fixes it; a change in Git does.
		if err := declarative.Normalize(obj); err != nil {
			return ctrl.Result{}, reconcile.TerminalError(fmt.Errorf("%s %q is invalid, so it is not synced: %w", kind.Kind, obj.GetName(), err))
		}
		if err := store.UpdateOrCreateControlPlaneResource(ctx, obj); err != nil {
			return ctrl.Result{}, terminalOnRejection(fmt.Errorf("update %s %q in store: %w", kind.Kind, obj.GetName(), err))
		}
		return ctrl.Result{}, nil
	})
}

func deleteControlPlaneResource(ctx context.Context, store ControlPlaneResourceStore, kind declarative.Kind, name string) error {
	if err := store.DeleteControlPlaneResource(ctx, kind, name); err != nil {
		return terminalOnRejection(fmt.Errorf("delete %s %q from store: %w", kind.Kind, name, err))
	}
	return nil
}
