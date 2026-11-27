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
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	aiv1 "github.com/kubeshop/testkube/api/ai/v1alpha1"
	insightsv1 "github.com/kubeshop/testkube/api/insights/v1alpha1"
	"github.com/kubeshop/testkube/pkg/capabilities"
	"github.com/kubeshop/testkube/pkg/controlplane/declarative"
)

// ControlPlaneResourceStore receives the Control Plane resources this runner syncs.
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

// syncedControlPlaneResources are the kinds a GitOps runner syncs. A kind is added here when the
// Control Plane can take it through the SyncService. A kind that refers to another comes after it
// (an AI Agent after the MCP servers it links and the Skills it refers to), so on a start
// the dependency is usually there first; when it is not, the Control Plane answers with a
// retryable error and the reference is resolved on a later attempt.
var syncedControlPlaneResources = []syncedControlPlaneResource{
	{resource: insightsv1.BoardResource, capability: capabilities.CapabilityControlPlaneResources},
	{resource: aiv1.MCPServerResource, capability: capabilities.CapabilityControlPlaneResourcesAI},
	{resource: aiv1.LLMProviderResource, capability: capabilities.CapabilityControlPlaneResourcesAI},
	{resource: aiv1.SkillResource, capability: capabilities.CapabilityControlPlaneResourcesSkills},
	{resource: aiv1.AIAgentResource, capability: capabilities.CapabilityControlPlaneResourcesAI},
}

// SyncedControlPlaneKinds returns the kinds of the declarative registry that a GitOps runner syncs
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

// ControlPlaneResourceCacheObjects returns an empty object of each kind that
// NewControlPlaneResourceSyncControllers may sync and whose CRD is installed, and a ConfigMap when one
// of them reads Skills from ConfigMaps. The manager caches these only in the runner's namespace,
// whatever GITOPS_NAMESPACES watches: their sync names a resource by kind and name alone, so the
// same name in two namespaces would overwrite or delete each other, and their access is checked in
// the runner's namespace only. A kind whose CRD is not installed is left out, because the manager
// resolves every object of its per-object cache settings when it starts.
func ControlPlaneResourceCacheObjects(mapper meta.RESTMapper, advertised []capabilities.Capability) ([]client.Object, error) {
	var objects []client.Object
	configMaps := false
	for _, kind := range SyncedControlPlaneKinds(advertised) {
		_, err := mapper.RESTMapping(kind.GroupVersionKind().GroupKind(), kind.GroupVersion.Version)
		if meta.IsNoMatchError(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("look up the %s CRD: %w", kind.Kind, err)
		}
		objects = append(objects, kind.New())
		configMaps = configMaps || readsSkillSources(kind)
	}
	if configMaps {
		objects = append(objects, &corev1.ConfigMap{})
	}
	return objects, nil
}

// NewControlPlaneResourceSyncControllers registers a sync controller for each synced kind that the
// Control Plane takes (it advertises the kind's capability) and whose custom resource the runner
// can watch in namespace. A kind whose CRD is not installed, that the
// runner's role does not let it list and watch, or whose access the runner may not check (a cluster
// that forbids SelfSubjectAccessReviews), is skipped with a warning rather than failing: these
// kinds are optional, and a controller that cannot sync its cache stops the manager, and with it
// the runner.
func NewControlPlaneResourceSyncControllers(ctx context.Context, mgr ctrl.Manager, store ControlPlaneResourceStore, namespace string, advertised []capabilities.Capability, synced *SyncedResources, logger *zap.SugaredLogger) error {
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
			logger.Warnw("not syncing a Control Plane resource kind: the runner may not check its access to it", "kind", kind.Kind, "group", kind.GroupVersion.Group, "namespace", namespace, "error", err)
			continue
		case err != nil:
			return fmt.Errorf("check access to %s: %w", kind.Resource, err)
		case !allowed:
			logger.Warnw("not syncing a Control Plane resource kind: the runner may not list and watch it", "kind", kind.Kind, "group", kind.GroupVersion.Group, "namespace", namespace)
			continue
		}
		watchConfigMaps := false
		if readsSkillSources(kind) {
			watchConfigMaps, err = canWatchSkillSources(ctx, mgr.GetScheme(), reviews, namespace, kind, logger)
			if err != nil {
				return err
			}
		}
		if err := NewControlPlaneResourceSyncController(mgr, store, kind, watchConfigMaps, synced); err != nil {
			return err
		}
	}
	return nil
}

