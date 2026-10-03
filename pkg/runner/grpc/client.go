package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cloudflare/backoff"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/grpcutils"
	executionv1 "github.com/kubeshop/testkube/pkg/proto/testkube/testworkflow/execution/v1"
	signaturev1 "github.com/kubeshop/testkube/pkg/proto/testkube/testworkflow/signature/v1"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/registry"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

const (
	// defaultCallTimeout bounds a single RPC. Note that WaitForReady(true) below
	// means a control plane that is simply down does not fail fast, so a failed
	// cycle costs the whole timeout before the backoff even starts.
	defaultCallTimeout = time.Second * 30
	// defaultPollInterval is how often the control plane is asked for work. It is
	// a latency floor, not a throughput cap: one poll can return a whole batch.
	defaultPollInterval = time.Second
	// defaultMaxPollBackoff caps the retry delay after a failed poll.
	//
	// The library default is six hours. Combined with a Duration() that was called
	// twice per failure, nine consecutive failures were enough to put a runner to
	// sleep for hours and stop it consuming its backlog entirely - which is what
	// turned one slow query into a permanent stall. The ladder here is
	// 1s, 2s, 4s, 8s, 16s, 30s, so a runner rejoins within about a minute of the
	// control plane recovering.
	defaultMaxPollBackoff = time.Second * 30
	// defaultStartConcurrency bounds how many executions from one batch are
	// started at once. Each one is a control plane round trip plus a
	// runner.Execute that creates Jobs, secrets and config maps - and in
	// standalone mode that shares a process with the control plane.
	defaultStartConcurrency = 4
)

type runner interface {
	Execute(request executionworkertypes.ExecuteRequest) (*executionworkertypes.ExecuteResult, error)
	Pause(executionId string) error
	Resume(executionId string) error
	Abort(executionId string, actor string, reason string) error
	Cancel(executionId string, actor string, reason string) error
}

type workflowStore interface {
	Get(ctx context.Context, environmentId string, name string) (*testkube.TestWorkflow, error)
}

type Client struct {
	OrganizationId     string
	ControlPlaneConfig testworkflowconfig.ControlPlaneConfig

	client           executionv1.TestWorkflowExecutionServiceClient
	logger           *zap.SugaredLogger
	workflowStore    workflowStore
	callOpts         []grpc.CallOption
	callTimeout      time.Duration
	runner           runner
	pollInterval     time.Duration
	maxPollBackoff   time.Duration
	startConcurrency int

	// lastPollOK is the unix nano timestamp of the last successful poll. Read by
	// the readiness check from another goroutine, so it is atomic.
	lastPollOK atomic.Int64

	// sleep exists so tests can observe the backoff ladder. Full jitter makes any
	// assertion on real elapsed time flaky.
	sleep func(time.Duration) <-chan time.Time
}

// Option overrides a Client default.
//
// Every option ignores a non-positive value, so an unset config field falls back
// to the default rather than producing a busy loop or a panic - backoff.New
// panics outright on a negative interval.
type Option func(*Client)

func WithCallTimeout(d time.Duration) Option {
	return func(c *Client) {
		if d > 0 {
			c.callTimeout = d
		}
	}
}

func WithPollInterval(d time.Duration) Option {
	return func(c *Client) {
		if d > 0 {
			c.pollInterval = d
		}
	}
}

func WithMaxPollBackoff(d time.Duration) Option {
	return func(c *Client) {
		if d > 0 {
			c.maxPollBackoff = d
		}
	}
}

func WithStartConcurrency(n int) Option {
	return func(c *Client) {
		if n > 0 {
			c.startConcurrency = n
		}
	}
}

func executionConfigFromStart(start *executionv1.ExecutionStart, organizationId string, rc *executionv1.ExecutionRunningContext) testworkflowconfig.ExecutionConfig {
	return testworkflowconfig.ExecutionConfig{
		Id:              start.GetExecutionId(),
		GroupId:         start.GetGroupId(),
		Name:            start.GetName(),
		Number:          start.GetNumber(),
		ScheduledAt:     start.GetQueuedAt().AsTime(),
		DisableWebhooks: start.GetDisableWebhooks(),
		Tags:            maps.Clone(start.GetTags()),
		Debug:           false,
		OrganizationId:  organizationId,
		EnvironmentId:   start.GetEnvironmentId(),
		ParentIds:       strings.Join(start.AncestorExecutionIds, "/"),
		RunningContext:  runningContextFromProto(rc),
		Lineage:         lineageConfigFromProto(start.GetLineage()),
	}
}

