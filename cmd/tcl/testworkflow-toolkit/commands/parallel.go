// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	commontcl "github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/common"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/spawn"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/artifacts"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env/config"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/transfer"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/credentials"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/ui"
)

// ParallelOptions contains all dependencies for parallel execution
type ParallelOptions struct {
	Storage artifacts.InternalArtifactStorage
}

// DefaultParallelOptions creates options with production dependencies
func DefaultParallelOptions(cfg *config.ConfigV2) (*ParallelOptions, error) {
	cloud, err := env.Cloud()
	if err != nil {
		return nil, errors.Wrap(err, "could not create cloud client")
	}

	storage := artifacts.InternalStorageForAgent(
		cloud,
		cfg.Internal().Execution.EnvironmentId,
		cfg.Internal().Execution.Id,
		cfg.Internal().Workflow.Name,
		cfg.Ref(),
	)

	return &ParallelOptions{
		Storage: storage,
	}, nil
}

const (
	// DefaultWorkerTimeout is the default timeout for worker execution.
	// Workers that exceed this timeout are forcefully terminated.
	DefaultWorkerTimeout = 30 * time.Minute

	// UpdatesPerWorker is the expected number of status updates per worker.
	// Used to calculate buffer size for the updates channel to prevent blocking.
	// Typical updates: started, status change, completed = 3 updates.
	UpdatesPerWorker = 3
)

// WorkerRegistry manages worker status and lifecycle.
// Implementations must be thread-safe as multiple goroutines update worker state concurrently.
type WorkerRegistry interface {
	// SetStatus updates the execution status for a worker
	SetStatus(index int64, status *testkube.TestWorkflowStatus)
	// SetAddress stores the pod IP address for a worker
	SetAddress(index int64, address string)
	// Destroy removes all registry entries for a worker
	Destroy(index int64)
	// Count returns the number of registered workers
	Count() int64
	// AllPaused returns true if all registered workers are in paused state
	AllPaused() bool
	// Indexes returns all registered worker indices
	Indexes() []int64
}

// ParallelStatus represents the current status of a parallel worker.
// Serialized as JSON and sent via PrintOutput for UI updates.
type ParallelStatus struct {
	Index       int                              `json:"index"`
	Description string                           `json:"description,omitempty"`
	Current     string                           `json:"current,omitempty"`
	Logs        string                           `json:"logs,omitempty"`
	Status      testkube.TestWorkflowStatus      `json:"status,omitempty"`
	Signature   []testkube.TestWorkflowSignature `json:"signature,omitempty"`
	Result      *testkube.TestWorkflowResult     `json:"result,omitempty"`
}

// AsMap converts ParallelStatus to a map for JSON output.
// Used by the runner package to serialize status updates.
func (s ParallelStatus) AsMap() (v map[string]interface{}) {
	serialized, _ := json.Marshal(s)
	_ = json.Unmarshal(serialized, &v)
	return
}

// Update represents a worker status update sent through the updates channel.
// Used for communication between WorkerExecutor and ResumeOrchestrator.
type Update struct {
	index  int64                        // Worker index
	result *testkube.TestWorkflowResult // Current execution result (nil if no change)
	done   bool                         // True when worker execution completes
	err    error                        // Execution error if any
}

func NewParallelCmd() *cobra.Command {
	var base64Encoded bool

	cmd := &cobra.Command{
		Use:   "parallel <spec>",
		Short: "Run parallel steps",
		Args:  cobra.ExactArgs(1),

		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.LoadConfigV2()
			if err != nil {
				ui.ExitOnError("loading configuration", err)
			}

			err = RunParallel(cmd.Context(), args[0], cfg, base64Encoded)
			if err != nil {
				ui.ExitOnError("parallel execution", err)
			}
			os.Exit(0)
		},
	}

	cmd.Flags().BoolVar(&base64Encoded, "base64", false, "spec is base64 encoded")

	return cmd
}

// ParallelExecutionResult contains the results of parallel execution.
// Used to determine overall success/failure of the parallel command.
type ParallelExecutionResult struct {
	TotalWorkers  int64 // Total number of workers executed
	FailedWorkers int64 // Number of workers that failed
}

