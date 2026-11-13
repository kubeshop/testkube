package controller

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authorizationv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	insightsv1 "github.com/kubeshop/testkube/api/insights/v1alpha1"
	gitopssync "github.com/kubeshop/testkube/internal/sync"
	"github.com/kubeshop/testkube/pkg/capabilities"
	"github.com/kubeshop/testkube/pkg/controlplane/declarative"
)

// boardClient returns a copy of Board, without its type, as a cache read may.
type boardClient struct {
	Board *insightsv1.Board
	Err   error
}

func (c boardClient) Get(_ context.Context, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	if c.Err != nil {
		return c.Err
	}
	b := c.Board.DeepCopy()
	b.TypeMeta = metav1.TypeMeta{}
	*obj.(*insightsv1.Board) = *b
	return nil
}

func (c boardClient) List(_ context.Context, _ client.ObjectList, _ ...client.ListOption) error {
	return nil
}

type fakeControlPlaneResourceStore struct {
	Sent        declarative.Object
	DeletedKind declarative.Kind
	Deleted     string
	Err         error
}

func (s *fakeControlPlaneResourceStore) UpdateOrCreateControlPlaneResource(_ context.Context, obj declarative.Object) error {
	s.Sent = obj
	return s.Err
}

func (s *fakeControlPlaneResourceStore) DeleteControlPlaneResource(_ context.Context, kind declarative.Kind, name string) error {
	s.DeletedKind, s.Deleted = kind, name
	return s.Err
}

func boardKind(t *testing.T) declarative.Kind {
	t.Helper()
	kinds := SyncedControlPlaneKinds([]capabilities.Capability{capabilities.CapabilityControlPlaneResources})
	require.Len(t, kinds, 1, "boards are the only kind the base capability syncs")
	require.Equal(t, "boards", kinds[0].Resource)
	return kinds[0]
}

func resources(kinds []declarative.Kind) []string {
	out := make([]string, 0, len(kinds))
	for _, k := range kinds {
		out = append(out, k.Resource)
	}
	return out
}

// A kind is synced only to a Control Plane that advertises the capability of its family, so an
// older one, or one that serves no AI resources, never gets a kind it refuses. A kind comes after
// the kinds it refers to.
func TestSyncedControlPlaneKinds(t *testing.T) {
	assert.Empty(t, SyncedControlPlaneKinds(nil))
	assert.Equal(t, []string{"boards"}, resources(SyncedControlPlaneKinds([]capabilities.Capability{
		capabilities.CapabilityControlPlaneResources,
	})))
	assert.Equal(t, []string{"mcpservers", "llmproviders", "aiagents"}, resources(SyncedControlPlaneKinds([]capabilities.Capability{
		capabilities.CapabilityControlPlaneResourcesAI,
	})))
	assert.Equal(t, []string{"boards", "mcpservers", "llmproviders", "aiagents", "testbundles"}, resources(SyncedControlPlaneKinds([]capabilities.Capability{
		capabilities.CapabilityControlPlaneResourcesTestBundles,
		capabilities.CapabilityControlPlaneResourcesAI,
		capabilities.CapabilityControlPlaneResources,
	})))
}

func request(name string) reconcile.Request {
	return reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "testkube", Name: name}}
}

// A board is normalized with the shared rules before it is sent, so the Control Plane gets what a
// testkube apply of the same manifest would send.
func TestControlPlaneResourceSyncReconcilerUpdateOrCreate(t *testing.T) {
	board := &insightsv1.Board{ObjectMeta: metav1.ObjectMeta{Name: "release-health", Namespace: "testkube"}}
	store := &fakeControlPlaneResourceStore{}

	_, err := controlPlaneResourceSyncReconciler(boardClient{Board: board}, store, boardKind(t)).Reconcile(t.Context(), request("release-health"))
	require.NoError(t, err)

	sent, ok := store.Sent.(*insightsv1.Board)
	require.True(t, ok)
	assert.Equal(t, "Board", sent.Kind, "the type is set again for the shared rules")
	assert.Equal(t, "release-health", sent.Spec.DisplayName, "normalized: the display name defaults to the name")
}

// An invalid board is never sent, and is not retried: only a change in Git fixes it.
func TestControlPlaneResourceSyncReconcilerInvalid(t *testing.T) {
	board := &insightsv1.Board{ObjectMeta: metav1.ObjectMeta{Name: "Not A Name"}}
	store := &fakeControlPlaneResourceStore{}

	_, err := controlPlaneResourceSyncReconciler(boardClient{Board: board}, store, boardKind(t)).Reconcile(t.Context(), request("Not A Name"))
	require.Error(t, err)
	assert.True(t, errors.Is(err, reconcile.TerminalError(nil)), "terminal: %v", err)
	assert.Nil(t, store.Sent)
}

