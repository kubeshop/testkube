package testworkflow

type OutputPresignSaveLogRequest struct {
	ID           string `json:"id"`
	WorkflowName string `json:"workflowName"`
}

type OutputPresignSaveLogResponse struct {
	URL string `json:"url"`
}

type OutputPresignReadLogRequest struct {
	ID           string `json:"id"`
	WorkflowName string `json:"workflowName"`
}

type OutputPresignReadLogResponse struct {
	URL string `json:"url"`
}

type OutputHasLogRequest struct {
	ID           string `json:"id"`
	WorkflowName string `json:"workflowName"`
}

type OutputHasLogResponse struct {
	Has bool `json:"has"`
}
