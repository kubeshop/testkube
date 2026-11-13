package grpc

import (
	"context"
	"encoding/json"
	"fmt"

	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	syncagent "github.com/kubeshop/testkube/internal/sync"
	syncv1 "github.com/kubeshop/testkube/pkg/proto/testkube/sync/v1"
)

// UpdateOrCreateTestTrigger sends a request to the Control Plane informing that an change has occurred
// to a TestTrigger object on this Agent.
func (c Client) UpdateOrCreateTestTrigger(ctx context.Context, target syncagent.Target, obj testtriggersv1.TestTrigger) error {
	obj.Labels = withSourceNamespace(obj.Labels, target.Namespace)
	jsonEncodedObj, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("json encode testtrigger object: %w", err)
	}

	callCtx, cancel := c.callContext(ctx, target)
	defer cancel()

	if _, err := c.client.UpdateOrCreate(callCtx, &syncv1.UpdateOrCreateRequest{
		Namespace: optionalNamespace(target.Namespace),
		Payload: &syncv1.UpdateOrCreateRequest_TestTrigger{
			TestTrigger: &syncv1.TestTrigger{
				Payload: jsonEncodedObj,
			},
		},
	}, c.callOpts...); err != nil {
		return fmt.Errorf("send request to update or create testtrigger: %w", translateError(err))
	}

	return nil
}

// DeleteTestTrigger sends a request to the Control Plane informing that a TestTrigger object has
// been removed from the observable scope of this Agent.
func (c Client) DeleteTestTrigger(ctx context.Context, target syncagent.Target, name string) error {
	callCtx, cancel := c.callContext(ctx, target)
	defer cancel()

	if _, err := c.client.Delete(callCtx, &syncv1.DeleteRequest{
		Id: &syncv1.DeleteRequest_TestTrigger{
			TestTrigger: &syncv1.TestTriggerId{
				Id:        &name,
				Namespace: optionalNamespace(target.Namespace),
			},
		},
	}, c.callOpts...); err != nil {
		return fmt.Errorf("send request to delete testtrigger: %w", translateError(err))
	}

	return nil
}
