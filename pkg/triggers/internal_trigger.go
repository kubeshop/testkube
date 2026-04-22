package triggers

import (
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	commonv1 "github.com/kubeshop/testkube/api/common/v1"
	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	workflowtriggersv1 "github.com/kubeshop/testkube/api/workflowtriggers/v1"
)

const (
	triggerSourceV1 = "v1"
	triggerSourceV2 = "v2"

	concurrencyPolicyForbid  = "forbid"
	concurrencyPolicyReplace = "replace"
)

// internalTrigger is the unified trigger representation used by the matcher and executor.
// Both v1 TestTrigger and v2 WorkflowTrigger convert to this type at ingestion.
type internalTrigger struct {
	Name      string
	Namespace string
	Labels    map[string]string
	Source    string

	// Watch
	ResourceGroup     string
	ResourceVersion   string
	ResourceKind      string
	ResourceName      string
	ResourceNamespace string
	Selector          *internalTriggerSelector

	// v1 legacy: event label selector (matches against auto-generated event labels)
	EventLabelSelector *metav1.LabelSelector

	// When
	Event string

	// Match
	FieldConditions []workflowtriggersv1.WorkflowTriggerFieldCondition

	// Wait
	Conditions *internalWaitConditions
	Probes     *internalWaitProbes

	// Run
	WorkflowSelector  internalTriggerSelector
	Target            *commonv1.Target
	Config            map[string]string
	Tags              map[string]string
	ConcurrencyPolicy string
	Delay             *time.Duration

	// Execution is the v1 TestTrigger.Spec.Execution value ("test", "testsuite",
	// or "testworkflow"). v2 has no equivalent and leaves this empty (implicitly
	// testworkflow). The matcher skips v1 triggers set to anything other than
	// testworkflow or empty — v1 non-testworkflow paths are no longer supported.
	Execution string

	Disabled bool
}

type internalTriggerSelector struct {
	Name           string
	NameRegex      string
	Namespace      string
	NamespaceRegex string
	LabelSelector  *metav1.LabelSelector
}

type internalWaitConditions struct {
	Items   []internalCondition
	Timeout int32
	Delay   int32
}

type internalCondition struct {
	Type   string
	Status *string
	Reason string
	TTL    int32
}

type internalWaitProbes struct {
	Items   []internalProbe
	Timeout int32
	Delay   int32
}

type internalProbe struct {
	Scheme  string
	Host    string
	Path    string
	Port    int32
	Headers map[string]string
}

// convertV1ToInternal converts a v1 TestTrigger CRD to the internal representation.
func convertV1ToInternal(t *testtriggersv1.TestTrigger) *internalTrigger {
	it := &internalTrigger{
		Name:               t.Name,
		Namespace:          t.Namespace,
		Labels:             t.Labels,
		Source:             triggerSourceV1,
		Event:              string(t.Spec.Event),
		EventLabelSelector: t.Spec.Selector,
		FieldConditions:    t.Spec.Match,
		Execution:          string(t.Spec.Execution),
		Disabled:           t.Spec.Disabled,
	}

	// Resolve v1 resource enum to GVK via the single builtinTypes source of truth.
	resourceStr := strings.ToLower(string(t.Spec.Resource))
	if b, ok := builtinTypes[resourceStr]; ok {
		it.ResourceGroup = b.Group
		it.ResourceVersion = b.Version
		it.ResourceKind = b.Kind
	} else {
		it.ResourceKind = string(t.Spec.Resource)
	}

	// ResourceRef overrides the Resource enum when set.
	if t.Spec.ResourceRef != nil {
		it.ResourceGroup = t.Spec.ResourceRef.Group
		it.ResourceVersion = t.Spec.ResourceRef.Version
		it.ResourceKind = t.Spec.ResourceRef.Kind
	}

	// Resource selector
	sel := t.Spec.ResourceSelector
	it.ResourceName = sel.Name
	it.ResourceNamespace = sel.Namespace
	if sel.NameRegex != "" || sel.NamespaceRegex != "" || sel.LabelSelector != nil {
		it.Selector = &internalTriggerSelector{
			NameRegex:      sel.NameRegex,
			NamespaceRegex: sel.NamespaceRegex,
			LabelSelector:  sel.LabelSelector,
		}
	}

	// Wait conditions
	if t.Spec.ConditionSpec != nil && len(t.Spec.ConditionSpec.Conditions) > 0 {
		it.Conditions = &internalWaitConditions{
			Timeout: t.Spec.ConditionSpec.Timeout,
			Delay:   t.Spec.ConditionSpec.Delay,
		}
		for _, c := range t.Spec.ConditionSpec.Conditions {
			ic := internalCondition{
				Type:   c.Type_,
				Reason: c.Reason,
				TTL:    c.Ttl,
			}
			if c.Status != nil {
				s := string(*c.Status)
				ic.Status = &s
			}
			it.Conditions.Items = append(it.Conditions.Items, ic)
		}
	}

	// Wait probes
	if t.Spec.ProbeSpec != nil && len(t.Spec.ProbeSpec.Probes) > 0 {
		it.Probes = &internalWaitProbes{
			Timeout: t.Spec.ProbeSpec.Timeout,
			Delay:   t.Spec.ProbeSpec.Delay,
		}
		for _, p := range t.Spec.ProbeSpec.Probes {
			it.Probes.Items = append(it.Probes.Items, internalProbe{
				Scheme:  p.Scheme,
				Host:    p.Host,
				Path:    p.Path,
				Port:    p.Port,
				Headers: p.Headers,
			})
		}
	}

	// Workflow selector (testSelector in v1)
	it.WorkflowSelector = internalTriggerSelector{
		Name:          t.Spec.TestSelector.Name,
		NameRegex:     t.Spec.TestSelector.NameRegex,
		LabelSelector: t.Spec.TestSelector.LabelSelector,
	}

	// Action parameters
	if t.Spec.ActionParameters != nil {
		it.Config = t.Spec.ActionParameters.Config
		it.Tags = t.Spec.ActionParameters.Tags
		it.Target = t.Spec.ActionParameters.Target
	}

	// Concurrency policy
	it.ConcurrencyPolicy = string(t.Spec.ConcurrencyPolicy)

	// Delay
	if t.Spec.Delay != nil {
		d := t.Spec.Delay.Duration
		it.Delay = &d
	}

	return it
}

