package triggers

import (
	"context"
	"strings"
	"sync"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeshop/testkube/pkg/operator/validation/tests/v1/testtrigger"
)

// dynamicInformerEntry tracks a running dynamic informer and the set of
// triggers currently interested in its GVR. The informer stops only when
// the last trigger releases — tracking by trigger key (rather than a bare
// counter) makes add/release idempotent and prevents leaks when informer
// resyncs re-invoke AddFunc for the same TestTrigger object.
type dynamicInformerEntry struct {
	stopCh chan struct{}
	refs   map[string]struct{}
}

type dynamicInformerManager struct {
	client     dynamic.Interface
	mapper     meta.RESTMapper
	informers  map[string]*dynamicInformerEntry
	namespaces []string
	logger     *zap.SugaredLogger
	mu         sync.Mutex
}

func newDynamicInformerManager(client dynamic.Interface, mapper meta.RESTMapper, namespaces []string, logger *zap.SugaredLogger) *dynamicInformerManager {
	return &dynamicInformerManager{
		client:     client,
		mapper:     mapper,
		informers:  make(map[string]*dynamicInformerEntry),
		namespaces: namespaces,
		logger:     logger,
	}
}

// ensureInformer starts a dynamic informer for gvr if none is running, and
// records that triggerKey holds a reference. Safe to call repeatedly for the
// same (gvr, triggerKey) — only the first call registers the reference.
func (m *dynamicInformerManager) ensureInformer(ctx context.Context, gvr schema.GroupVersionResource, triggerKey string, handler cache.ResourceEventHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := gvr.String()
	if entry, ok := m.informers[key]; ok {
		if _, already := entry.refs[triggerKey]; !already {
			entry.refs[triggerKey] = struct{}{}
			m.logger.Debugf("trigger service: dynamic informer: trigger %q references %s (total %d)", triggerKey, key, len(entry.refs))
		}
		return
	}

	stopCh := make(chan struct{})
	m.informers[key] = &dynamicInformerEntry{
		stopCh: stopCh,
		refs:   map[string]struct{}{triggerKey: {}},
	}

	namespaces := m.namespaces
	if len(namespaces) == 0 {
		namespaces = []string{metav1.NamespaceAll}
	}

	for _, ns := range namespaces {
		factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(m.client, 0, ns, nil)
		informer := factory.ForResource(gvr).Informer()
		informer.AddEventHandler(handler)
		go informer.Run(stopCh)
	}

	m.logger.Infof("trigger service: dynamic informer: started watching %s (trigger %q)", key, triggerKey)
}

// releaseInformer drops triggerKey's reference to the informer for gvr,
// stopping the informer when the last reference is released.
// Idempotent: unknown (gvr, triggerKey) pairs are a no-op.
func (m *dynamicInformerManager) releaseInformer(gvr schema.GroupVersionResource, triggerKey string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := gvr.String()
	entry, ok := m.informers[key]
	if !ok {
		return
	}
	if _, held := entry.refs[triggerKey]; !held {
		return
	}
	delete(entry.refs, triggerKey)
	if len(entry.refs) == 0 {
		close(entry.stopCh)
		delete(m.informers, key)
		m.logger.Infof("trigger service: dynamic informer: stopped watching %s", key)
	}
}

func (m *dynamicInformerManager) stopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for key, entry := range m.informers {
		close(entry.stopCh)
		m.logger.Debugf("trigger service: dynamic informer: stopped %s on shutdown", key)
	}
	m.informers = make(map[string]*dynamicInformerEntry)
}

func (s *Service) dynamicEventHandler(ctx context.Context, gvr schema.GroupVersionResource) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			u, ok := obj.(*unstructured.Unstructured)
			if !ok {
				return
			}
			if inPast(u.GetCreationTimestamp().Time, s.watchFromDate) {
				return
			}
			resourceType := testtrigger.ResourceType(strings.ToLower(u.GetKind()))
			s.logger.Debugf("trigger service: dynamic informer: %s %s/%s created", resourceType, u.GetNamespace(), u.GetName())
			event := s.newWatcherEvent(testtrigger.EventCreated, u, u.Object, resourceType)
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("trigger service: dynamic informer: error matching create event: %v", err)
			}
		},
		UpdateFunc: func(oldObj, newObj any) {
			oldU, ok := oldObj.(*unstructured.Unstructured)
			if !ok {
				return
			}
			newU, ok := newObj.(*unstructured.Unstructured)
			if !ok {
				return
			}
			resourceType := testtrigger.ResourceType(strings.ToLower(newU.GetKind()))
			s.logger.Debugf("trigger service: dynamic informer: %s %s/%s updated", resourceType, newU.GetNamespace(), newU.GetName())
			event := s.newWatcherEvent(testtrigger.EventModified, newU, newU.Object, resourceType, withOldObject(oldU.Object))
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("trigger service: dynamic informer: error matching update event: %v", err)
			}
		},
		DeleteFunc: func(obj any) {
			u, ok := obj.(*unstructured.Unstructured)
			if !ok {
				// Handle tombstone events from watch resets
				tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					s.logger.Errorf("trigger service: dynamic informer: unexpected type %T for delete event", obj)
					return
				}
				u, ok = tombstone.Obj.(*unstructured.Unstructured)
				if !ok {
					s.logger.Errorf("trigger service: dynamic informer: unexpected tombstone object type %T", tombstone.Obj)
					return
				}
			}
			resourceType := testtrigger.ResourceType(strings.ToLower(u.GetKind()))
			s.logger.Debugf("trigger service: dynamic informer: %s %s/%s deleted", resourceType, u.GetNamespace(), u.GetName())
			event := s.newWatcherEvent(testtrigger.EventDeleted, u, u.Object, resourceType)
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("trigger service: dynamic informer: error matching delete event: %v", err)
			}
		},
	}
}
