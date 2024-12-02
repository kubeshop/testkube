package registry

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/singleflight"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/controller"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

const (
	DeregisterDelay = 1 * time.Second
)

type controllersRegistry struct {
	clientSet              kubernetes.Interface
	namespaces             NamespacesRegistry
	ips                    PodIpsRegistry
	controllers            map[string]controller.Controller
	controllerReservations map[string]int
	mu                     sync.RWMutex
	connectionsGroup       singleflight.Group
}

type ControllersRegistry interface {
	Get(id string) (ctrl controller.Controller, recycle func())
	Connect(ctx context.Context, id string, hints executionworkertypes.Hints) (ctrl controller.Controller, err error, recycle func())
	GetPodIP(ctx context.Context, id string) (string, error)
	GetNamespace(ctx context.Context, id string) (string, error)
	RegisterNamespace(id, namespace string)
	RegisterPodIP(id, podIp string)
}

func NewControllersRegistry(clientSet kubernetes.Interface, namespaces NamespacesRegistry, podIpCacheSize int) ControllersRegistry {
	r := &controllersRegistry{
		clientSet:              clientSet,
		namespaces:             namespaces,
		controllers:            make(map[string]controller.Controller),
		controllerReservations: make(map[string]int),
	}
	ipsRegistry := NewPodIpsRegistry(clientSet, r.GetNamespace, podIpCacheSize)
	r.ips = ipsRegistry
	return r
}

func (r *controllersRegistry) unsafeGet(id string) (ctrl controller.Controller, recycle func()) {
	// Search for active controller
	ctrl, ok := r.controllers[id]
	if !ok {
		return nil, func() {}
	}

	r.controllerReservations[id]++
	reserved := true
	return ctrl, func() {
		if !reserved {
			return
		}
		reserved = false
		r.deregister(id)
	}
}

func (r *controllersRegistry) deregister(id string) {
	r.mu.Lock()
	r.controllerReservations[id]--
	if r.controllerReservations[id] < 0 {
		r.controllerReservations[id] = 0
	}
	if r.controllerReservations[id] == 0 && r.controllers[id] != nil {
		podIp, err := r.controllers[id].PodIP()
		if err == nil && podIp != "" {
			r.ips.Register(id, podIp)
		}
		r.namespaces.Register(id, r.controllers[id].Namespace())

		// Wait a moment to deregister - common case is that something may want to read it immediately afterwards
		go func() {
			time.Sleep(DeregisterDelay)
			r.mu.Lock()
			defer r.mu.Unlock()
			if r.controllerReservations[id] == 0 && r.controllers[id] != nil {
				r.controllers[id].StopController()
				delete(r.controllers, id)
			}
		}()
	}
	r.mu.Unlock()
}

func (r *controllersRegistry) Get(id string) (ctrl controller.Controller, recycle func()) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.unsafeGet(id)
}

func (r *controllersRegistry) Connect(ctx context.Context, id string, hints executionworkertypes.Hints) (ctrl controller.Controller, err error, recycle func()) {
	for {
		// Either connect a new controller or use existing one
		obj, err, _ := r.connectionsGroup.Do(id, func() (interface{}, error) {
			r.mu.Lock()
			ctrl := r.controllers[id]
			r.mu.Unlock()

			if ctrl == nil {
				var signature []stage.Signature
				if len(hints.Signature) > 0 {
					signature = stage.MapSignatureList(hints.Signature)
				}
				namespace := hints.Namespace
				if namespace == "" {
					namespace, err = r.GetNamespace(ctx, id)
					if err != nil {
						return nil, err
					}
				}
				scheduledAt := common.ResolvePtr(hints.ScheduledAt, time.Time{}) // TODO: consider caching or making it optional
				nextCtrl, err := controller.New(ctx, r.clientSet, namespace, id, scheduledAt, controller.ControllerOptions{
					Signature: signature,
				})
				if err != nil {
					return nil, err
				}
				r.mu.Lock()
				r.controllers[id] = nextCtrl
				r.mu.Unlock()
				return nextCtrl, nil
			}

			// TODO: update scheduledAt / signature if provided and missing
			return ctrl, nil
		})

		// Try again, if context if initial caller has been called
		// TODO: Think how to better use context across multiple callers
		if errors.Is(err, context.Canceled) && ctx.Err() == nil {
			continue
		}

		if err != nil {
			return nil, err, func() {}
		}

		r.mu.Lock()
		r.controllerReservations[id]++
		r.mu.Unlock()

		reserved := true
		return obj.(controller.Controller), nil, func() {
			if !reserved {
				return
			}
			reserved = false
			r.deregister(id)
		}
	}
}

// TODO: Consider hinting with expected namespace
func (r *controllersRegistry) GetPodIP(ctx context.Context, id string) (string, error) {
	// Get the namespaces from existing controller
	r.mu.RLock()
	ctrl, ok := r.controllers[id]
	r.mu.RUnlock()
	if ok && ctrl.HasPod() {
		return ctrl.PodIP()
	}
	return r.ips.Get(ctx, id)
}

func (r *controllersRegistry) GetNamespace(ctx context.Context, id string) (string, error) {
	// Get the namespaces from existing controller
	r.mu.RLock()
	ctrl, ok := r.controllers[id]
	r.mu.RUnlock()
	if ok {
		return ctrl.Namespace(), nil
	}
	return r.namespaces.Get(ctx, id)
}

func (r *controllersRegistry) RegisterNamespace(id, namespace string) {
	r.namespaces.Register(id, namespace)
}

func (r *controllersRegistry) RegisterPodIP(id, podIp string) {
	r.ips.Register(id, podIp)
}
