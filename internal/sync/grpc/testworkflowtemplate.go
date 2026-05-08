package grpc

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/grpc/metadata"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	syncv1 "github.com/kubeshop/testkube/pkg/proto/testkube/sync/v1"
)

// UpdateOrCreateTestWorkflowTemplate sends a request to the Control Plane informing that an change has occurred
// to a TestWorkflowTemplate object on this Agent.
func (c Client) UpdateOrCreateTestWorkflowTemplate(ctx context.Context, obj testworkflowsv1.TestWorkflowTemplate) error {
	jsonEncodedObj, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("json encode testworkflow object: %w", err)
	}

	// Execute with our own call timeout context to prevent stalling out.
	callCtx, cancel := context.WithTimeout(ctx, c.callTimeout)
	defer cancel()
	// Add metadata to the call.
	callCtx = metadata.AppendToOutgoingContext(callCtx, "organization-id", c.OrganizationId)

	if _, err := c.client.UpdateOrCreate(callCtx, &syncv1.UpdateOrCreateRequest{
		Payload: &syncv1.UpdateOrCreateRequest_TestWorkflowTemplate{
			TestWorkflowTemplate: &syncv1.TestWorkflowTemplate{
				Payload: jsonEncodedObj,
			},
		},
	}, c.callOpts...); err != nil {
		return fmt.Errorf("send request to update or create testworkflowtemplate: %w", err)
	}

	return nil
}

// DeleteTestWorkflowTemplate sends a request to the Control Plane informing that a TestWorkflowTemplate object has
// been removed from the observable scope of this Agent.
func (c Client) DeleteTestWorkflowTemplate(ctx context.Context, name string) error {
	// Execute with our own call timeout context to prevent stalling out.
	callCtx, cancel := context.WithTimeout(ctx, c.callTimeout)
	defer cancel()
	// Add metadata to the call.
	callCtx = metadata.AppendToOutgoingContext(callCtx, "organization-id", c.OrganizationId)

	if _, err := c.client.Delete(callCtx, &syncv1.DeleteRequest{
		Id: &syncv1.DeleteRequest_TestWorkflowTemplate{
			TestWorkflowTemplate: &syncv1.TestWorkflowTemplateId{
				Id: &name,
			},
		},
	}, c.callOpts...); err != nil {
		return fmt.Errorf("send request to delete testworkflowtemplate: %w", err)
	}

	return nil
}