// lineageConfigFromProto carries the execution's lineage across to the pod, so
// that the reserved execution("rerun") reference resolves inside it.
func lineageConfigFromProto(lineage *executionv1.ExecutionLineage) *testworkflowconfig.LineageConfig {
	if lineage == nil {
		return nil
	}
	return &testworkflowconfig.LineageConfig{
		BaseId:  lineage.GetBaseExecutionId(),
		RootId:  lineage.GetRootExecutionId(),
		Attempt: lineage.GetAttempt(),
	}
}

// runningContextFromProto expands the flat proto projection back into the
// legacy TestWorkflowRunningContext shape the toolkit already knows how to
// read (parentActorTypeFromRunningContext dereferences Actor.Type_).
func runningContextFromProto(rc *executionv1.ExecutionRunningContext) *testkube.TestWorkflowRunningContext {
	if rc == nil {
		return nil
	}
	actorType := rc.GetActorType()
	actorName := rc.GetActorName()
	if actorType == "" && actorName == "" {
		return nil
	}
	actor := &testkube.TestWorkflowRunningContextActor{
		Name: actorName,
	}
	if actorType != "" {
		t := testkube.TestWorkflowRunningContextActorType(actorType)
		actor.Type_ = &t
	}
	return &testkube.TestWorkflowRunningContext{Actor: actor}
}

// NewClient creates a client for retrieving updates about executions.
func NewClient(conn grpc.ClientConnInterface, logger *zap.SugaredLogger, r runner, apiToken, organizationId string, tlsEnabled bool, controlPlane testworkflowconfig.ControlPlaneConfig, workflows workflowStore, options ...Option) *Client {
	client := executionv1.NewTestWorkflowExecutionServiceClient(conn)

	opts := []grpc.CallOption{
		// In the event of a transient failure on the server wait for it to come back rather than
		// failing immediately.
		grpc.WaitForReady(true),
	}

	// Standalone deployment does not have an API Token for now
	if apiToken != "" {
		tokenSource := oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: apiToken,
		})
		perRPCCreds := grpc.PerRPCCredentials(oauth.TokenSource{
			TokenSource: tokenSource,
		})
		if !tlsEnabled {
			perRPCCreds = grpc.PerRPCCredentials(grpcutils.InsecureDangerousTokenSource{
				TokenSource: tokenSource,
			})
		}
		opts = append(opts, perRPCCreds)
	}

	c := &Client{
		OrganizationId:     organizationId,
		ControlPlaneConfig: controlPlane,

		client:           client,
		logger:           logger,
		workflowStore:    workflows,
		callOpts:         opts,
		callTimeout:      defaultCallTimeout,
		runner:           r,
		pollInterval:     defaultPollInterval,
		maxPollBackoff:   defaultMaxPollBackoff,
		startConcurrency: defaultStartConcurrency,
		sleep:            time.After,
	}
	for _, opt := range options {
		opt(c)
	}
	// Seed the readiness clock so the runner is not reported stale in the window
	// between construction and the first successful poll.
	c.lastPollOK.Store(time.Now().UnixNano())
	return c
}

// Start begins polling the control plane for updates to executions for this runner and passes
// the instructions to the runner to be implemented.
// The environmentId is not an optional field, setting it to an invalid value will cause the calls
// to set newly started executions as "SCHEDULING" to fail, resulting in duplicate start requests
// to be received. Whilst this is not a severe issue it could cause executions to become "stuck"
// in a queue at the Control Plane awaiting them going live on the runner.
func (c *Client) Start(ctx context.Context, environmentId string) error {
	b := backoff.New(c.maxPollBackoff, c.pollInterval)
	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()
	req := &executionv1.GetExecutionUpdatesRequest{}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			response, err := c.getUpdates(ctx, environmentId, req)
			if err != nil {
				// Duration() advances the attempt counter, so it has to be called
				// exactly once per failure. Calling it again for the log field
				// doubled the exponent on every failure and logged a number that
				// was not the one slept.
				retryAfter := b.Duration()
				c.logger.Warnw("Failed to get execution updates, backing off before retrying.",
					"backoff", retryAfter,
					"error", err)
				// Wait for the backoff, but stay cancellable: without this a
				// shutdown during a long backoff blocks here, and the errgroup
				// waiting on this goroutine blocks with it.
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-c.sleep(retryAfter):
				}
				continue
			}
			// If request succeeds then backoffs can be reset.
			b.Reset()
			c.lastPollOK.Store(time.Now().UnixNano())

			// Run in a separate function to clean up logic here.
			// THIS FUNCTION SHOULD BLOCK UNTIL PROCESSING THE REQUEST IS COMPLETE.
			// If you put this in a separate goroutine then you risk polling faster
			// than you can process responses and introducing race conditions.
			c.executeResponse(ctx, response)
		}
	}
}