// readsSkillSources reports whether resources of kind may read Skills from ConfigMaps
// (skillFrom.configMapKeyRef).
func readsSkillSources(kind declarative.Kind) bool {
	return kind.GroupVersion == aiv1.GroupVersion && (kind.Kind == aiv1.SkillKind || kind.Kind == aiv1.AIAgentKind)
}

// canWatchSkillSources reports whether the sync of kind may watch the ConfigMaps Skills are read
// from: the manager's scheme must know ConfigMaps, and the runner must be allowed to list and
// watch them in namespace. Otherwise it warns, and the kind is synced without the watch, so a
// changed SKILL.md is synced only when its resource changes.
func canWatchSkillSources(ctx context.Context, scheme *runtime.Scheme, reviews accessReviews, namespace string, kind declarative.Kind, logger *zap.SugaredLogger) (bool, error) {
	if !scheme.Recognizes(corev1.SchemeGroupVersion.WithKind("ConfigMap")) {
		logger.Warnw("not watching the ConfigMaps Skills are read from: the controller scheme does not register core/v1 ConfigMaps, so a changed SKILL.md is synced only when its resource changes", "kind", kind.Kind)
		return false, nil
	}
	allowed, err := canListAndWatch(ctx, reviews, namespace, "", "configmaps")
	if err != nil && !errors.Is(err, errAccessReviewForbidden) {
		return false, fmt.Errorf("check access to configmaps: %w", err)
	}
	if !allowed {
		logger.Warnw("not watching the ConfigMaps Skills are read from: the runner may not list and watch them, so a changed SKILL.md is synced only when its resource changes", "kind", kind.Kind, "namespace", namespace)
	}
	return allowed, nil
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
//
// A delete seen during a watch interruption (DeleteStateUnknown) carries the last object the runner
// happened to see, which may predate an annotation added before the delete. Nothing else holds the
// annotation, the object being gone and the Control Plane keeping none, so such a delete is not sent
// (missingUnknown): the resource stays in the Control Plane, to be deleted there if it should go.
type pausedResources struct {
	mu    sync.Mutex
	names map[types.NamespacedName]observedResource
}

// observedResource is what the last event of a resource said.
type observedResource struct {
	paused  bool
	deleted bool
	// unknown is set for a delete whose final state is unknown (DeleteStateUnknown).
	unknown bool
}

func newPausedResources() *pausedResources {
	return &pausedResources{names: map[types.NamespacedName]observedResource{}}
}

// observe records obj as an event carries it, whether the event is its delete, and whether that
// delete's final state is unknown.
func (p *pausedResources) observe(obj client.Object, deleted, unknown bool) {
	if obj == nil {
		return
	}
	name := types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.names[name] = observedResource{paused: syncPaused(obj), deleted: deleted, unknown: deleted && unknown}
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
	// missingUnknown: deleted while the runner's watch was interrupted, so whether it was paused
	// is unknown; its delete is not sent.
	missingUnknown
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
	case observed.unknown:
		return missingUnknown
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
		CreateFunc:  func(e event.CreateEvent) bool { p.observe(e.Object, false, false); return true },
		UpdateFunc:  func(e event.UpdateEvent) bool { p.observe(e.ObjectNew, false, false); return true },
		DeleteFunc:  func(e event.DeleteEvent) bool { p.observe(e.Object, true, e.DeleteStateUnknown); return true },
		GenericFunc: func(e event.GenericEvent) bool { p.observe(e.Object, false, false); return true },
	}
}

// SyncedResources counts the Control Plane resources a runner syncs, by resource (boards,
// aiagents, ...), for the telemetry heartbeat. A resource counts once the Control Plane took it,
// and stops counting once its custom resource is deleted, pauses its sync, turns invalid or is
// refused (such as one another runner owns). A failure that is retried leaves it as it was. Its
// zero value is not usable; a nil one counts nothing.
type SyncedResources struct {
	mu    sync.Mutex
	names map[string]map[types.NamespacedName]struct{}
}

