package grpc

import (
	"context"
	"encoding/json"
	"fmt"

	executorv1 "github.com/kubeshop/testkube/api/executor/v1"
	syncagent "github.com/kubeshop/testkube/internal/sync"
	syncv1 "github.com/kubeshop/testkube/pkg/proto/testkube/sync/v1"
)

// UpdateOrCreateWebhook sends a request to the Control Plane informing that an change has occurred
// to a Webhook object on this Agent.
func (c Client) UpdateOrCreateWebhook(ctx context.Context, target syncagent.Target, obj executorv1.Webhook) error {
	obj.Labels = withSourceNamespace(obj.Labels, target.Namespace)
	jsonEncodedObj, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("json encode webhook object: %w", err)
	}

	callCtx, cancel := c.callContext(ctx, target)
	defer cancel()

	if _, err := c.client.UpdateOrCreate(callCtx, &syncv1.UpdateOrCreateRequest{
		Namespace: optionalNamespace(target.Namespace),
		Payload: &syncv1.UpdateOrCreateRequest_Webhook{
			Webhook: &syncv1.Webhook{
				Payload: jsonEncodedObj,
			},
		},
	}, c.callOpts...); err != nil {
		return fmt.Errorf("send request to update or create webhook: %w", translateError(err))
	}

	return nil
}

// DeleteWebhook sends a request to the Control Plane informing that a Webhook object has
// been removed from the observable scope of this Agent.
func (c Client) DeleteWebhook(ctx context.Context, target syncagent.Target, name string) error {
	callCtx, cancel := c.callContext(ctx, target)
	defer cancel()

	if _, err := c.client.Delete(callCtx, &syncv1.DeleteRequest{
		Id: &syncv1.DeleteRequest_Webhook{
			Webhook: &syncv1.WebhookId{
				Id:        &name,
				Namespace: optionalNamespace(target.Namespace),
			},
		},
	}, c.callOpts...); err != nil {
		return fmt.Errorf("send request to delete webhook: %w", translateError(err))
	}

	return nil
}
