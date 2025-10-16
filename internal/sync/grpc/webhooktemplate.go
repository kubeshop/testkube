package grpc

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/grpc/metadata"

	executorv1 "github.com/kubeshop/testkube/api/executor/v1"
	syncv1 "github.com/kubeshop/testkube/pkg/proto/testkube/sync/v1"
)

// UpdateOrCreateWebhookTemplate sends a request to the Control Plane informing that an change has occurred
// to a WebhookTemplate object on this Agent.
func (c Client) UpdateOrCreateWebhookTemplate(ctx context.Context, obj executorv1.WebhookTemplate) error {
	jsonEncodedObj, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("json encode webhooktemplate object: %w", err)
	}

	// Execute with our own call timeout context to prevent stalling out.
	callCtx, cancel := context.WithTimeout(ctx, c.callTimeout)
	defer cancel()
	// Add metadata to the call.
	callCtx = metadata.AppendToOutgoingContext(callCtx, "organisation-id", c.OrganisationId)

	if _, err := c.client.UpdateOrCreate(callCtx, &syncv1.UpdateOrCreateRequest{
		Payload: &syncv1.UpdateOrCreateRequest_WebhookTemplate{
			WebhookTemplate: &syncv1.WebhookTemplate{
				Payload: jsonEncodedObj,
			},
		},
	}); err != nil {
		return fmt.Errorf("send request to update or create webhooktemplate: %w", err)
	}

	return nil
}

// DeleteWebhookTemplate sends a request to the Control Plane informing that a WebhookTemplate object has
// been removed from the observable scope of this Agent.
func (c Client) DeleteWebhookTemplate(ctx context.Context, name string) error {
	// Execute with our own call timeout context to prevent stalling out.
	callCtx, cancel := context.WithTimeout(ctx, c.callTimeout)
	defer cancel()
	// Add metadata to the call.
	callCtx = metadata.AppendToOutgoingContext(callCtx, "organisation-id", c.OrganisationId)

	if _, err := c.client.Delete(callCtx, &syncv1.DeleteRequest{
		Id: &syncv1.DeleteRequest_WebhookTemplate{
			WebhookTemplate: &syncv1.WebhookTemplateId{
				Id: &name,
			},
		},
	}); err != nil {
		return fmt.Errorf("send request to delete webhooktemplate: %w", err)
	}

	return nil
}
