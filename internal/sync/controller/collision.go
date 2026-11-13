package controller

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"

	syncagent "github.com/kubeshop/testkube/internal/sync"
)

// checkNameCollision refuses obj when another watched namespace of the same Environment holds a
// resource of the same kind and name that wins over it. The Control Plane keys a resource by its
// name, so two of them would overwrite each other on every change.
//
// The resource with the earlier creationTimestamp wins, and on a tie the namespace that sorts
// first. The rule depends only on the objects, so every replica of the agent comes to the same
// answer. list is an empty list of the kind of obj; the cache of the manager holds only the
// watched namespaces, so the List sees nothing else.
func checkNameCollision(ctx context.Context, reader client.Reader, list client.ObjectList, obj client.Object, target syncagent.Target, resolveTarget syncagent.TargetFunc) error {
	kind := fmt.Sprintf("%T", list)
	if err := reader.List(ctx, list); err != nil {
		return fmt.Errorf("list %s: %w", kind, err)
	}
	items, err := meta.ExtractList(list)
	if err != nil {
		return fmt.Errorf("list %s: %w", kind, err)
	}

	for _, item := range items {
		other, ok := item.(client.Object)
		if !ok || other.GetName() != obj.GetName() || other.GetNamespace() == obj.GetNamespace() {
			continue
		}
		if resolveTarget(other.GetNamespace()).EnvironmentID != target.EnvironmentID {
			continue
		}
		if winsOver(other, obj) {
			return fmt.Errorf("%w: %q in namespace %q", syncagent.ErrNameCollision, obj.GetName(), other.GetNamespace())
		}
	}
	return nil
}

// syncDelete removes the resource named name in the namespace of target from the store. When
// another watched namespace of the same Environment still holds a resource of the same kind and
// name, the Control Plane record can belong to that resource. syncDelete then sends the winning one
// of them with upsert instead, so the delete of a refused resource keeps the winner. An ownership
// conflict on that upsert means that the record still belongs to the deleted resource, so
// syncDelete deletes it with del.
func syncDelete[T any, PT interface {
	*T
	client.Object
}](ctx context.Context, reader client.Reader, list client.ObjectList, name string, target syncagent.Target, resolveTarget syncagent.TargetFunc,
	upsert func(context.Context, syncagent.Target, T) error, del func(context.Context, syncagent.Target, string) error) error {
	winner, err := remainingWinner(ctx, reader, list, name, target, resolveTarget)
	if err != nil {
		return err
	}
	if obj, ok := winner.(PT); ok {
		err := upsert(ctx, resolveTarget(obj.GetNamespace()), *obj)
		if err == nil {
			return nil
		}
		if !errors.Is(err, syncagent.ErrOwnershipConflict) {
			return fmt.Errorf("update the resource of namespace %q: %w", obj.GetNamespace(), err)
		}
	}
	return del(ctx, target, name)
}

// remainingWinner returns the winning resource named name among the other watched namespaces of
// the Environment of target, or nil when there is none. It skips a resource that is marked for
// deletion.
func remainingWinner(ctx context.Context, reader client.Reader, list client.ObjectList, name string, target syncagent.Target, resolveTarget syncagent.TargetFunc) (client.Object, error) {
	kind := fmt.Sprintf("%T", list)
	if err := reader.List(ctx, list); err != nil {
		return nil, fmt.Errorf("list %s: %w", kind, err)
	}
	items, err := meta.ExtractList(list)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", kind, err)
	}

	var winner client.Object
	for _, item := range items {
		other, ok := item.(client.Object)
		if !ok || other.GetName() != name || other.GetNamespace() == target.Namespace || !other.GetDeletionTimestamp().IsZero() {
			continue
		}
		if resolveTarget(other.GetNamespace()).EnvironmentID != target.EnvironmentID {
			continue
		}
		if winner == nil || winsOver(other, winner) {
			winner = other
		}
	}
	return winner, nil
}

// winsOver reports whether a was created before b, or at the same time in a namespace that sorts
// before the namespace of b.
func winsOver(a, b client.Object) bool {
	at, bt := a.GetCreationTimestamp(), b.GetCreationTimestamp()
	if !at.Equal(&bt) {
		return at.Before(&bt)
	}
	return a.GetNamespace() < b.GetNamespace()
}
