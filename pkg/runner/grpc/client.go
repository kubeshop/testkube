package grpc

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/cloudflare/backoff"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	executionv1 "github.com/kubeshop/testkube/pkg/proto/testkube/testworkflow/execution/v1"
	signaturev1 "github.com/kubeshop/testkube/pkg/proto/testkube/testworkflow/signature/v1"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

// TODO: are these the correct values?
const (
	defaultCallTimeout  = time.Second * 30
	defaultPollInterval = time.Second
)

type runner interface {
	Execute(request executionworkertypes.ExecuteRequest) (*executionworkertypes.ExecuteResult, error)
	Pause(executionId string) error
	Resume(executionId string) error
	Abort(executionId string) error
	Cancel(executionId string) error
}

type workflowStore interface {
	Get(ctx context.Context, environmentId string, name string) (*testkube.TestWorkflow, error)
}

type Client struct {
	OrganisationId     string
	ControlPlaneConfig testworkflowconfig.ControlPlaneConfig

	client        executionv1.TestWorkflowExecutionServiceClient
	logger        *zap.SugaredLogger
	workflowStore workflowStore
	callOpts      []grpc.CallOption
	callTimeout   time.Duration
	runner        runner
	pollInterval  time.Duration
}

// NewClient creates a client for retrieving updates about executions.
func NewClient(conn grpc.ClientConnInterface, logger *zap.SugaredLogger, r runner, apiToken, organisationId string, controlPlane testworkflowconfig.ControlPlaneConfig, workflows workflowStore) Client {
	client := executionv1.NewTestWorkflowExecutionServiceClient(conn)

	opts := []grpc.CallOption{
		// In the event of a transient failure on the server wait for it to come back rather than
		// failing immediately.
		grpc.WaitForReady(true),
	}

	// Standalone deployment does not have an API Token for now
	if apiToken != "" {
		// Note: This requires TLS to be correctly configured, otherwise the gRPC library will
		// abort the connection. It is not secure to send authentication tokens over an
		// unencrypted connection so this is appropriate behaviour.
		opts = append(opts, grpc.PerRPCCredentials(oauth.TokenSource{
			TokenSource: oauth2.StaticTokenSource(&oauth2.Token{
				AccessToken: apiToken,
			}),
		}))
	}

	return Client{
		OrganisationId:     organisationId,
		ControlPlaneConfig: controlPlane,

		client:        client,
		logger:        logger,
		workflowStore: workflows,
		callOpts:      opts,
		callTimeout:   defaultCallTimeout,
		runner:        r,
		pollInterval:  defaultPollInterval,
	}
}

// IsSupported attempts to contact the Control Plane to determine whether or not there is an implementation
// of the required server to support this client.
// It will block until it receives either a successful response, returning true and indicating that the server
// supports this client.
// Or it receives an "Unimplemented" response, returning false and indicating that the server does not support
// this client and a fallback should be used instead.
// In the event of any other error, such as an authentication failure or a network failure, it will also fail
// after logging a message indicating that an error occurred.
func (c Client) IsSupported(ctx context.Context, environmentId string) bool {
	// Execute with our own call timeout context to prevent stalling out.
	callCtx, cancel := context.WithTimeout(ctx, c.callTimeout)
	defer cancel()
	// Add metadata to the call.
	// Environment ID should only be sent in some instances so it should be omitted
	// if it is not set to any specific value.
	callCtx = metadata.AppendToOutgoingContext(callCtx, "organisation-id", c.OrganisationId)
	if environmentId != "" {
		callCtx = metadata.AppendToOutgoingContext(callCtx, "environment-id", environmentId)
	}
	_, err := c.client.GetExecutionUpdates(callCtx, &executionv1.GetExecutionUpdatesRequest{}, c.callOpts...)
	code, ok := status.FromError(err)
	switch {
	case ok && code.Code() == codes.Unimplemented:
		// Server does not have the implementation for this client.
		return false
	case err != nil:
		c.logger.Warnw("Failed to check if server supports polling execution updates.",
			"error", err)
		return false
	}

	// Server has implementation.
	return true
}

