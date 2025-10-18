package grpc

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/grpc/metadata"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	syncv1 "github.com/kubeshop/testkube/pkg/proto/testkube/sync/v1"
)

// UpdateOrCreateTestWorkflow sends a request to the Control Plane informing that an change has occurred
// to a TestWorkflow object on this Agent.
func (c Client) UpdateOrCreateTestWorkflow(ctx context.Context, obj testworkflowsv1.TestWorkflow) error {
	jsonEncodedObj, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("json encode testworkflow object: %w", err)
	}

	// Execute with our own call timeout context to prevent stalling out.
	callCtx, cancel := context.WithTimeout(ctx, c.callTimeout)
	defer cancel()
	// Add metadata to the call.
	callCtx = metadata.AppendToOutgoingContext(callCtx, "organisation-id", c.OrganisationId)

	if _, err := c.client.UpdateOrCreate(callCtx, &syncv1.UpdateOrCreateRequest{
		Payload: &syncv1.UpdateOrCreateRequest_TestWorkflow{
			TestWorkflow: &syncv1.TestWorkflow{
				Payload: jsonEncodedObj,
			},
		},
	}); err != nil {
		return fmt.Errorf("send request to update or create testworkflow: %w", err)
	}

	return nil
}

// DeleteTestWorkflow sends a request to the Control Plane informing that a TestWorkflow object has
// been removed from the observable scope of this Agent.
func (c Client) DeleteTestWorkflow(ctx context.Context, name string) error {
	// Execute with our own call timeout context to prevent stalling out.
	callCtx, cancel := context.WithTimeout(ctx, c.callTimeout)
	defer cancel()
	// Add metadata to the call.
	callCtx = metadata.AppendToOutgoingContext(callCtx, "organisation-id", c.OrganisationId)

	if _, err := c.client.Delete(callCtx, &syncv1.DeleteRequest{
		Id: &syncv1.DeleteRequest_TestWorkflow{
			TestWorkflow: &syncv1.TestWorkflowId{
				Id: &name,
			},
		},
	}); err != nil {
		return fmt.Errorf("send request to delete testworkflow: %w", err)
	}

	return nil
}
