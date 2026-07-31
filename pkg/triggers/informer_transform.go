package triggers

import (
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/cache"
)

// stripManagedFields drops a large share of every serialized object that nothing in the trigger
// service reads. Values without object metadata, such as tombstones, pass through rather than
// failing the delta, which would drop the event entirely.
func stripManagedFields(obj any) (any, error) {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return obj, nil
	}

	accessor.SetManagedFields(nil)
	return obj, nil
}

// watchInformer installs the transform before the informer starts, which client-go requires, and
// skips it when managed fields have to stay readable: cached objects are what v2 trigger
// expressions resolve `resource` against.
func watchInformer(logger *zap.SugaredLogger, name string, keepManagedFields bool, informer cache.SharedIndexInformer, handler cache.ResourceEventHandler) {
	if !keepManagedFields {
		if err := informer.SetTransform(stripManagedFields); err != nil {
			logger.Warnf("trigger service: failed to strip managed fields from %s informer cache: %v", name, err)
		}
	}

	informer.AddEventHandler(handler)
}