// Start begins polling the control plane for updates to executions for this runner and passes
// the instructions to the runner to be implemented.
// The environmentId is not an optional field, setting it to an invalid value will cause the calls
// to set newly started executions as "SCHEDULING" to fail, resulting in duplicate start requests
// to be received. Whilst this is not a severe issue it could cause executions to become "stuck"
// in a queue at the Control Plane awaiting them going live on the runner.
func (c Client) Start(ctx context.Context, environmentId string) error {
	b := backoff.New(backoff.DefaultMaxDuration, c.pollInterval)
	ticker := time.Tick(c.pollInterval)
	req := &executionv1.GetExecutionUpdatesRequest{}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker:
			// Execute with our own call timeout context to prevent stalling out.
			callCtx, cancel := context.WithTimeout(ctx, c.callTimeout)
			// Add metadata to the call.
			// Environment ID should only be sent in some instances so it should be omitted
			// if it is not set to any specific value.
			callCtx = metadata.AppendToOutgoingContext(callCtx, "organisation-id", c.OrganisationId)
			if environmentId != "" {
				callCtx = metadata.AppendToOutgoingContext(callCtx, "environment-id", environmentId)
			}
			response, err := c.client.GetExecutionUpdates(callCtx, req, c.callOpts...)
			cancel()
			if err != nil {
				c.logger.Warnw("Failed to get execution updates, backing off before retrying.",
					"backoff", b.Duration(),
					"error", err)
				// In the event of an error wait for backoff before trying again.
				<-time.After(b.Duration())
				continue
			}
			// If request succeeds then backoffs can be reset.
			b.Reset()

			// Run in a separate function to clean up logic here.
			// THIS FUNCTION SHOULD BLOCK UNTIL PROCESSING THE REQUEST IS COMPLETE.
			// If you put this in a separate goroutine then you risk polling faster
			// than you can process responses and introducing race conditions.
			c.executeResponse(ctx, response)
		}
	}
}

