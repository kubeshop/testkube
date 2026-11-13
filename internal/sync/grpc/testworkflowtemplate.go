package grpc

import (
	"context"
	"encoding/json"
	"fmt"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	syncagent "github.com/kubeshop/testkube/internal/sync"
	syncv1 "github.com/kubeshop/testkube/pkg/proto/testkube/sync/v1"
)

// UpdateOrCreateTestWorkflowTemplate sends a request to the Control Plane informing that an change has occurred
// to a TestWorkflowTemplate object on this Agent.
func (c Client) UpdateOrCreateTestWorkflowTemplate(ctx context.Context, target syncagent.Target, obj testworkflowsv1.TestWorkflowTemplate) error {
	obj.Labels = withSourceNamespace(obj.Labels, target.Namespace)
	jsonEncodedObj, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("json encode testworkflow object: %w", err)
	}

	callCtx, cancel := c.callContext(ctx, target)
	defer cancel()

	if _, err := c.client.UpdateOrCreate(callCtx, &syncv1.UpdateOrCreateRequest{
		Namespace: optionalNamespace(target.Namespace),
		Payload: &syncv1.UpdateOrCreateRequest_TestWorkflowTemplate{
			TestWorkflowTemplate: &syncv1.TestWorkflowTemplate{
				Payload: jsonEncodedObj,
			},
		},
	}, c.callOpts...); err != nil {
		return fmt.Errorf("send request to update or create testworkflowtemplate: %w", translateError(err))
	}

	return nil
}

// DeleteTestWorkflowTemplate sends a request to the Control Plane informing that a TestWorkflowTemplate object has
// been removed from the observable scope of this Agent.
func (c Client) DeleteTestWorkflowTemplate(ctx context.Context, target syncagent.Target, name string) error {
	callCtx, cancel := c.callContext(ctx, target)
	defer cancel()

	if _, err := c.client.Delete(callCtx, &syncv1.DeleteRequest{
		Id: &syncv1.DeleteRequest_TestWorkflowTemplate{
			TestWorkflowTemplate: &syncv1.TestWorkflowTemplateId{
				Id:        &name,
				Namespace: optionalNamespace(target.Namespace),
			},
		},
	}, c.callOpts...); err != nil {
		return fmt.Errorf("send request to delete testworkflowtemplate: %w", translateError(err))
	}

	return nil
}