// RunParallel executes parallel workers and returns an error if any failures occur.
// This is the main entry point for parallel execution, handling the complete lifecycle:
// 1. Parse and validate the parallel specification
// 2. Calculate the number of workers based on matrix/shards/count
// 3. Build individual worker specifications with resolved expressions
// 4. Execute workers with configured parallelism limits
// 5. Monitor execution and handle failures
//
// The base64Encoded flag indicates the spec was encoded to prevent testworkflow-init
// from prematurely resolving expressions like {{ matrix.* }} or {{ shard.* }}.
// Returns nil if all workers succeed, error if any fail or on configuration errors.
func RunParallel(ctx context.Context, specContent string, cfg *config.ConfigV2, base64Encoded bool) error {
	opts, err := DefaultParallelOptions(cfg)
	if err != nil {
		return err
	}
	return RunParallelWithOptions(ctx, specContent, cfg, base64Encoded, opts)
}

// RunParallelWithOptions executes parallel workers with injected dependencies.
// See RunParallel for base64Encoded parameter explanation.
func RunParallelWithOptions(ctx context.Context, specContent string, cfg *config.ConfigV2, base64Encoded bool, opts *ParallelOptions) error {
	// Load state and credentials once at the beginning
	data.GetState() // Load state to ensure StateMachine is populated
	stateMachine := data.StateMachine
	credentialMachine := credentials.NewCredentialMachine(data.Credentials())

	parallel, params, err := parseAndValidateSpec(specContent, cfg, base64Encoded, stateMachine, credentialMachine)
	if err != nil {
		return err
	}

	if params.Count == 0 {
		fmt.Printf("0 instances requested (combinations=%d, count=%d), skipping\n", params.MatrixCount, params.ShardCount)
		return nil
	}

	parallelism := int64(parallel.Parallelism)
	if parallelism <= 0 {
		parallelism = spawn.DefaultParallelism
	}
	fmt.Println("Parallelism: " + params.String(parallelism))

	workers, transferSrv, err := prepareWorkers(parallel, params, cfg, stateMachine, credentialMachine)
	if err != nil {
		return err
	}

	if err := StartTransferServer(transferSrv); err != nil {
		return errors.Wrap(err, "starting transfer server")
	}

	if len(workers) == 0 {
		fmt.Println("nothing to run")
		return nil
	}

	result, err := executeWorkersWithStorage(ctx, workers, params, parallelism, parallel.FailFast, cfg, opts.Storage, stateMachine, credentialMachine)
	if err != nil {
		return err
	}

	if result.FailedWorkers == 0 {
		fmt.Printf("Successfully finished %d workers.\n", result.TotalWorkers)
	} else {
		fmt.Printf("Failed to finish %d out of %d expected workers.\n", result.FailedWorkers, result.TotalWorkers)
		return fmt.Errorf("%d workers failed", result.FailedWorkers)
	}

	return nil
}

// parseAndValidateSpec parses the input spec and calculates execution parameters.
// Uses a machine without env resolution to preserve {{env.X}} expressions for workers.
// Returns the normalized spec and calculated parameters (worker count, matrix, shards).
func parseAndValidateSpec(specContent string, cfg *config.ConfigV2, base64Encoded bool, stateMachine expressions.Machine, credentialMachine expressions.Machine) (*testworkflowsv1.StepParallel, *commontcl.ParamsSpec, error) {
	parser := &ParallelSpecParser{}
	parallel, err := parser.ParseSpec(specContent, base64Encoded)
	if err != nil {
		return nil, nil, errors.Wrap(err, "parsing parallel spec")
	}

	parser.NormalizeParallelSpec(parallel)

	// Preserve env.* expressions for parallel workers
	baseMachine := spawn.ParallelCreateBaseMachineWithoutEnv(cfg, stateMachine, credentialMachine)

	params, err := commontcl.GetParamsSpec(parallel.Matrix, parallel.Shards, parallel.Count, parallel.MaxCount, baseMachine)
	if err != nil {
		return nil, nil, errors.Wrap(err, "compute matrix and sharding")
	}

	return parallel, params, nil
}

