package declarative

import (
	"fmt"
	"slices"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	aiv1 "github.com/kubeshop/testkube/api/ai/v1alpha1"
	commonv1 "github.com/kubeshop/testkube/api/common/v1"
	insightsv1 "github.com/kubeshop/testkube/api/insights/v1alpha1"
	testbundlesv1 "github.com/kubeshop/testkube/api/testbundles/v1alpha1"
)

// Object is a Control Plane resource.
type Object interface {
	metav1.Object
	runtime.Object
}

// Scope is where a kind lives in the Control Plane.
type Scope string

const (
	// ScopeOrganization kinds are shared by every environment of an
	// organization, and are addressed without an environment.
	ScopeOrganization Scope = "organization"
	// ScopeEnvironment kinds belong to one environment.
	ScopeEnvironment Scope = "environment"
)

// Kind describes one kind of Control Plane resource. Each entity family has
// an API group of its own.
type Kind struct {
	// GroupVersion is the kind's API group and version.
	GroupVersion schema.GroupVersion
	// Kind is the object's kind, such as Board.
	Kind string
	// Resource is the plural name, used as the path segment of the generic
	// endpoints. It is unique across groups.
	Resource string
	// Names are what the CLI accepts for the kind: the singular name first,
	// then the plural and any short names.
	Names []string
	// Scope is where the kind lives in the Control Plane.
	Scope Scope

	newObject func() Object
}

// GroupVersionKind returns the kind's group, version and kind.
func (k Kind) GroupVersionKind() schema.GroupVersionKind {
	return k.GroupVersion.WithKind(k.Kind)
}

// New returns an empty object of the kind, with apiVersion and kind set.
func (k Kind) New() Object {
	obj := k.newObject()
	obj.GetObjectKind().SetGroupVersionKind(k.GroupVersionKind())
	return obj
}

// Name returns the kind's singular CLI name.
func (k Kind) Name() string {
	return k.Names[0]
}

// kinds is in apply order: a kind comes after every kind it refers to, so
// SortForApply can put an AI Agent after the MCP servers and LLM providers in
// the same input.
var kinds = []Kind{
	{
		GroupVersion: insightsv1.GroupVersion,
		Kind:         insightsv1.BoardKind,
		Resource:     insightsv1.BoardResource,
		Names:        []string{"board", "boards"},
		Scope:        ScopeOrganization,
		newObject:    func() Object { return &insightsv1.Board{} },
	},
	{
		GroupVersion: aiv1.GroupVersion,
		Kind:         aiv1.LLMProviderKind,
		Resource:     aiv1.LLMProviderResource,
		Names:        []string{"llmprovider", "llmproviders"},
		Scope:        ScopeEnvironment,
		newObject:    func() Object { return &aiv1.LLMProvider{} },
	},
	{
		GroupVersion: aiv1.GroupVersion,
		Kind:         aiv1.MCPServerKind,
		Resource:     aiv1.MCPServerResource,
		Names:        []string{"mcpserver", "mcpservers"},
		Scope:        ScopeEnvironment,
		newObject:    func() Object { return &aiv1.MCPServer{} },
	},
	{
		GroupVersion: aiv1.GroupVersion,
		Kind:         aiv1.AIAgentKind,
		Resource:     aiv1.AIAgentResource,
		Names:        []string{"aiagent", "aiagents"},
		Scope:        ScopeEnvironment,
		newObject:    func() Object { return &aiv1.AIAgent{} },
	},
	{
		GroupVersion: testbundlesv1.GroupVersion,
		Kind:         testbundlesv1.TestBundleKind,
		Resource:     testbundlesv1.TestBundleResource,
		Names:        []string{"testbundle", "testbundles"},
		Scope:        ScopeEnvironment,
		newObject:    func() Object { return &testbundlesv1.TestBundle{} },
	},
}

// SortForApply orders objects so each comes after the kinds it refers to,
// keeping the input order within a kind. Applying an AI Agent and its MCP
// server from one input then works whatever order the input has them in.
func SortForApply(objects []Object) {
	rank := func(obj Object) int {
		kind, err := KindOf(obj)
		if err != nil {
			return len(kinds)
		}
		return slices.IndexFunc(kinds, func(k Kind) bool { return k.GroupVersionKind() == kind.GroupVersionKind() })
	}
	slices.SortStableFunc(objects, func(a, b Object) int { return rank(a) - rank(b) })
}

// Kinds returns every supported kind.
func Kinds() []Kind {
	return slices.Clone(kinds)
}

// LookupKind returns the kind an object declares with apiVersion and kind.
func LookupKind(apiVersion, kind string) (Kind, error) {
	known := false
	for _, k := range kinds {
		if k.GroupVersion.String() != apiVersion {
			continue
		}
		known = true
		if k.Kind == kind {
			return k, nil
		}
	}
	if !known {
		return Kind{}, fmt.Errorf("apiVersion %q is not a Control Plane resource (supported: %s)", apiVersion, strings.Join(supported(), ", "))
	}
	return Kind{}, fmt.Errorf("unknown kind %q for %s (supported: %s)", kind, apiVersion, strings.Join(supported(), ", "))
}

// LookupName returns the kind a CLI name refers to, case-insensitively.
func LookupName(name string) (Kind, bool) {
	name = strings.ToLower(name)
	for _, k := range kinds {
		if slices.Contains(k.Names, name) {
			return k, true
		}
	}
	return Kind{}, false
}

// KindOf returns the kind of an object.
func KindOf(obj Object) (Kind, error) {
	for _, k := range kinds {
		if fmt.Sprintf("%T", k.newObject()) == fmt.Sprintf("%T", obj) {
			return k, nil
		}
	}
	return Kind{}, fmt.Errorf("unsupported object type %T", obj)
}

// supported lists every kind as apiVersion/kind.
func supported() []string {
	names := make([]string, 0, len(kinds))
	for _, k := range kinds {
		names = append(names, k.GroupVersion.String()+"/"+k.Kind)
	}
	return names
}

// StatusOf returns the Control Plane's view of an object.
func StatusOf(obj Object) commonv1.ControlPlaneResourceStatus {
	switch o := obj.(type) {
	case *insightsv1.Board:
		return o.Status
	case *aiv1.AIAgent:
		return o.Status
	case *aiv1.MCPServer:
		return o.Status.ControlPlaneResourceStatus
	case *aiv1.LLMProvider:
		return o.Status
	case *testbundlesv1.TestBundle:
		return o.Status.ControlPlaneResourceStatus
	}
	return commonv1.ControlPlaneResourceStatus{}
}