func TestControlPlaneResourceSyncReconcilerDelete(t *testing.T) {
	store := &fakeControlPlaneResourceStore{}

	_, err := controlPlaneResourceSyncReconciler(boardClient{Err: fakeNotFoundErr}, store, boardKind(t)).Reconcile(t.Context(), request("release-health"))
	require.NoError(t, err)
	assert.Equal(t, "release-health", store.Deleted)
	assert.Equal(t, "boards", store.DeletedKind.Resource)
}

func TestControlPlaneResourceSyncReconcilerDeleteWhenMarkedForDeletion(t *testing.T) {
	now := metav1.Now()
	board := &insightsv1.Board{ObjectMeta: metav1.ObjectMeta{Name: "release-health", DeletionTimestamp: &now, Finalizers: []string{"keep"}}}
	store := &fakeControlPlaneResourceStore{}

	_, err := controlPlaneResourceSyncReconciler(boardClient{Board: board}, store, boardKind(t)).Reconcile(t.Context(), request("release-health"))
	require.NoError(t, err)
	assert.Equal(t, "release-health", store.Deleted)
	assert.Nil(t, store.Sent)
}

// An ownership conflict is terminal for updates and deletes alike; any other failure is retried.
func TestControlPlaneResourceSyncReconcilerRejection(t *testing.T) {
	board := &insightsv1.Board{ObjectMeta: metav1.ObjectMeta{Name: "release-health"}}
	conflict := &fakeControlPlaneResourceStore{Err: gitopssync.ErrOwnershipConflict}

	_, err := controlPlaneResourceSyncReconciler(boardClient{Board: board}, conflict, boardKind(t)).Reconcile(t.Context(), request("release-health"))
	assert.True(t, errors.Is(err, reconcile.TerminalError(nil)), "update: %v", err)
	_, err = controlPlaneResourceSyncReconciler(boardClient{Err: fakeNotFoundErr}, conflict, boardKind(t)).Reconcile(t.Context(), request("release-health"))
	assert.True(t, errors.Is(err, reconcile.TerminalError(nil)), "delete: %v", err)

	unavailable := &fakeControlPlaneResourceStore{Err: errors.New("unavailable")}
	_, err = controlPlaneResourceSyncReconciler(boardClient{Board: board}, unavailable, boardKind(t)).Reconcile(t.Context(), request("release-health"))
	require.Error(t, err)
	assert.False(t, errors.Is(err, reconcile.TerminalError(nil)), "retried: %v", err)
}

type fakeAccessReviews struct {
	denied string
	err    error
	asked  []authorizationv1.ResourceAttributes
}

func (f *fakeAccessReviews) Create(_ context.Context, review *authorizationv1.SelfSubjectAccessReview, _ metav1.CreateOptions) (*authorizationv1.SelfSubjectAccessReview, error) {
	f.asked = append(f.asked, *review.Spec.ResourceAttributes)
	if f.err != nil {
		return nil, f.err
	}
	out := review.DeepCopy()
	out.Status.Allowed = review.Spec.ResourceAttributes.Verb != f.denied
	return out, nil
}

// A kind is synced only when the runner may both list and watch it in its namespace.
func TestCanListAndWatch(t *testing.T) {
	reviews := &fakeAccessReviews{}
	allowed, err := canListAndWatch(t.Context(), reviews, "testkube", "insights.testkube.io", "boards")
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, []authorizationv1.ResourceAttributes{
		{Namespace: "testkube", Verb: "list", Group: "insights.testkube.io", Resource: "boards"},
		{Namespace: "testkube", Verb: "watch", Group: "insights.testkube.io", Resource: "boards"},
	}, reviews.asked)

	allowed, err = canListAndWatch(t.Context(), &fakeAccessReviews{denied: "watch"}, "testkube", "insights.testkube.io", "boards")
	require.NoError(t, err)
	assert.False(t, allowed)
}

// A cluster that forbids the access review itself makes the kind unavailable, which the caller
// skips, rather than an error that stops the runner; any other failure is still an error.
func TestCanListAndWatch_ReviewForbidden(t *testing.T) {
	forbidden := apierrors.NewForbidden(schema.GroupResource{Group: "authorization.k8s.io", Resource: "selfsubjectaccessreviews"}, "", errors.New("RBAC: access denied"))
	allowed, err := canListAndWatch(t.Context(), &fakeAccessReviews{err: forbidden}, "testkube", "insights.testkube.io", "boards")
	require.ErrorIs(t, err, errAccessReviewForbidden)
	assert.False(t, allowed)

	unavailable := apierrors.NewServiceUnavailable("apiserver is starting")
	_, err = canListAndWatch(t.Context(), &fakeAccessReviews{err: unavailable}, "testkube", "insights.testkube.io", "boards")
	require.Error(t, err)
	assert.NotErrorIs(t, err, errAccessReviewForbidden)
}