// prepareWorkers initializes transfer server and builds worker specifications.
// Creates a transfer server for file sharing between parent and workers.
// Uses expression machine without env resolution to preserve environment expressions.
// Returns worker specs and initialized transfer server.
func prepareWorkers(parallel *testworkflowsv1.StepParallel, params *commontcl.ParamsSpec, cfg *config.ConfigV2, stateMachine expressions.Machine, credentialMachine expressions.Machine) ([]WorkerSpec, transfer.Server, error) {
	transferSrv := transfer.NewServer(constants.DefaultTransferDirPath, cfg.IP(), constants.DefaultTransferPort)

	// Preserve env.* expressions for parallel workers
	baseMachine := spawn.ParallelCreateBaseMachineWithoutEnv(cfg, stateMachine, credentialMachine)

	builder := NewParallelWorkersBuilder(transferSrv, baseMachine, params, cfg)
	workers, err := builder.BuildWorkerSpecs(parallel)
	if err != nil {
		return nil, nil, errors.Wrap(err, "building worker specs")
	}

	return workers, transferSrv, nil
}

// ParallelSpecParser handles parsing of parallel specifications
type ParallelSpecParser struct{}

// ParseSpec parses the parallel spec from command arguments.
// Supports two input formats:
//   - Base64 encoded JSON (--base64 flag): Used by the processor to prevent testworkflow-init
//     from prematurely resolving expressions like {{ matrix.value }} or {{ shard.index }}
//   - Raw JSON (legacy): Direct JSON as argument (kept for backward compatibility)
//
// Returns parsed StepParallel or error if parsing fails.
func (p *ParallelSpecParser) ParseSpec(specContent string, base64Encoded bool) (*testworkflowsv1.StepParallel, error) {
	if specContent == "" {
		return nil, errors.New("no spec provided")
	}

	var parallel testworkflowsv1.StepParallel
	if base64Encoded {
		// Decode spec that was base64-encoded to hide expressions from testworkflow-init
		err := expressionstcl.DecodeBase64JSON(specContent, &parallel)
		if err != nil {
			return nil, errors.Wrap(err, "decoding parallel spec")
		}
	} else {
		if err := json.Unmarshal([]byte(specContent), &parallel); err != nil {
			return nil, errors.Wrap(err, "parsing parallel spec")
		}
	}

	return &parallel, nil
}

// NormalizeParallelSpec normalizes the parallel spec by injecting short syntax.
// Transforms StepOperations/StepControl at the parallel level into the first step.
// Also ensures required fields have default values (Content, Pod, ServiceAccount).
// Preserves StepExecuteStrategy (Count, MaxCount, Matrix, Shards) as these define execution parameters.
func (p *ParallelSpecParser) NormalizeParallelSpec(parallel *testworkflowsv1.StepParallel) {
	// Move top-level operations into a first step to maintain execution order
	// This allows users to specify operations directly on the parallel block
	// which will execute before any explicit steps
	if !reflect.ValueOf(parallel.StepControl).IsZero() || !reflect.ValueOf(parallel.StepOperations).IsZero() {
		parallel.Steps = append([]testworkflowsv1.Step{{
			StepControl:    parallel.StepControl,
			StepOperations: parallel.StepOperations,
		}}, parallel.Steps...)
		parallel.StepControl = testworkflowsv1.StepControl{}
		parallel.StepOperations = testworkflowsv1.StepOperations{}
	}

	// StepExecuteStrategy (Count, MaxCount, Matrix, Shards) contains core parallel execution parameters and must be preserved
	if parallel.Content == nil {
		parallel.Content = &testworkflowsv1.Content{}
	}

	if parallel.Pod == nil {
		parallel.Pod = &testworkflowsv1.PodConfig{}
	}
	if parallel.Pod.ServiceAccountName == "" {
		// Use templated expression that will be resolved to the actual service account
		parallel.Pod.ServiceAccountName = "{{internal.serviceaccount.default}}"
	}
}

// WorkerSpec contains specifications for a single parallel worker.
// Created by ParallelWorkersBuilder with all expressions resolved.
type WorkerSpec struct {
	Index        int64                            // Zero-based worker index
	Spec         testworkflowsv1.TestWorkflowSpec // Complete workflow spec for this worker
	Namespace    string                           // Kubernetes namespace for execution
	Description  string                           // Human-readable description with resolved expressions
	LogCondition *string                          // Condition for log collection (always/never/failed)
}

