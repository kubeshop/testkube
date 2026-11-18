package controller

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"sync"
	"time"

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

// deleteEventWait is how long a reconcile that reads a resource missing waits for its delete event
// before it looks again. The informer removes an object from its cache before it hands the event to
// the handlers, so the gap is short.
const deleteEventWait = time.Second

// pausedResources remembers, for every custom resource the runner has seen, what its last event
// said: whether it carried testkube.io/no-gitops-sync, and whether that event was its delete. A
// deleted resource is gone by the time it is reconciled, and the delete names it alone, so the delete
// event, which carries the object's final state, is what says whether its delete is paused.
//
// A resource read missing before its delete event is seen is not deleted yet: the informer removes
// it from its cache first, so a reconcile queued earlier can read it missing while its final state,
// such as an annotation added just before the delete, is still unseen (missingUnseen). Once the delete
// event is seen, a paused resource keeps its entry however often it is read missing, as a retry or a
// requeue reconciles the same name again; only an event changes it, such as the create of a new
// resource of the same name. A paused resource that stays deleted keeps its entry, one name each;
// the entry of one whose delete was sent is dropped.
type pausedResources struct {
	mu    sync.Mutex
	names map[types.NamespacedName]observedResource
}

// observedResource is what the last event of a resource said.
type observedResource struct {
	paused  bool
	deleted bool
}

func newPausedResources() *pausedResources {
	return &pausedResources{names: map[types.NamespacedName]observedResource{}}
}

// observe records obj as an event carries it, and whether the event is its delete.
func (p *pausedResources) observe(obj client.Object, deleted bool) {
	if obj == nil {
		return
	}
	name := types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.names[name] = observedResource{paused: syncPaused(obj), deleted: deleted}
}

// missingResource is what to do with a resource read missing.
type missingResource int

const (
	// missingUnseen: its delete event is not seen yet; look again later.
	missingUnseen missingResource = iota
	// missingPaused: deleted with testkube.io/no-gitops-sync; its delete is not sent.
	missingPaused
	// missingDeleted: deleted; its delete is sent.
	missingDeleted
	// missingGone: no entry, so its delete was sent already, or the runner never saw it.
	missingGone
)

// missing says what to do with the named resource, read missing.
func (p *pausedResources) missing(name types.NamespacedName) missingResource {
	p.mu.Lock()
	defer p.mu.Unlock()
	observed, ok := p.names[name]
	switch {
	case !ok:
		return missingGone
	case !observed.deleted:
		return missingUnseen
	case observed.paused:
		return missingPaused
	}
	return missingDeleted
}

// sent drops the entry of a resource whose delete was sent, unless an event changed it meanwhile,
// such as the create of a new resource of the same name.
func (p *pausedResources) sent(name types.NamespacedName) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if observed := p.names[name]; observed.deleted && !observed.paused {
		delete(p.names, name)
	}
}

// predicate observes every event and filters none.
func (p *pausedResources) predicate() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc:  func(e event.CreateEvent) bool { p.observe(e.Object, false); return true },
		UpdateFunc:  func(e event.UpdateEvent) bool { p.observe(e.ObjectNew, false); return true },
		DeleteFunc:  func(e event.DeleteEvent) bool { p.observe(e.Object, true); return true },
		GenericFunc: func(e event.GenericEvent) bool { p.observe(e.Object, false); return true },
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
			switch paused.missing(req.NamespacedName) {
			case missingUnseen:
				return ctrl.Result{RequeueAfter: deleteEventWait}, nil
			case missingPaused, missingGone:
				return ctrl.Result{}, nil
			}
			if err := deleteControlPlaneResource(ctx, store, kind, req.Name); err != nil {
				return ctrl.Result{}, err
			}
			paused.sent(req.NamespacedName)
			return ctrl.Result{}, nil
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
