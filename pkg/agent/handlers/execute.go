package handlers

import (
	"context"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/pkg/client/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/agent"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
	"github.com/kubeshop/testkube/pkg/utils/codec"
)

// TODO - valid error handling

func NewExecuteTestWorkflowHandler(
	testWorkflowExecutor testworkflowexecutor.TestWorkflowExecutor,
	testWorkflowsClient testworkflowsv1.Interface,
	log *zap.SugaredLogger,
) agent.CommandHandler {
	return func(ctx context.Context, c *cloud.ExecuteRequest) *cloud.ExecuteResponse {

		log = log.With("messageId", c.MessageId)
		request, err := codec.FromJSONBytes[testkube.TestWorkflowExecutionRequest](c.Body)
		if err != nil {
			return BadRequestResponse(cloud.ExecuteCommand, c.MessageId, errors.Wrap(err, "can't decode request body"))
		}
		log.Infow("got execute request", "request", request)

		workflow, err := testWorkflowsClient.Get(request.TestWorkflowExecutionName)
		if err != nil {
			return NotFoundResponse(cloud.ExecuteCommand, c.MessageId, errors.Wrap(err, "can't get workflow"))
		}
		log.Infow("got workflow", "request", request)

		execution, err := testWorkflowExecutor.Execute(ctx, *workflow, request)
		if err != nil {
			return InternalServerErrorResponse(cloud.ExecuteCommand, c.MessageId, errors.Wrap(err, "can't execute workflow"))
		}

		log.Infow("executed workflow", "execution", execution)

		body, err := codec.ToJSONBytes(execution)
		if err != nil {
			return InternalServerErrorResponse(cloud.ExecuteCommand, c.MessageId, errors.Wrap(err, "can't encode response data to JSON"))
		}
		return SuccessResponse(cloud.ExecuteCommand, c.MessageId, body)
	}
}