// ParallelWorkersBuilder builds worker specifications
type ParallelWorkersBuilder struct {
	transferSrv transfer.Server
	baseMachine expressions.Machine
	paramsSpec  *commontcl.ParamsSpec
	cfg         *config.ConfigV2
}

// NewParallelWorkersBuilder creates a new workers builder
func NewParallelWorkersBuilder(transferSrv transfer.Server, baseMachine expressions.Machine, paramsSpec *commontcl.ParamsSpec, cfg *config.ConfigV2) *ParallelWorkersBuilder {
	return &ParallelWorkersBuilder{
		transferSrv: transferSrv,
		baseMachine: baseMachine,
		paramsSpec:  paramsSpec,
		cfg:         cfg,
	}
}

// BuildWorkerSpecs builds specifications for all parallel workers.
// Creates a WorkerSpec for each worker instance with:
// - Resolved expressions (index, matrix, shard values)
// - Configured file transfers (tarballs)
// - Proper namespace assignment
// - Log collection conditions
// Returns slice of WorkerSpec or error if building fails.
func (b *ParallelWorkersBuilder) BuildWorkerSpecs(parallel *testworkflowsv1.StepParallel) ([]WorkerSpec, error) {
	specs := make([]WorkerSpec, b.paramsSpec.Count)

	for i := int64(0); i < b.paramsSpec.Count; i++ {
		spec, err := b.buildSingleWorkerSpec(parallel, i)
		if err != nil {
			return nil, errors.Wrapf(err, "building worker %d", i)
		}
		specs[i] = spec
	}

	return specs, nil
}

// buildSingleWorkerSpec creates specification for a single worker instance.
// Resolves all expressions using the worker's index and matrix/shard values.
// Processes file transfers by creating tarballs and adding to content.
// Handles fetch operations by appending collection steps.
// Preserves log conditions for conditional log collection.
func (b *ParallelWorkersBuilder) buildSingleWorkerSpec(parallel *testworkflowsv1.StepParallel, index int64) (WorkerSpec, error) {
	machines := []expressions.Machine{b.baseMachine, b.paramsSpec.MachineAt(index)}

	var logCondition *string
	if parallel.Logs != nil {
		// Preserve log condition expression for later evaluation
		logCondition = common.Ptr(*parallel.Logs)
	}

	spec := parallel.DeepCopy()
	if err := expressions.Simplify(&spec, machines...); err != nil {
		return WorkerSpec{}, err
	}

	tarballs, err := spawn.ProcessTransfer(b.transferSrv, spec.Transfer, machines...)
	if err != nil {
		return WorkerSpec{}, errors.Wrap(err, "processing transfer")
	}
	spec.Content.Tarball = append(spec.Content.Tarball, tarballs...)

	fetchStep, err := spawn.ParallelProcessFetch(b.cfg, b.transferSrv, spec.Fetch, machines...)
	if err != nil {
		return WorkerSpec{}, errors.Wrap(err, "processing fetch")
	}
	if fetchStep != nil {
		// Append fetch operations as 'after' steps to collect outputs after main execution
		spec.After = append(spec.After, *fetchStep)
	}

	namespace := b.cfg.Namespace()
	if spec.Job != nil && spec.Job.Namespace != "" {
		namespace = spec.Job.Namespace
	}

	return WorkerSpec{
		Index:        index,
		Spec:         *spec.NewTestWorkflowSpec(),
		Namespace:    namespace,
		Description:  spec.Description,
		LogCondition: logCondition,
	}, nil
}

// WorkerExecutor handles execution of a single worker
type WorkerExecutor struct {
	storage         artifacts.InternalArtifactStorage
	registry        WorkerRegistry
	updates         chan<- Update
	fullBaseMachine expressions.Machine
	cfg             *config.ConfigV2
}

// NewWorkerExecutor creates a new worker executor
func NewWorkerExecutor(storage artifacts.InternalArtifactStorage, registry WorkerRegistry, updates chan<- Update, fullBaseMachine expressions.Machine, cfg *config.ConfigV2) *WorkerExecutor {
	return &WorkerExecutor{
		storage:         storage,
		registry:        registry,
		updates:         updates,
		fullBaseMachine: fullBaseMachine,
		cfg:             cfg,
	}
}

