package testkube

// TestTriggerFieldCondition defines a field-level match condition. Entries are ANDed.
type TestTriggerFieldCondition struct {
	// Dot-path to a field on the K8s object (e.g. ".spec.replicas", ".spec.template.spec.containers.0.image")
	Path string `json:"path"`
	// Comparison operator. One of: equals, not_equals, exists, not_exists, changed, changed_to, changed_from.
	Operator TestTriggerFieldOperator `json:"operator"`
	// Value to compare against. Required for equals, not_equals, changed_to, changed_from.
	Value string `json:"value,omitempty"`
}

// TestTriggerFieldOperator : supported comparison operators for TestTrigger match conditions
type TestTriggerFieldOperator string

// List of TestTriggerFieldOperator
const (
	TestTriggerFieldOperatorEquals      TestTriggerFieldOperator = "equals"
	TestTriggerFieldOperatorNotEquals   TestTriggerFieldOperator = "not_equals"
	TestTriggerFieldOperatorExists      TestTriggerFieldOperator = "exists"
	TestTriggerFieldOperatorNotExists   TestTriggerFieldOperator = "not_exists"
	TestTriggerFieldOperatorChanged     TestTriggerFieldOperator = "changed"
	TestTriggerFieldOperatorChangedTo   TestTriggerFieldOperator = "changed_to"
	TestTriggerFieldOperatorChangedFrom TestTriggerFieldOperator = "changed_from"
)