// getUpdates performs one poll under its own call deadline.
func (c *Client) getUpdates(ctx context.Context, environmentId string, req *executionv1.GetExecutionUpdatesRequest) (*executionv1.GetExecutionUpdatesResponse, error) {
	// Execute with our own call timeout context to prevent stalling out.
	callCtx, cancel := context.WithTimeout(ctx, c.callTimeout)
	defer cancel()
	// Add metadata to the call.
	// Environment ID should only be sent in some instances so it should be omitted
	// if it is not set to any specific value.
	callCtx = metadata.AppendToOutgoingContext(callCtx, "organization-id", c.OrganizationId)
	if environmentId != "" {
		callCtx = metadata.AppendToOutgoingContext(callCtx, "environment-id", environmentId)
	}
	return c.client.GetExecutionUpdates(callCtx, req, c.callOpts...)
}

// LastSuccessfulPoll reports when the control plane was last reached.
func (c *Client) LastSuccessfulPoll() time.Time {
	return time.Unix(0, c.lastPollOK.Load())
}

// Healthy reports whether the poll loop is still consuming work.
//
// now is passed in rather than read here so the check stays a pure function.
// A stalled loop is the failure this reports: the process stays up and the gRPC
// server keeps answering, so nothing else observes that no execution has been
// picked up in hours.
func (c *Client) Healthy(now time.Time, staleAfter time.Duration) error {
	last := c.LastSuccessfulPoll()
	if age := now.Sub(last); age > staleAfter {
		return fmt.Errorf("no successful execution update poll for %s (last at %s)", age.Truncate(time.Second), last.UTC().Format(time.RFC3339))
	}
	return nil
}