// NewSyncedResources returns an empty count.
func NewSyncedResources() *SyncedResources {
	return &SyncedResources{names: map[string]map[types.NamespacedName]struct{}{}}
}

// set records whether the named resource of the kind is synced.
func (s *SyncedResources) set(resource string, name types.NamespacedName, synced bool) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !synced {
		delete(s.names[resource], name)
		return
	}
	if s.names[resource] == nil {
		s.names[resource] = map[types.NamespacedName]struct{}{}
	}
	s.names[resource][name] = struct{}{}
}

// Counts returns how many resources of each kind are synced, leaving out the kinds with none, or
// nil if there are none at all.
func (s *SyncedResources) Counts() map[string]int {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var out map[string]int
	for resource, names := range s.names {
		if len(names) == 0 {
			continue
		}
		if out == nil {
			out = map[string]int{}
		}
		out[resource] = len(names)
	}
	return out
}

// NewControlPlaneResourceSyncController syncs the custom resources of one kind of the declarative
// registry to the Control Plane, counting what it syncs in synced. The SKILL.md a resource reads with skillFrom is read straight from
// the API server, so the runner caches no ConfigMap; with watchConfigMaps, a change to one of those
// ConfigMaps (watched by metadata only) syncs the resources reading it again.
func NewControlPlaneResourceSyncController(mgr ctrl.Manager, store ControlPlaneResourceStore, kind declarative.Kind, watchConfigMaps bool, synced *SyncedResources) error {
	paused := newPausedResources()
	b := ctrl.NewControllerManagedBy(mgr).
		For(kind.New(), builder.WithPredicates(paused.predicate()))
	if watchConfigMaps {
		list, err := newList(mgr.GetScheme(), kind)
		if err != nil {
			return err
		}
		b = b.Watches(&corev1.ConfigMap{}, handler.EnqueueRequestsFromMapFunc(skillSourceReaders(mgr.GetClient(), list)), builder.OnlyMetadata)
	}
	if err := b.Complete(controlPlaneResourceSyncReconciler(mgr.GetClient(), mgr.GetAPIReader(), store, kind, paused, synced)); err != nil {
		return fmt.Errorf("create new sync controller for %s: %w", kind.Kind, err)
	}
	return nil
}

// newList returns an empty list of kind's resources, as the scheme registers it.
func newList(scheme *runtime.Scheme, kind declarative.Kind) (func() client.ObjectList, error) {
	gvk := kind.GroupVersion.WithKind(kind.Kind + "List")
	if _, err := scheme.New(gvk); err != nil {
		return nil, fmt.Errorf("list type of %s: %w", kind.Kind, err)
	}
	return func() client.ObjectList {
		obj, _ := scheme.New(gvk)
		return obj.(client.ObjectList)
	}, nil
}

// skillSourceReaders maps a ConfigMap to the resources, in its namespace, whose skillFrom reads it.
func skillSourceReaders(reader client.Reader, newList func() client.ObjectList) handler.MapFunc {
	return func(ctx context.Context, configMap client.Object) []reconcile.Request {
		list := newList()
		if err := reader.List(ctx, list, client.InNamespace(configMap.GetNamespace())); err != nil {
			log.FromContext(ctx).Error(err, "list the resources that may read Skills from a ConfigMap", "configMap", configMap.GetName())
			return nil
		}
		items, err := meta.ExtractList(list)
		if err != nil {
			log.FromContext(ctx).Error(err, "read the resources that may read Skills from a ConfigMap", "configMap", configMap.GetName())
			return nil
		}
		var requests []reconcile.Request
		for _, item := range items {
			obj, ok := item.(declarative.Object)
			if ok && slices.Contains(declarative.SkillSourceConfigMaps(obj), configMap.GetName()) {
				requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}})
			}
		}
		return requests
	}
}

// configMapKeyReader reads the SKILL.md a skillFrom selects from a ConfigMap in namespace.
func configMapKeyReader(ctx context.Context, reader client.Reader, namespace string) declarative.SkillSourceReader {
	return func(ref aiv1.ConfigMapKeyReference) (string, error) {
		var configMap corev1.ConfigMap
		if err := reader.Get(ctx, types.NamespacedName{Namespace: namespace, Name: ref.Name}, &configMap); err != nil {
			return "", err
		}
		if value, ok := configMap.Data[ref.Key]; ok {
			return value, nil
		}
		return "", fmt.Errorf("ConfigMap %q has no key %q", ref.Name, ref.Key)
	}
}

