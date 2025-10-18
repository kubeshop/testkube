package grpc

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/grpc/metadata"

	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	syncv1 "github.com/kubeshop/testkube/pkg/proto/testkube/sync/v1"
)

// UpdateOrCreateTestTrigger sends a request to the Control Plane informing that an change has occurred
// to a TestTrigger object on this Agent.
func (c Client) UpdateOrCreateTestTrigger(ctx context.Context, obj testtriggersv1.TestTrigger) error {
	jsonEncodedObj, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("json encode testtrigger object: %w", err)
	}

	// Execute with our own call timeout context to prevent stalling out.
	callCtx, cancel := context.WithTimeout(ctx, c.callTimeout)
	defer cancel()
	// Add metadata to the call.
	callCtx = metadata.AppendToOutgoingContext(callCtx, "organisation-id", c.OrganisationId)

	if _, err := c.client.UpdateOrCreate(callCtx, &syncv1.UpdateOrCreateRequest{
		Payload: &syncv1.UpdateOrCreateRequest_TestTrigger{
			TestTrigger: &syncv1.TestTrigger{
				Payload: jsonEncodedObj,
			},
		},
	}); err != nil {
		return fmt.Errorf("send request to update or create testtrigger: %w", err)
	}

	return nil
}

// DeleteTestTrigger sends a request to the Control Plane informing that a TestTrigger object has
// been removed from the observable scope of this Agent.
func (c Client) DeleteTestTrigger(ctx context.Context, name string) error {
	// Execute with our own call timeout context to prevent stalling out.
	callCtx, cancel := context.WithTimeout(ctx, c.callTimeout)
	defer cancel()
	// Add metadata to the call.
	callCtx = metadata.AppendToOutgoingContext(callCtx, "organisation-id", c.OrganisationId)

	if _, err := c.client.Delete(callCtx, &syncv1.DeleteRequest{
		Id: &syncv1.DeleteRequest_TestTrigger{
			TestTrigger: &syncv1.TestTriggerId{
				Id: &name,
			},
		},
	}); err != nil {
		return fmt.Errorf("send request to delete testtrigger: %w", err)
	}

	return nil
}
