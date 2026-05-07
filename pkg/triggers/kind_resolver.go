package triggers

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/restmapper"
)

// builtinType holds the GVK and plural resource name for a Kubernetes type
// that the trigger service already watches via a typed informer.
type builtinType struct {
	Kind     string
	Resource string
	Group    string
	Version  string
}

// builtinTypes is the single source of truth for the 8 kinds with first-party
// static informers. Keys are lowercased for case-insensitive lookup by kind,
// resource, or the v1 Resource enum (which also uses lowercase values).
//
// Triggers that reference one of these kinds via resourceRef are served by
// the typed informer, not the dynamic informer, so events fire exactly once.
var builtinTypes = map[string]builtinType{
	"pod":         {Kind: "Pod", Resource: "pods", Group: "", Version: "v1"},
	"deployment":  {Kind: "Deployment", Resource: "deployments", Group: "apps", Version: "v1"},
	"statefulset": {Kind: "StatefulSet", Resource: "statefulsets", Group: "apps", Version: "v1"},
	"daemonset":   {Kind: "DaemonSet", Resource: "daemonsets", Group: "apps", Version: "v1"},
	"service":     {Kind: "Service", Resource: "services", Group: "", Version: "v1"},
	"ingress":     {Kind: "Ingress", Resource: "ingresses", Group: "networking.k8s.io", Version: "v1"},
	"event":       {Kind: "Event", Resource: "events", Group: "", Version: "v1"},
	"configmap":   {Kind: "ConfigMap", Resource: "configmaps", Group: "", Version: "v1"},
}

// isBuiltinResource reports whether the given kind has a first-party typed informer.
// Case-insensitive.
func isBuiltinResource(kind string) bool {
	_, ok := builtinTypes[strings.ToLower(kind)]
	return ok
}

// newCachedRESTMapper creates a REST mapper backed by an in-memory discovery cache.
// Reuse across multiple resolveGVR calls to avoid redundant discovery API requests.
func newCachedRESTMapper(discoveryClient discovery.DiscoveryInterface) meta.RESTMapper {
	cached := memory.NewMemCacheClient(discoveryClient)
	return restmapper.NewDeferredDiscoveryRESTMapper(cached)
}

// resolveGVR resolves a Group/Version/Kind to a GroupVersionResource.
// For the 8 built-in types, uses a hardcoded map (no API call).
// For custom resources, uses the provided REST mapper, transparently invalidating
// the discovery cache and retrying once if the kind is unknown — otherwise a CRD
// installed after the agent started would never be discoverable.
func resolveGVR(mapper meta.RESTMapper, group, version, kind string) (schema.GroupVersionResource, error) {
	if b, ok := builtinTypes[strings.ToLower(kind)]; ok {
		if group == "" || group == b.Group {
			return schema.GroupVersionResource{Group: b.Group, Version: b.Version, Resource: b.Resource}, nil
		}
	}

	gk := schema.GroupKind{Group: group, Kind: kind}
	var versions []string
	if version != "" {
		versions = append(versions, version)
	}

	mapping, err := mapper.RESTMapping(gk, versions...)
	if err != nil {
		// DeferredDiscoveryRESTMapper only auto-resets when its cache is stale.
		// Once populated, the cache is considered fresh forever, so a CRD
		// installed post-startup stays invisible. Force an invalidation, retry once.
		if resettable, ok := mapper.(meta.ResettableRESTMapper); ok {
			resettable.Reset()
			mapping, err = mapper.RESTMapping(gk, versions...)
		}
		if err != nil {
			return schema.GroupVersionResource{}, fmt.Errorf("resolve GVR for %s/%s/%s: %w", group, version, kind, err)
		}
	}

	return mapping.Resource, nil
}