// ExecuteWorker executes a single worker.
// Handles the complete worker lifecycle:
// 1. Builds execution configuration with proper expression resolution
// 2. Registers worker in the registry
// 3. Deploys worker via ExecutionWorker client
// 4. Monitors execution status
// 5. Handles cleanup (logs, resources)
// Returns true if worker passed, false if failed, and error for execution issues.
// Uses non-blocking channel operations to prevent deadlocks.
func (e *WorkerExecutor) ExecuteWorker(ctx context.Context, worker WorkerSpec, paramsSpec *commontcl.ParamsSpec) (bool, error) {
	log := spawn.ParallelCreateLogger("worker", worker.Description, worker.Index, paramsSpec.Count)

	cfg := *e.cfg.Internal()
	cfg.Resource = spawn.ParallelCreateResourceConfig(e.cfg, e.cfg.Ref()+"-", worker.Index)
	cfg.Worker.Namespace = worker.Namespace
	// Build expression machine with full context hierarchy:
	// resource -> worker -> base -> PVC -> params (index/matrix/shard)
	machine := expressions.CombinedMachines(
		testworkflowconfig.CreateResourceMachine(&cfg.Resource),
		testworkflowconfig.CreateWorkerMachine(&cfg.Worker),
		e.fullBaseMachine,
		testworkflowconfig.CreatePvcMachine(cfg.Execution.PvcNames),
		paramsSpec.MachineAt(worker.Index),
	)

	// Apply final expression resolution with all context (resource, worker, PVC, params)
	_ = expressions.Simplify(&worker.Spec, machine)

	// Register worker with nil status to indicate it's queued but not yet started
	e.registry.SetStatus(worker.Index, nil)

	// Non-blocking update: if context cancelled, cleanup and exit
	// Otherwise send initial status update
	select {
	case <-ctx.Done():
		e.registry.Destroy(worker.Index)
		return false, ctx.Err()
	case e.updates <- Update{index: worker.Index}:
	}

	// Deploy the resource with timeout
	scheduledAt := time.Now()

	execCtx, execCancel := context.WithTimeout(ctx, DefaultWorkerTimeout)
	defer execCancel()

	result, err := spawn.ParallelExecutionWorker(e.cfg).Execute(execCtx, executionworkertypes.ExecuteRequest{
		ResourceId:   cfg.Resource.Id,
		Execution:    cfg.Execution,
		Workflow:     testworkflowsv1.TestWorkflow{ObjectMeta: metav1.ObjectMeta{Name: cfg.Workflow.Name, Labels: cfg.Workflow.Labels}, Spec: worker.Spec},
		ScheduledAt:  &scheduledAt,
		ControlPlane: cfg.ControlPlane,
		// Each worker gets unique artifact path based on its index
		ArtifactsPathPrefix: spawn.ParallelCreateResourceConfig(e.cfg, "", worker.Index).FsPrefix,
	})
	if err != nil {
		log("error", "failed to prepare resources", err.Error())
		e.registry.Destroy(worker.Index)
		return false, err
	}

	return e.monitorWorkerExecution(ctx, worker, *result, cfg, machine, paramsSpec, log)
}

// monitorWorkerExecution monitors a worker's execution and handles status updates.
// Watches for status notifications from the execution worker.
// Sends non-blocking updates to prevent channel congestion.
// Ensures cleanup happens via defer even if monitoring fails.
// Returns true if worker passed, false otherwise.
func (e *WorkerExecutor) monitorWorkerExecution(ctx context.Context, worker WorkerSpec, result executionworkertypes.ExecuteResult, cfg testworkflowconfig.InternalConfig, machine expressions.Machine, paramsSpec *commontcl.ParamsSpec, log func(...string)) (bool, error) {
	var lastResult testkube.TestWorkflowResult

	defer func() {
		e.handleWorkerCleanup(ctx, worker, lastResult, cfg, machine, paramsSpec, log)
	}()

	instructions.PrintOutput(e.cfg.Ref(), "parallel", ParallelStatus{Index: int(worker.Index), Signature: result.Signature})

	monitorCtx, monitorCancel := context.WithCancel(ctx)
	defer monitorCancel()

	notifications := spawn.ParallelExecutionWorker(e.cfg).StatusNotifications(monitorCtx, cfg.Resource.Id, executionworkertypes.StatusNotificationsOptions{
		Hints: executionworkertypes.Hints{
			Namespace:   result.Namespace,
			Signature:   result.Signature,
			ScheduledAt: &result.ScheduledAt,
		},
	})

	if notifications.Err() != nil {
		log("error", "failed to connect to the parallel worker", notifications.Err().Error())
		return false, notifications.Err()
	}

	log("created")

	return e.processWorkerNotifications(notifications, worker, result.Signature, log, &lastResult)
}