// controlPlaneResourceSyncReconciler syncs one kind. A resource carrying testkube.io/no-gitops-sync
// is sent, and the Control Plane skips it, but its delete is not sent at all: paused says whether
// the resource carried the annotation when the runner last saw it. configMaps reads the SKILL.md
// files that skillFrom selects. synced counts what it syncs.
func controlPlaneResourceSyncReconciler(client client.Reader, configMaps client.Reader, store ControlPlaneResourceStore, kind declarative.Kind, paused *pausedResources, synced *SyncedResources) reconcile.Reconciler {
	return reconcile.Func(func(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
		obj := kind.New()
		err := client.Get(ctx, req.NamespacedName, obj)
		switch {
		case apierrors.IsNotFound(err):
			// Deleted. Resources are named by their name alone in the Control Plane.
			synced.set(kind.Resource, req.NamespacedName, false)
			switch paused.missing(req.NamespacedName) {
			case missingUnseen:
				return ctrl.Result{RequeueAfter: deleteEventWait}, nil
			case missingPaused, missingGone:
				return ctrl.Result{}, nil
			case missingUnknown:
				log.FromContext(ctx).Info("not deleting a Control Plane resource: it was deleted while the runner's watch was interrupted, so whether it carried testkube.io/no-gitops-sync is unknown; delete it in the Control Plane if it should go",
					"kind", kind.Kind, "name", req.Name)
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
			synced.set(kind.Resource, req.NamespacedName, false)
			if syncPaused(obj) {
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, deleteControlPlaneResource(ctx, store, kind, req.Name)
		}

		// Reading it back may leave the type unset, which the shared rules read.
		obj.GetObjectKind().SetGroupVersionKind(kind.GroupVersionKind())
		// A Skill read from a SKILL.md is sent as written inline: the Control Plane cannot read the
		// ConfigMap. A missing ConfigMap or key is retried, as Git may bring it next; a SKILL.md
		// that does not fit is fixed only by a change, which the ConfigMap watch picks up.
		if err := declarative.ResolveSkillSources(obj, configMapKeyReader(ctx, configMaps, req.Namespace)); err != nil {
			synced.set(kind.Resource, req.NamespacedName, false)
			err = fmt.Errorf("%s %q reads a Skill it cannot use, so it is not synced: %w", kind.Kind, obj.GetName(), err)
			if errors.Is(err, declarative.ErrInvalidSkillSource) {
				return ctrl.Result{}, reconcile.TerminalError(err)
			}
			return ctrl.Result{}, err
		}
		// Checked here with the rules the Control Plane applies, so an invalid resource is reported
		// where it is declared and never sent. No retry fixes it; a change in Git does.
		if err := declarative.Normalize(obj); err != nil {
			synced.set(kind.Resource, req.NamespacedName, false)
			return ctrl.Result{}, reconcile.TerminalError(fmt.Errorf("%s %q is invalid, so it is not synced: %w", kind.Kind, obj.GetName(), err))
		}
		if err := store.UpdateOrCreateControlPlaneResource(ctx, obj); err != nil {
			err = terminalOnRejection(fmt.Errorf("update %s %q in store: %w", kind.Kind, obj.GetName(), err))
			if errors.Is(err, reconcile.TerminalError(nil)) {
				synced.set(kind.Resource, req.NamespacedName, false)
			}
			return ctrl.Result{}, err
		}
		// The Control Plane skips a paused resource, so it is not counted as synced.
		synced.set(kind.Resource, req.NamespacedName, !syncPaused(obj))
		return ctrl.Result{}, nil
	})
}

func deleteControlPlaneResource(ctx context.Context, store ControlPlaneResourceStore, kind declarative.Kind, name string) error {
	if err := store.DeleteControlPlaneResource(ctx, kind, name); err != nil {
		return terminalOnRejection(fmt.Errorf("delete %s %q from store: %w", kind.Kind, name, err))
	}
	return nil
}
