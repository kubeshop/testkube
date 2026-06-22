package controller

import (
	"context"
	"time"

	"go.uber.org/zap"
	apiextclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apiextinformers "k8s.io/apiextensions-apiserver/pkg/client/informers/externalversions"
	"k8s.io/client-go/tools/cache"
)

// crdResyncPeriod gives the informer a slow safety-net resync - even if we
// miss a watch event (controller restart, etcd hiccup), we'll learn about the
// CRD set within this window. Inventory has a separate hourly safety-net push
// too, so this is mostly belt-and-suspenders.
const crdResyncPeriod = 30 * time.Minute

// StartCRDChangeNotifier launches a controller-runtime informer on
// CustomResourceDefinition resources and returns a channel that emits whenever
// a CRD is added, modified, or deleted. The returned channel never blocks the
// informer: it has a small buffer and silently drops on overflow because the
// downstream consumer (ClusterResourcesController) debounces signals into a
// single discovery+push anyway.
//
// The agent service-account needs `get/list/watch` on
// `customresourcedefinitions.apiextensions.k8s.io` for this to work; without
// it the informer's initial list fails and the notifier never emits.
func StartCRDChangeNotifier(ctx context.Context, client apiextclient.Interface, log *zap.SugaredLogger) <-chan struct{} {
	out := make(chan struct{}, 8)

	factory := apiextinformers.NewSharedInformerFactory(client, crdResyncPeriod)
	informer := factory.Apiextensions().V1().CustomResourceDefinitions().Informer()

	notify := func() {
		select {
		case out <- struct{}{}:
		default:
			// Buffer full: a debounced push is already queued downstream, so
			// dropping additional change events doesn't lose information.
		}
	}
	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(_ interface{}) { notify() },
		UpdateFunc: func(_, _ interface{}) { notify() },
		DeleteFunc: func(_ interface{}) { notify() },
	})
	if err != nil {
		log.Warnw("inventory: CRD informer event handler registration failed; CRD-change push triggers disabled", "error", err)
		return out
	}

	// WaitForCacheSync runs in a separate goroutine so we don't gate the parent
	// ctx on it: if RBAC denies list/watch, the agent should still run with
	// hourly-poll-only behavior.
	factory.Start(ctx.Done())
	go func() {
		if !cache.WaitForCacheSync(ctx.Done(), informer.HasSynced) {
			log.Warnw("inventory: CRD informer cache failed to sync; running without change-triggered pushes")
		}
	}()

	return out
}