func (c Client) executeResponse(ctx context.Context, response *executionv1.GetExecutionUpdatesResponse) {
	var wg sync.WaitGroup
	for _, transition := range response.GetUpdate() {
		switch transition.GetTransitionTo() {
		case executionv1.ExecutionState_EXECUTION_STATE_RUNNING:
			wg.Go(func() {
				if err := c.runner.Resume(transition.GetExecutionId()); err != nil {
					c.logger.Errorw("Failed to resume execution",
						"execution", transition.GetExecutionId(),
						"error", err)
				}
			})
		case executionv1.ExecutionState_EXECUTION_STATE_PAUSED:
			wg.Go(func() {
				if err := c.runner.Pause(transition.GetExecutionId()); err != nil {
					c.logger.Errorw("Failed to pause execution",
						"execution", transition.GetExecutionId(),
						"error", err)
				}
			})
		case executionv1.ExecutionState_EXECUTION_STATE_CANCELLED:
			wg.Go(func() {
				if err := c.runner.Cancel(transition.GetExecutionId()); err != nil {
					c.logger.Errorw("Failed to cancel execution",
						"execution", transition.GetExecutionId(),
						"error", err)
				}
			})
		case executionv1.ExecutionState_EXECUTION_STATE_ABORTED:
			wg.Go(func() {
				if err := c.runner.Abort(transition.GetExecutionId()); err != nil {
					c.logger.Errorw("Failed to abort execution",
						"execution", transition.GetExecutionId(),
						"error", err)
				}
			})
		default:
			c.logger.Infow("Unknown execution state transition request received, ignoring this request.",
				"transition", transition.GetTransitionTo(),
				"executionId", transition.GetExecutionId())
		}
	}
	for _, start := range response.GetStart() {
		// Grab the full workflow.
		workflowResponse, err := c.client.GetExecutionWorkflow(ctx, &executionv1.GetExecutionWorkflowRequest{
			ExecutionId:   start.ExecutionId,
			EnvironmentId: start.EnvironmentId,
		}, c.callOpts...)
		if err != nil {
			// We cannot process this request as we do not know about the workflow to be executed.
			c.logger.Errorw("Failed to retrieve workflow for execution, this execution will not be started.",
				"executionId", start.GetExecutionId(),
				"workflow name", start.GetWorkflowName(),
				"error", err)
			continue
		}
		// Deserialise the workflow.
		var workflow testworkflowsv1.TestWorkflow
		if err := json.Unmarshal(workflowResponse.GetWorkflow().GetJson(), &workflow); err != nil {
			c.logger.Errorw("Failed to unmarshal workflow for execution, this execution will not be started.",
				"executionId", start.GetExecutionId(),
				"workflow name", start.GetWorkflowName(),
				"error", err)
			continue
		}
		wg.Go(func() {
			result, err := c.runner.Execute(executionworkertypes.ExecuteRequest{
				Token: start.GetExecutionToken(),
				Runtime: &executionworkertypes.Runtime{
					Variables: start.GetVariableOverrides(),
				},
				Execution: testworkflowconfig.ExecutionConfig{
					Id:              start.GetExecutionId(),
					GroupId:         start.GetGroupId(),
					Name:            start.GetName(),
					Number:          start.GetNumber(),
					ScheduledAt:     start.GetQueuedAt().AsTime(),
					DisableWebhooks: start.GetDisableWebhooks(),
					Debug:           false,
					OrganizationId:  c.OrganisationId,
					EnvironmentId:   start.GetEnvironmentId(),
					ParentIds:       strings.Join(start.AncestorExecutionIds, "/"),
				},
				Workflow:     workflow,
				ControlPlane: c.ControlPlaneConfig,
			})
			if err != nil {
				c.logger.Errorw("Failed to start execution.",
					"executionId", start.GetExecutionId(),
					"error", err)
				// Execute with our own call timeout context to prevent stalling out.
				callCtx, cancel := context.WithTimeout(ctx, c.callTimeout)
				// Add required metadata to the call.
				callCtx = metadata.AppendToOutgoingContext(callCtx,
					"organisation-id", c.OrganisationId,
					"environment-id", start.GetEnvironmentId())
				// Report the error to the control plane to prevent getting the execution on
				// subsequent calls.
				_, callErr := c.client.DeclineExecution(callCtx, &executionv1.DeclineExecutionRequest{
					ExecutionId: start.ExecutionId,
				}, c.callOpts...)
				cancel()
				if callErr != nil {
					c.logger.Errorw("Failed to report execution start error.",
						"executionId", start.GetExecutionId(),
						"error", callErr)
					return
				}
				return
			}

			if result.Redundant {
				// Execution already existed.
				return
			}

			// Execute with our own call timeout context to prevent stalling out.
			callCtx, cancel := context.WithTimeout(ctx, c.callTimeout)
			// Add required metadata to the call.
			callCtx = metadata.AppendToOutgoingContext(callCtx,
				"organisation-id", c.OrganisationId,
				"environment-id", start.GetEnvironmentId())
			// Update the control plane that the execution is awaiting scheduling by Kubernetes.
			_, err = c.client.AcceptExecution(callCtx, &executionv1.AcceptExecutionRequest{
				ExecutionId: start.ExecutionId,
				Namespace:   &result.Namespace,
				Signature:   translateSignature(result.Signature),
			}, c.callOpts...)
			cancel()
			if err != nil {
				c.logger.Errorw("Failed to set execution scheduling",
					"executionId", start.GetExecutionId(),
					"error", err)
				return
			}
		})
	}
	// Wait for everything to finish before returning.
	wg.Wait()
}

// translateSignature recursively translates signatures in order for them to be transmittable via gRPC.
func translateSignature(sigs []testkube.TestWorkflowSignature) []*signaturev1.Signature {
	var ret []*signaturev1.Signature
	for _, sig := range sigs {
		ret = append(ret, &signaturev1.Signature{
			Ref:      &sig.Ref,
			Name:     &sig.Name,
			Category: &sig.Category,
			Optional: &sig.Optional,
			Negative: &sig.Negative,
			Children: translateSignature(sig.Children),
		})
	}
	return ret
}
