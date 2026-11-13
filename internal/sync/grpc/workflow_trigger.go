package grpc

import (
	"context"
	"encoding/json"
	"fmt"

	workflowtriggersv1 "github.com/kubeshop/testkube/api/workflowtriggers/v1"
	syncagent "github.com/kubeshop/testkube/internal/sync"
	syncv1 "github.com/kubeshop/testkube/pkg/proto/testkube/sync/v1"
)

// UpdateOrCreateWorkflowTrigger sends a request to the Control Plane informing that a change
// has occurred to a WorkflowTrigger object on this Agent.
func (c Client) UpdateOrCreateWorkflowTrigger(ctx context.Context, target syncagent.Target, obj workflowtriggersv1.WorkflowTrigger) error {
	obj.Labels = withSourceNamespace(obj.Labels, target.Namespace)
	jsonEncodedObj, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("json encode workflowtrigger object: %w", err)
	}

	callCtx, cancel := c.callContext(ctx, target)
	defer cancel()

	if _, err := c.client.UpdateOrCreate(callCtx, &syncv1.UpdateOrCreateRequest{
		Namespace: optionalNamespace(target.Namespace),
		Payload: &syncv1.UpdateOrCreateRequest_WorkflowTrigger{
			WorkflowTrigger: &syncv1.WorkflowTrigger{
				Payload: jsonEncodedObj,
			},
		},
	}, c.callOpts...); err != nil {
		return fmt.Errorf("send request to update or create workflowtrigger: %w", translateError(err))
	}

	return nil
}

// DeleteWorkflowTrigger sends a request to the Control Plane informing that a WorkflowTrigger object has
// been removed from the observable scope of this Agent.
func (c Client) DeleteWorkflowTrigger(ctx context.Context, target syncagent.Target, name string) error {
	callCtx, cancel := c.callContext(ctx, target)
	defer cancel()

	if _, err := c.client.Delete(callCtx, &syncv1.DeleteRequest{
		Id: &syncv1.DeleteRequest_WorkflowTrigger{
			WorkflowTrigger: &syncv1.WorkflowTriggerId{
				Id:        &name,
				Namespace: optionalNamespace(target.Namespace),
			},
		},
	}, c.callOpts...); err != nil {
		return fmt.Errorf("send request to delete workflowtrigger: %w", translateError(err))
	}

	return nil
}