// processWorkerNotifications processes status updates from a worker.
// Tracks node assignment, IP allocation, and execution status changes.
// Uses non-blocking channel sends to avoid deadlocks during status updates.
// Handles abnormal termination by setting ABORTED status.
// Returns worker success status and any notification errors.
func (e *WorkerExecutor) processWorkerNotifications(notifications executionworkertypes.StatusNotificationsWatcher, worker WorkerSpec, signature []testkube.TestWorkflowSignature, log func(...string), lastResult *testkube.TestWorkflowResult) (bool, error) {
	prevStatus := testkube.QUEUED_TestWorkflowStatus
	prevStep := ""
	scheduled := false
	ipAssigned := false

	for v := range notifications.Channel() {
		if !scheduled && v.NodeName != "" {
			scheduled = true
			log(fmt.Sprintf("assigned to %s node", ui.LightBlue(v.NodeName)))
		}

		if !ipAssigned && v.PodIp != "" {
			ipAssigned = true
			e.registry.SetAddress(worker.Index, v.PodIp)
		}

		if v.Result != nil {
			*lastResult = *v.Result
			status := testkube.QUEUED_TestWorkflowStatus
			if lastResult.Status != nil {
				status = *lastResult.Status
			}
			// Get current step reference from the workflow signature
			step := lastResult.Current(signature)

			if status != prevStatus || step != prevStep {
				if status != prevStatus {
					log(string(status))
				}
				// Try to send update, but don't block if channel is full
				// This prevents deadlock if orchestrator is slow to process
				select {
				case e.updates <- Update{index: worker.Index, result: v.Result}:
				default:
					log("warning", "skipping status update due to full channel")
				}
				prevStep = step
				prevStatus = status

				if lastResult.IsFinished() {
					instructions.PrintOutput(e.cfg.Ref(), "parallel", ParallelStatus{Index: int(worker.Index), Status: status, Result: v.Result})
					return v.Result.IsPassed(), nil
				}
				instructions.PrintOutput(e.cfg.Ref(), "parallel", ParallelStatus{Index: int(worker.Index), Status: status, Current: step})
			}
		}
	}

	if notifications.Err() != nil {
		log("error", notifications.Err().Error())
		return false, notifications.Err()
	}

	// Fallback for abnormal termination
	log("could not determine status of the worker - aborting")
	lastResult.Status = common.Ptr(testkube.ABORTED_TestWorkflowStatus)
	if lastResult.FinishedAt.IsZero() {
		// Set finish time for aborted workers that didn't report completion
		lastResult.FinishedAt = time.Now().UTC()
	}
	instructions.PrintOutput(e.cfg.Ref(), "parallel", ParallelStatus{Index: int(worker.Index), Status: testkube.ABORTED_TestWorkflowStatus, Result: lastResult})
	select {
	case e.updates <- Update{index: worker.Index, result: lastResult.Clone()}:
		// Update sent successfully
	default:
		log("error", "could not send final abort status")
	}

	return false, nil
}

