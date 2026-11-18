package controller

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"sync"

	"go.uber.org/zap"
	authorizationv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kubeshop/testkube/pkg/controlplane/declarative"
)

// ControlPlaneResourceStore receives the Control Plane resources this runner syncs.
type ControlPlaneResourceStore interface {
	UpdateOrCreateControlPlaneResource(ctx context.Context, obj declarative.Object) error
	DeleteControlPlaneResource(ctx context.Context, kind declarative.Kind, name string) error
}

// syncedControlPlaneResources names, by resource, the kinds of the declarative registry that are
// synced. A kind is added here when the Control Plane can take it through the SyncService.
var syncedControlPlaneResources = []string{"boards"}

// SyncedControlPlaneKinds returns the kinds of the declarative registry that a GitOps runner syncs.
func SyncedControlPlaneKinds() []declarative.Kind {
	var out []declarative.Kind
	for _, k := range declarative.Kinds() {
		if slices.Contains(syncedControlPlaneResources, k.Resource) {
			out = append(out, k)
		}
	}
	return out
}

// NewControlPlaneResourceSyncControllers registers a sync controller for each synced kind whose
// custom resource the runner can watch in namespace. A kind whose CRD is not installed, that the
// runner's role does not let it list and watch, or whose access the runner may not check (a cluster
// that forbids SelfSubjectAccessReviews), is skipped with a warning rather than failing: these
// kinds are optional, and a controller that cannot sync its cache stops the manager, and with it
// the runner.
func NewControlPlaneResourceSyncControllers(ctx context.Context, mgr ctrl.Manager, store ControlPlaneResourceStore, namespace string, logger *zap.SugaredLogger) error {
	clientset, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return fmt.Errorf("create client for access reviews: %w", err)
	}
	reviews := clientset.AuthorizationV1().SelfSubjectAccessReviews()
	for _, kind := range SyncedControlPlaneKinds() {
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
			logger.Warnw("not syncing a Control Plane resource kind: the runner may not check its access to it", "kind", kind.Kind, "group", kind.GroupVersion.Group, "namespace", namespace, "error", err)
			continue
		case err != nil:
			return fmt.Errorf("check access to %s: %w", kind.Resource, err)
		case !allowed:
			logger.Warnw("not syncing a Control Plane resource kind: the runner may not list and watch it", "kind", kind.Kind, "group", kind.GroupVersion.Group, "namespace", namespace)
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

// errAccessReviewForbidden is returned by canListAndWatch when the cluster refuses the runner's
// access review itself, so whether it may read the resource is unknown.
var errAccessReviewForbidden = errors.New("the runner may not create SelfSubjectAccessReviews")

// canListAndWatch reports whether the runner may list and watch the resource in namespace. A
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

// noGitOpsSyncAnnotation pauses the sync of one resource. The Control Plane skips the creates and
// updates of a resource carrying it; the runner skips its deletes (pausedResources), as the
// Control Plane does not store the annotations of these kinds and so cannot.
const noGitOpsSyncAnnotation = "testkube.io/no-gitops-sync"

// syncPaused reports whether obj carries testkube.io/no-gitops-sync set to true, read as the
// Control Plane reads it.
func syncPaused(obj client.Object) bool {
	value, ok := obj.GetAnnotations()[noGitOpsSyncAnnotation]
	if !ok {
		return false
	}
	paused, err := strconv.ParseBool(value)
	return err == nil && paused
}

// pausedResources remembers which custom resources carried testkube.io/no-gitops-sync when last
// seen. A deleted resource is gone by the time it is reconciled, and the delete names it alone, so
// what the runner saw of it last is what says whether its delete is paused: every event carries the
// object, the delete event its final state.
type pausedResources struct {
	mu    sync.Mutex
	names map[types.NamespacedName]bool
}

func newPausedResources() *pausedResources {
	return &pausedResources{names: map[types.NamespacedName]bool{}}
}

// observe records whether obj, as an event carries it, pauses its sync.
func (p *pausedResources) observe(obj client.Object) {
	if obj == nil {
		return
	}
	name := types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}
	p.mu.Lock()
	defer p.mu.Unlock()
	if syncPaused(obj) {
		p.names[name] = true
	} else {
		delete(p.names, name)
	}
}

// deleted reports whether the delete of the resource is paused, and forgets it: once gone, it is
// not seen again unless created anew.
func (p *pausedResources) deleted(name types.NamespacedName) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	paused := p.names[name]
	delete(p.names, name)
	return paused
}

// predicate observes every event and filters none.
func (p *pausedResources) predicate() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc:  func(e event.CreateEvent) bool { p.observe(e.Object); return true },
		UpdateFunc:  func(e event.UpdateEvent) bool { p.observe(e.ObjectNew); return true },
		DeleteFunc:  func(e event.DeleteEvent) bool { p.observe(e.Object); return true },
		GenericFunc: func(e event.GenericEvent) bool { p.observe(e.Object); return true },
	}
}

// NewControlPlaneResourceSyncController syncs the custom resources of one kind of the declarative
// registry to the Control Plane.
func NewControlPlaneResourceSyncController(mgr ctrl.Manager, store ControlPlaneResourceStore, kind declarative.Kind) error {
	paused := newPausedResources()
	if err := ctrl.NewControllerManagedBy(mgr).
		For(kind.New(), builder.WithPredicates(paused.predicate())).
		Complete(controlPlaneResourceSyncReconciler(mgr.GetClient(), store, kind, paused)); err != nil {
		return fmt.Errorf("create new sync controller for %s: %w", kind.Kind, err)
	}
	return nil
}

// controlPlaneResourceSyncReconciler syncs one kind. A resource carrying testkube.io/no-gitops-sync
// is sent, and the Control Plane skips it, but its delete is not sent at all: paused says whether
// the resource carried the annotation when the runner last saw it.
func controlPlaneResourceSyncReconciler(client client.Reader, store ControlPlaneResourceStore, kind declarative.Kind, paused *pausedResources) reconcile.Reconciler {
	return reconcile.Func(func(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
		obj := kind.New()
		err := client.Get(ctx, req.NamespacedName, obj)
		switch {
		case apierrors.IsNotFound(err):
			// Deleted. Resources are named by their name alone in the Control Plane.
			if paused.deleted(req.NamespacedName) {
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, deleteControlPlaneResource(ctx, store, kind, req.Name)
		case err != nil:
			return ctrl.Result{}, fmt.Errorf("retrieve %s %q from Kubernetes: %w", kind.Kind, req.NamespacedName, err)
		}

		// Marked for deletion: there may be no event when it finally goes, so this is when the
		// Control Plane is told, unless the resource pauses its sync.
		if !obj.GetDeletionTimestamp().IsZero() {
			if syncPaused(obj) {
				return ctrl.Result{}, nil
			}
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
