package testkube

// WorkflowTrigger is the flat REST-API shape for WorkflowTrigger v2.
// It mirrors the cloud-api representation so CLI + control-plane share a contract.
// The CRD (api/workflowtriggers/v1) wraps Watch/When/.../Run inside Spec; this
// flattens them to the top level for ergonomics.
type WorkflowTrigger struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Disabled    bool              `json:"disabled,omitempty"`

	Watch *WorkflowTriggerWatch           `json:"watch,omitempty"`
	When  WorkflowTriggerWhen             `json:"when"`
	Match []WorkflowTriggerFieldCondition `json:"match,omitempty"`
	Wait  *WorkflowTriggerWait            `json:"wait,omitempty"`
	Run   WorkflowTriggerRun              `json:"run"`
}

type WorkflowTriggerWatch struct {
	Resource WorkflowTriggerResource  `json:"resource"`
	Selector *WorkflowTriggerSelector `json:"selector,omitempty"`
}

type WorkflowTriggerResource struct {
	Group     string `json:"group,omitempty"`
	Version   string `json:"version,omitempty"`
	Kind      string `json:"kind"`
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

type WorkflowTriggerSelector struct {
	NameRegex      string                   `json:"nameRegex,omitempty"`
	NamespaceRegex string                   `json:"namespaceRegex,omitempty"`
	LabelSelector  *WorkflowTriggerLabelSel `json:"labelSelector,omitempty"`
}

type WorkflowTriggerLabelSel struct {
	MatchLabels map[string]string `json:"matchLabels,omitempty"`
}

type WorkflowTriggerWhen struct {
	Event string `json:"event,omitempty"`
}

type WorkflowTriggerFieldCondition struct {
	Path     string `json:"path"`
	Operator string `json:"operator"`
	Value    string `json:"value,omitempty"`
}

type WorkflowTriggerWait struct {
	Conditions *WorkflowTriggerWaitConditions `json:"conditions,omitempty"`
	Probes     *WorkflowTriggerWaitProbes     `json:"probes,omitempty"`
}

type WorkflowTriggerWaitConditions struct {
	Items   []WorkflowTriggerCondition `json:"items"`
	Timeout int32                      `json:"timeout,omitempty"`
	Delay   int32                      `json:"delay,omitempty"`
}

type WorkflowTriggerCondition struct {
	Type   string `json:"type"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
	TTL    int32  `json:"ttl,omitempty"`
}

type WorkflowTriggerWaitProbes struct {
	Items   []WorkflowTriggerProbe `json:"items"`
	Timeout int32                  `json:"timeout,omitempty"`
	Delay   int32                  `json:"delay,omitempty"`
}

type WorkflowTriggerProbe struct {
	Scheme  string            `json:"scheme,omitempty"`
	Host    string            `json:"host,omitempty"`
	Path    string            `json:"path,omitempty"`
	Port    int32             `json:"port,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type WorkflowTriggerRun struct {
	Workflow          WorkflowTriggerWorkflowSelector `json:"workflow"`
	Parameters        *WorkflowTriggerRunParameters   `json:"parameters,omitempty"`
	ConcurrencyPolicy string                          `json:"concurrencyPolicy,omitempty"`
	Delay             string                          `json:"delay,omitempty"`
}

type WorkflowTriggerWorkflowSelector struct {
	Name          string                   `json:"name,omitempty"`
	NameRegex     string                   `json:"nameRegex,omitempty"`
	LabelSelector *WorkflowTriggerLabelSel `json:"labelSelector,omitempty"`
}

type WorkflowTriggerRunParameters struct {
	Config map[string]string `json:"config,omitempty"`
	Tags   map[string]string `json:"tags,omitempty"`
}

func (w WorkflowTrigger) GetName() string                   { return w.Name }
func (w WorkflowTrigger) GetNamespace() string              { return w.Namespace }
func (w WorkflowTrigger) GetLabels() map[string]string      { return w.Labels }
func (w WorkflowTrigger) GetAnnotations() map[string]string { return w.Annotations }