// handleWorkerCleanup performs cleanup operations for a completed worker.
// Evaluates log condition to determine if logs should be saved.
// Saves logs to artifact storage if condition is met.
// If the execution context was cancelled (e.g. fail-fast), aborts the worker
// (patches termination annotations + destroys K8s resources) instead of just destroying.
// Uses a separate cleanup context so cleanup always runs even when execution is cancelled.
// Always executes via defer to ensure cleanup even on failures.
func (e *WorkerExecutor) handleWorkerCleanup(ctx context.Context, worker WorkerSpec, lastResult testkube.TestWorkflowResult, cfg testworkflowconfig.InternalConfig, machine expressions.Machine, paramsSpec *commontcl.ParamsSpec, log func(...string)) {
	// Use a separate context for cleanup - the execution context may be cancelled
	// by fail-fast but we still need to save logs and clean up K8s resources
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cleanupCancel()

	cancelled := ctx.Err() != nil

	// Default behavior: save logs when no condition specified
	// Otherwise evaluate the condition expression (e.g., "failed", "always", "never")
	shouldSaveLogs := worker.LogCondition == nil
	if !shouldSaveLogs {
		var err error
		shouldSaveLogs, err = spawn.EvalLogCondition(*worker.LogCondition, lastResult, machine, e.fullBaseMachine, paramsSpec.MachineAt(worker.Index))
		if err != nil {
			log("warning", "log condition evaluation failed", err.Error())
		}
	}

	if shouldSaveLogs {
		logsFilePath, err := spawn.ParallelSaveLogs(e.cfg, cleanupCtx, e.storage, worker.Namespace, cfg.Resource.Id, "", worker.Index)
		if err == nil {
			instructions.PrintOutput(e.cfg.Ref(), "parallel", ParallelStatus{Index: int(worker.Index), Logs: e.storage.FullPath(logsFilePath)})
			log("saved logs")
		} else {
			log("warning", "problem saving the logs", err.Error())
		}
	}

	// If cancelled by fail-fast, abort the worker (patches termination annotations + destroys)
	// Otherwise just destroy resources normally
	var err error
	if cancelled {
		err = spawn.ParallelExecutionWorker(e.cfg).Abort(cleanupCtx, cfg.Resource.Id, executionworkertypes.DestroyOptions{
			Namespace: worker.Namespace,
		})
		if err == nil {
			log("aborted")
		} else {
			log("warning", "problem aborting worker", err.Error())
		}
	} else {
		err = spawn.ParallelExecutionWorker(e.cfg).Destroy(cleanupCtx, cfg.Resource.Id, executionworkertypes.DestroyOptions{
			Namespace: worker.Namespace,
		})
		if err == nil {
			log("cleaned resources")
		} else {
			log("warning", "problem cleaning up resources", err.Error())
		}
	}

	select {
	case e.updates <- Update{index: worker.Index, done: true, err: err}:
	default:
		log("warning", "could not send final update for worker", fmt.Sprintf("index=%d", worker.Index))
	}
}

// ResumeOrchestrator handles resuming paused workers
type ResumeOrchestrator struct {
	registry     WorkerRegistry
	updates      <-chan Update
	namespaces   []string
	descriptions []string
	cfg          *config.ConfigV2
}

// NewResumeOrchestrator creates a new resume orchestrator
func NewResumeOrchestrator(registry WorkerRegistry, updates <-chan Update, namespaces []string, descriptions []string, cfg *config.ConfigV2) *ResumeOrchestrator {
	return &ResumeOrchestrator{
		registry:     registry,
		updates:      updates,
		namespaces:   namespaces,
		descriptions: descriptions,
		cfg:          cfg,
	}
}

// Start starts the resume orchestration.
// Monitors worker status updates and coordinates resume operations.
// When all workers reach paused state, triggers simultaneous resume.
// Exits cleanly on context cancellation or channel closure.
// This ensures synchronized execution phases across all workers.
func (o *ResumeOrchestrator) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case update, ok := <-o.updates:
			if !ok {
				return
			}

			if update.result != nil {
				o.registry.SetStatus(update.index, update.result.Status)
			}

			// Delete obsolete data
			if update.done || update.err != nil {
				o.registry.Destroy(update.index)
			}

			// Trigger synchronized resume when all active workers reach pause state
			if o.registry.Count() > 0 && o.registry.AllPaused() {
				o.resumeAllWorkers(ctx)
			}
		}
	}
}

// resumeAllWorkers resumes all paused workers simultaneously.
// Called when all workers reach paused state to ensure synchronized execution.
// Handles resume failures by aborting failed workers.
// Checks context cancellation to avoid operations after shutdown.
func (o *ResumeOrchestrator) resumeAllWorkers(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	fmt.Println("resuming all workers")
	ids := common.MapSlice(o.registry.Indexes(), func(index int64) string {
		return spawn.ParallelGetResourceId(o.cfg, o.cfg.Ref()+"-", index)
	})

	errs := spawn.ParallelExecutionWorker(o.cfg).ResumeMany(ctx, ids, executionworkertypes.ControlOptions{})
	for _, err := range errs {
		if err.Id == "" {
			fmt.Printf("warn: %s\n", err.Error)
		} else {
			// Extract worker index from resource ID to identify which worker failed
			_, index := spawn.GetServiceByResourceId(err.Id)
			spawn.ParallelCreateLogger("worker", o.descriptions[index], index, int64(len(o.descriptions)))("warning", "failed to resume", err.Error.Error())

			select {
			case <-ctx.Done():
				return
			default:
				_ = spawn.ParallelExecutionWorker(o.cfg).Abort(ctx, err.Id, executionworkertypes.DestroyOptions{
					Namespace: o.namespaces[index],
				})
			}
		}
	}
}

