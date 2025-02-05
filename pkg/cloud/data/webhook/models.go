package webhook

import "github.com/kubeshop/testkube/pkg/api/v1/testkube"

type WebhookExecutionCollectResultRequest struct {
	ExecutionID  string             `json:"executionId"`
	Name         string             `json:"name"`
	WebhookName  string             `json:"webhookName"`
	EventType    testkube.EventType `json:"eventType"`
	ErrorMessage string             `json:"errorMessage"`
	StatusCode   int                `json:"statusCode"`
}

type WebhookExecutionCollectResultResponse struct{}
