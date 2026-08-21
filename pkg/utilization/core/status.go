package core

const (
	ResourceMetricsWarningOutputName = "resource-metrics-warning"
	ResourceMetricsStatusOutputName  = "resource-metrics-status"
)

const ResourceMetricsReasonNoSamples = "no-samples"

type ResourceMetricsStatus struct {
	Recorded bool   `json:"recorded"`
	Reason   string `json:"reason,omitempty"`
}
