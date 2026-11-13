package controller

import (
	"context"
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
func checkNameCollision(ctx context.Context, reader client.Reader, list client.ObjectList, obj client.Object, target syncagent.Target, resolve syncagent.TargetFunc) error {
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
		if resolve(other.GetNamespace()).EnvironmentID != target.EnvironmentID {
			continue
		}
		if winsOver(other, obj) {
			return fmt.Errorf("%w: %q in namespace %q", syncagent.ErrNameCollision, obj.GetName(), other.GetNamespace())
		}
	}
	return nil
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
