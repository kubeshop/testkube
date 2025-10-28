package grpc

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/grpc/metadata"

	executorv1 "github.com/kubeshop/testkube/api/executor/v1"
	syncv1 "github.com/kubeshop/testkube/pkg/proto/testkube/sync/v1"
)

// UpdateOrCreateWebhook sends a request to the Control Plane informing that an change has occurred
// to a Webhook object on this Agent.
func (c Client) UpdateOrCreateWebhook(ctx context.Context, obj executorv1.Webhook) error {
	jsonEncodedObj, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("json encode webhook object: %w", err)
	}

	// Execute with our own call timeout context to prevent stalling out.
	callCtx, cancel := context.WithTimeout(ctx, c.callTimeout)
	defer cancel()
	// Add metadata to the call.
	callCtx = metadata.AppendToOutgoingContext(callCtx, "organisation-id", c.OrganisationId)

	if _, err := c.client.UpdateOrCreate(callCtx, &syncv1.UpdateOrCreateRequest{
		Payload: &syncv1.UpdateOrCreateRequest_Webhook{
			Webhook: &syncv1.Webhook{
				Payload: jsonEncodedObj,
			},
		},
	}); err != nil {
		return fmt.Errorf("send request to update or create webhook: %w", err)
	}

	return nil
}

// DeleteWebhook sends a request to the Control Plane informing that a Webhook object has
// been removed from the observable scope of this Agent.
func (c Client) DeleteWebhook(ctx context.Context, name string) error {
	// Execute with our own call timeout context to prevent stalling out.
	callCtx, cancel := context.WithTimeout(ctx, c.callTimeout)
	defer cancel()
	// Add metadata to the call.
	callCtx = metadata.AppendToOutgoingContext(callCtx, "organisation-id", c.OrganisationId)

	if _, err := c.client.Delete(callCtx, &syncv1.DeleteRequest{
		Id: &syncv1.DeleteRequest_Webhook{
			Webhook: &syncv1.WebhookId{
				Id: &name,
			},
		},
	}); err != nil {
		return fmt.Errorf("send request to delete webhook: %w", err)
	}

	return nil
}
