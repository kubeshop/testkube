package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestWorkflowExecutionSummary contains TestWorkflow execution summary
type TestWorkflowExecutionSummary struct {
	// unique execution identifier
	Id string `json:"id"`
	// execution name
	Name string `json:"name"`
	// sequence number for the execution
	Number int32 `json:"number,omitempty"`
	// when the execution has been scheduled to run
	ScheduledAt metav1.Time `json:"scheduledAt,omitempty"`
	// when the execution result's status has changed last time (queued, passed, failed)
	StatusAt metav1.Time                `json:"statusAt,omitempty"`
	Result   *TestWorkflowResultSummary `json:"result,omitempty"`
	Workflow *TestWorkflowSummary       `json:"workflow"`
	// test workflow execution tags
	Tags map[string]string `json:"tags,omitempty"`
	// running context for the test workflow execution (Pro edition only)
	RunningContext *TestWorkflowRunningContext `json:"runningContext,omitempty"`
}

// TestWorkflowResultSummary defines TestWorkflow result summary
type TestWorkflowResultSummary struct {
	Status          *TestWorkflowStatus `json:"status"`
	PredictedStatus *TestWorkflowStatus `json:"predictedStatus"`
	// when the pod was created
	QueuedAt metav1.Time `json:"queuedAt,omitempty"`
	// when the pod has been successfully assigned
	StartedAt metav1.Time `json:"startedAt,omitempty"`
	// when the pod has been completed
	FinishedAt metav1.Time `json:"finishedAt,omitempty"`
	// Go-formatted (human-readable) duration
	Duration string `json:"duration,omitempty"`
	// Go-formatted (human-readable) duration (incl. pause)
	TotalDuration string `json:"totalDuration,omitempty"`
	// Duration in milliseconds
	DurationMs int32 `json:"durationMs"`
	// Duration in milliseconds (incl. pause)
	TotalDurationMs int32 `json:"totalDurationMs"`
	// Pause duration in milliseconds
	PausedMs int32 `json:"pausedMs"`
}

// TestWorkflowSummary fas TestWorkflow summary
type TestWorkflowSummary struct {
	Name        string            `json:"name,omitempty"`
	Namespace   string            `json:"namespace,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}