// executeWorkersWithStorage is like executeWorkers but accepts storage as parameter
func executeWorkersWithStorage(ctx context.Context, workers []WorkerSpec, params *commontcl.ParamsSpec, parallelism int64, failFast bool, cfg *config.ConfigV2, storage artifacts.InternalArtifactStorage, stateMachine expressions.Machine, credentialMachine expressions.Machine) (*ParallelExecutionResult, error) {
	for _, worker := range workers {
		instructions.PrintOutput(cfg.Ref(), "parallel", ParallelStatus{
			Index:       int(worker.Index),
			Description: worker.Description,
		})
	}

	execCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	updates := make(chan Update, len(workers)*UpdatesPerWorker)
	registry := spawn.NewRegistry()

	namespaces := make([]string, len(workers))
	descriptions := make([]string, len(workers))
	for i, worker := range workers {
		namespaces[i] = worker.Namespace
		descriptions[i] = worker.Description
	}

	resumeOrchestrator := NewResumeOrchestrator(registry, updates, namespaces, descriptions, cfg)
	go resumeOrchestrator.Start(execCtx)

	fullBaseMachine := spawn.ParallelCreateBaseMachine(cfg, stateMachine, credentialMachine)
	executor := NewWorkerExecutor(storage, registry, updates, fullBaseMachine, cfg)

	// ExecuteParallel callback - matches worker by index and namespace
	// Returns true if worker passed, false if failed
	run := func(index int64, namespace string, spec *testworkflowsv1.TestWorkflowSpec) bool {
		var worker WorkerSpec
		for _, w := range workers {
			if w.Index == index && w.Namespace == namespace {
				worker = w
				worker.Spec = *spec
				break
			}
		}

		passed, err := executor.ExecuteWorker(execCtx, worker, params)
		if err != nil {
			fmt.Printf("%d: error: %v\n", index, err)
		}
		if !passed && failFast {
			cancel()
		}
		return passed
	}

	// Extract specs and namespaces for ExecuteParallel API
	// Maintains order correspondence with workers slice
	specs := make([]testworkflowsv1.TestWorkflowSpec, len(workers))
	workerNamespaces := make([]string, len(workers))
	for i, worker := range workers {
		specs[i] = worker.Spec
		workerNamespaces[i] = worker.Namespace
	}

	// Execute workers with parallelism limit - blocks until all complete
	failed := spawn.ExecuteParallel(execCtx, run, specs, workerNamespaces, parallelism)

	// Signal orchestrator to exit by closing updates channel
	close(updates)

	return &ParallelExecutionResult{
		TotalWorkers:  params.Count,
		FailedWorkers: failed,
	}, nil
}

// StartTransferServer starts the transfer server if needed.
// Only starts if there are files to transfer or fetch requests.
// The server enables efficient file sharing between parent and workers.
// Returns error if server fails to start.
func StartTransferServer(transferSrv transfer.Server) error {
	if transferSrv.Count() == 0 && transferSrv.RequestsCount() == 0 {
		return nil
	}

	infos := make([]string, 0)
	if transferSrv.Count() > 0 {
		infos = append(infos, fmt.Sprintf("sending %d tarballs", transferSrv.Count()))
	}
	if transferSrv.RequestsCount() > 0 {
		infos = append(infos, fmt.Sprintf("fetching %d requests", transferSrv.RequestsCount()))
	}

	fmt.Printf("Starting transfer server for %s...\n", strings.Join(infos, " and "))
	if _, err := transferSrv.Listen(); err != nil {
		return errors.Wrap(err, "failed to start transfer server")
	}
	fmt.Printf("Transfer server started.\n")
	return nil
}
