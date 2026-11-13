package grpc

import (
	"context"
	"encoding/json"
	"fmt"

	executorv1 "github.com/kubeshop/testkube/api/executor/v1"
	syncagent "github.com/kubeshop/testkube/internal/sync"
	syncv1 "github.com/kubeshop/testkube/pkg/proto/testkube/sync/v1"
)

// UpdateOrCreateWebhookTemplate sends a request to the Control Plane informing that an change has occurred
// to a WebhookTemplate object on this Agent.
func (c Client) UpdateOrCreateWebhookTemplate(ctx context.Context, target syncagent.Target, obj executorv1.WebhookTemplate) error {
	obj.Labels = withSourceNamespace(obj.Labels, target.Namespace)
	jsonEncodedObj, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("json encode webhooktemplate object: %w", err)
	}

	callCtx, cancel := c.callContext(ctx, target)
	defer cancel()

	if _, err := c.client.UpdateOrCreate(callCtx, &syncv1.UpdateOrCreateRequest{
		Namespace: optionalNamespace(target.Namespace),
		Payload: &syncv1.UpdateOrCreateRequest_WebhookTemplate{
			WebhookTemplate: &syncv1.WebhookTemplate{
				Payload: jsonEncodedObj,
			},
		},
	}, c.callOpts...); err != nil {
		return fmt.Errorf("send request to update or create webhooktemplate: %w", translateError(err))
	}

	return nil
}

// DeleteWebhookTemplate sends a request to the Control Plane informing that a WebhookTemplate object has
// been removed from the observable scope of this Agent.
func (c Client) DeleteWebhookTemplate(ctx context.Context, target syncagent.Target, name string) error {
	callCtx, cancel := c.callContext(ctx, target)
	defer cancel()

	if _, err := c.client.Delete(callCtx, &syncv1.DeleteRequest{
		Id: &syncv1.DeleteRequest_WebhookTemplate{
			WebhookTemplate: &syncv1.WebhookTemplateId{
				Id:        &name,
				Namespace: optionalNamespace(target.Namespace),
			},
		},
	}, c.callOpts...); err != nil {
		return fmt.Errorf("send request to delete webhooktemplate: %w", translateError(err))
	}

	return nil
}