// convertV2ToInternal converts a v2 WorkflowTrigger CRD to the internal representation.
func convertV2ToInternal(t *workflowtriggersv1.WorkflowTrigger) *internalTrigger {
	it := &internalTrigger{
		Name:      t.Name,
		Namespace: t.Namespace,
		Labels:    t.Labels,
		Source:    triggerSourceV2,
		Disabled:  t.Spec.Disabled,
	}

	// When
	it.Event = t.Spec.When.Event

	// Watch
	if t.Spec.Watch != nil {
		it.ResourceGroup = t.Spec.Watch.Resource.Group
		it.ResourceVersion = t.Spec.Watch.Resource.Version
		it.ResourceKind = t.Spec.Watch.Resource.Kind
		it.ResourceName = t.Spec.Watch.Resource.Name
		it.ResourceNamespace = t.Spec.Watch.Resource.Namespace

		if t.Spec.Watch.Selector != nil {
			it.Selector = &internalTriggerSelector{
				NameRegex:      t.Spec.Watch.Selector.NameRegex,
				NamespaceRegex: t.Spec.Watch.Selector.NamespaceRegex,
				LabelSelector:  t.Spec.Watch.Selector.LabelSelector,
			}
		}
	}

	// Match
	it.FieldConditions = t.Spec.Match

	// Wait
	if t.Spec.Wait != nil {
		if t.Spec.Wait.Conditions != nil && len(t.Spec.Wait.Conditions.Items) > 0 {
			it.Conditions = &internalWaitConditions{
				Timeout: t.Spec.Wait.Conditions.Timeout,
				Delay:   t.Spec.Wait.Conditions.Delay,
			}
			for _, c := range t.Spec.Wait.Conditions.Items {
				ic := internalCondition{
					Type:   c.Type,
					Reason: c.Reason,
					TTL:    c.TTL,
				}
				if c.Status != nil {
					s := string(*c.Status)
					ic.Status = &s
				}
				it.Conditions.Items = append(it.Conditions.Items, ic)
			}
		}

		if t.Spec.Wait.Probes != nil && len(t.Spec.Wait.Probes.Items) > 0 {
			it.Probes = &internalWaitProbes{
				Timeout: t.Spec.Wait.Probes.Timeout,
				Delay:   t.Spec.Wait.Probes.Delay,
			}
			for _, p := range t.Spec.Wait.Probes.Items {
				it.Probes.Items = append(it.Probes.Items, internalProbe{
					Scheme:  p.Scheme,
					Host:    p.Host,
					Path:    p.Path,
					Port:    p.Port,
					Headers: p.Headers,
				})
			}
		}
	}

	// Run
	it.WorkflowSelector = internalTriggerSelector{
		Name:          t.Spec.Run.Workflow.Name,
		NameRegex:     t.Spec.Run.Workflow.NameRegex,
		LabelSelector: t.Spec.Run.Workflow.LabelSelector,
	}
	it.Target = t.Spec.Run.Target
	it.ConcurrencyPolicy = t.Spec.Run.ConcurrencyPolicy

	if t.Spec.Run.Parameters != nil {
		it.Config = t.Spec.Run.Parameters.Config
		it.Tags = t.Spec.Run.Parameters.Tags
	}

	if t.Spec.Run.Delay != nil {
		d := t.Spec.Run.Delay.Duration
		it.Delay = &d
	}

	return it
}