func (c *Client) executeResponse(ctx context.Context, response *executionv1.GetExecutionUpdatesResponse) {
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
				err := c.runner.Cancel(transition.GetExecutionId(), transition.GetActor(), transition.GetReason())
				switch {
				case errors.Is(err, registry.ErrResourceNotFound):
					// Mission failed successfully!
					c.logger.Debugw("Request to cancel execution that does not exist, execution was probably already cancelled but the Control Plane doesn't know.",
						"execution", transition.GetExecutionId())
				case err != nil:
					c.logger.Errorw("Failed to cancel execution",
						"execution", transition.GetExecutionId(),
						"error", err)
				}
			})
		case executionv1.ExecutionState_EXECUTION_STATE_ABORTED:
			wg.Go(func() {
				err := c.runner.Abort(transition.GetExecutionId(), transition.GetActor(), transition.GetReason())
				switch {
				case errors.Is(err, registry.ErrResourceNotFound):
					// Mission failed successfully!
					c.logger.Debugw("Request to abort execution that does not exist, execution was probably already aborted but the Control Plane doesn't know.",
						"execution", transition.GetExecutionId())
				case err != nil:
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

	// Starts fan out under a bounded errgroup. The workflow fetch used to run
	// sequentially and inline on this goroutine, so a batch of N starts cost N
	// round trips before the first execution began - the same stall this change
	// exists to remove, just relocated to the runner.
	limit := c.startConcurrency
	if limit <= 0 {
		// A zero limit gives errgroup a zero-capacity semaphore and Go blocks
		// forever. A Client built as a struct literal has one.
		limit = defaultStartConcurrency
	}
	var starts errgroup.Group
	starts.SetLimit(limit)
	for _, start := range response.GetStart() {
		starts.Go(func() error {
			c.startExecution(ctx, start)
			return nil
		})
	}

	// Wait for everything to finish before returning.
	wg.Wait()
	_ = starts.Wait()
}

// startExecution fetches the workflow for one dispatched execution and hands it
// to the runner, reporting the outcome back to the control plane.
func (c *Client) startExecution(ctx context.Context, start *executionv1.ExecutionStart) {
	// Execute with our own call timeout context to prevent stalling out.
	callCtx, cancel := context.WithTimeout(ctx, c.callTimeout)
	// Add required metadata to the call.
	callCtx = metadata.AppendToOutgoingContext(callCtx,
		"organization-id", c.OrganizationId,
		"environment-id", start.GetEnvironmentId())
	// Grab the full workflow.
	workflowResponse, err := c.client.GetExecutionWorkflow(callCtx, &executionv1.GetExecutionWorkflowRequest{
		ExecutionId:   start.ExecutionId,
		EnvironmentId: start.EnvironmentId,
	}, c.callOpts...)
	cancel()
	if err != nil {
		// We cannot process this request as we do not know about the workflow to be executed.
		c.logger.Errorw("Failed to retrieve workflow for execution, this execution will not be started.",
			"executionId", start.GetExecutionId(),
			"workflow name", start.GetWorkflowName(),
			"error", err)
		// A transient failure is retried by the next poll, which re-sends this
		// start. A permanent one never resolves, so say so: otherwise the
		// execution stays STARTING forever and is re-sent on every poll, taking a
		// slot in every batch behind it.
		if !grpcutils.IsTransient(err) {
			c.declineExecution(ctx, start, testkube.StartReasonUnknown, err)
		}
		return
	}
	// Deserialise the workflow.
	var workflow testworkflowsv1.TestWorkflow
	if err := json.Unmarshal(workflowResponse.GetWorkflow().GetJson(), &workflow); err != nil {
		c.logger.Errorw("Failed to unmarshal workflow for execution, this execution will not be started.",
			"executionId", start.GetExecutionId(),
			"workflow name", start.GetWorkflowName(),
			"error", err)
		// Permanent by construction: the next poll would fail to parse it too.
		c.declineExecution(ctx, start, testkube.StartReasonDefinitionInvalid, err)
		return
	}

	result, err := c.runner.Execute(executionworkertypes.ExecuteRequest{
		Token: start.GetExecutionToken(),
		Runtime: &executionworkertypes.Runtime{
			Variables: start.GetVariableOverrides(),
		},
		Execution:    executionConfigFromStart(start, c.OrganizationId, workflowResponse.GetRunningContext()),
		Workflow:     workflow,
		ControlPlane: c.ControlPlaneConfig,
	})
	if err != nil {
		reason := executionworkertypes.StartReasonOf(err)
		c.logger.Errorw("Failed to start execution.",
			"executionId", start.GetExecutionId(),
			"organizationId", c.OrganizationId,
			"environmentId", start.GetEnvironmentId(),
			"workflowName", start.GetWorkflowName(),
			"reason", reason,
			"error", err)
		c.declineExecution(ctx, start, reason, err)
		return
	}

	if result.Redundant {
		// Execution already existed.
		return
	}

	// Execute with our own call timeout context to prevent stalling out.
	acceptCtx, acceptCancel := context.WithTimeout(ctx, c.callTimeout)
	// Add required metadata to the call.
	acceptCtx = metadata.AppendToOutgoingContext(acceptCtx,
		"organization-id", c.OrganizationId,
		"environment-id", start.GetEnvironmentId())
	// Update the control plane that the execution is awaiting scheduling by Kubernetes.
	_, err = c.client.AcceptExecution(acceptCtx, &executionv1.AcceptExecutionRequest{
		ExecutionId: start.ExecutionId,
		Namespace:   &result.Namespace,
		Signature:   translateSignature(result.Signature),
	}, c.callOpts...)
	acceptCancel()
	if err != nil {
		// The control plane refuses to accept an execution it has already
		// finished - typically one whose dispatch lease expired and was reaped
		// while this setup was still in flight. The resources exist now, so tear
		// them down rather than leaving a Job running for an execution that is
		// recorded as aborted.
		if grpcutils.ErrorCode(err) == codes.FailedPrecondition {
			c.logger.Warnw("Control plane rejected the started execution, aborting it locally.",
				"executionId", start.GetExecutionId(),
				"error", err)
			if abortErr := c.runner.Abort(start.GetExecutionId(), "", "execution already finished in the control plane"); abortErr != nil &&
				!errors.Is(abortErr, registry.ErrResourceNotFound) {
				c.logger.Errorw("Failed to abort a rejected execution",
					"executionId", start.GetExecutionId(),
					"error", abortErr)
			}
			return
		}
		c.logger.Errorw("Failed to set execution scheduling",
			"executionId", start.GetExecutionId(),
			"error", err)
	}
}

// declineExecution reports to the control plane that this execution will not
// start, so that it is failed with a cause the user can see rather than being
// re-offered on every subsequent poll.
func (c *Client) declineExecution(ctx context.Context, start *executionv1.ExecutionStart, reason testkube.StartReason, cause error) {
	// Execute with our own call timeout context to prevent stalling out.
	callCtx, cancel := context.WithTimeout(ctx, c.callTimeout)
	defer cancel()
	// Add required metadata to the call.
	callCtx = metadata.AppendToOutgoingContext(callCtx,
		"organization-id", c.OrganizationId,
		"environment-id", start.GetEnvironmentId())
	// The reason and message let the control plane store the cause on the
	// execution, so the user does not need the runner log.
	if _, err := c.client.DeclineExecution(callCtx, &executionv1.DeclineExecutionRequest{
		ExecutionId: start.ExecutionId,
		Reason:      proto.String(string(reason)),
		Message:     proto.String(cause.Error()),
	}, c.callOpts...); err != nil {
		c.logger.Errorw("Failed to report execution start error.",
			"executionId", start.GetExecutionId(),
			"error", err)
	}
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
