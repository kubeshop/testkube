package v1

// SchedulerPolicy controls whether a targeted execution should be created.
// +kubebuilder:validation:Enum=OnlyWhenMatches
type SchedulerPolicy string

const SchedulerPolicyOnlyWhenMatches SchedulerPolicy = "OnlyWhenMatches"

// +kubebuilder:object:generate=true
type Target struct {
	SchedulerPolicy SchedulerPolicy     `json:"schedulerPolicy,omitempty"`
	Match           map[string][]string `json:"match,omitempty"`
	Not             map[string][]string `json:"not,omitempty"`
	Replicate       []string            `json:"replicate,omitempty"`
}
