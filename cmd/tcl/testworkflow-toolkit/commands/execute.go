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
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	commontcl "github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/common"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/execute"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/spawn"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	toolkitcommon "github.com/kubeshop/testkube/cmd/testworkflow-toolkit/common"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env/config"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/transfer"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/credentials"
	"github.com/kubeshop/testkube/pkg/executiondata"
	"github.com/kubeshop/testkube/pkg/expressions"
	commonmapper "github.com/kubeshop/testkube/pkg/mapper/common"
	"github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/ui"
)

const (
	CreateExecutionRetryOnFailureMaxAttempts = 10
	CreateExecutionRetryOnFailureBaseDelay   = 500 * time.Millisecond

	GetExecutionRetryOnFailureMaxAttempts = 30
	GetExecutionRetryOnFailureDelay       = 500 * time.Millisecond

	ExecutionResultPollingTime = 200 * time.Millisecond
)

type testWorkflowExecutionDetails struct {
	Id               string `json:"id"`
	Name             string `json:"name"`
	TestWorkflowName string `json:"testWorkflowName"`
	Description      string `json:"description,omitempty"`
}

type executionResult struct {
	Id     string `json:"id"`
	Status string `json:"status"`
}

// executionRecorder tracks the test workflows this step schedules, so their outputs
// can be read back with the execution() expression - by later entries of the same
// step, and by later steps of the workflow.
//
// Every change is published as an output instruction, which puts the group into the
// workflow state (for the following steps) and into the execution record (for the API).
type executionRecorder struct {
	mu       sync.Mutex
	registry *executiondata.Registry
	// claimed tracks the references this step has taken over from earlier steps
	claimed map[string]struct{}
}

func newExecutionRecorder(registry *executiondata.Registry) *executionRecorder {
	return &executionRecorder{registry: registry, claimed: map[string]struct{}{}}
}

// schedule records a freshly scheduled execution and assigns it a position within
// its group, so that fan-out instances can be addressed as execution("name", index).
func (r *executionRecorder) schedule(alias, workflowName string, exec testkube.TestWorkflowExecution) executiondata.Execution {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry := executiondata.Execution{
		Id:       exec.Id,
		Name:     exec.Name,
		Workflow: workflowName,
		Alias:    alias,
	}

	// An earlier step may have run the same workflow. This step redefines what the
	// reference means, so drop the stale group before indexing into it - otherwise
	// execution("name") would keep pointing at the previous step's run.
	if _, ok := r.claimed[entry.Key()]; !ok {
		r.claimed[entry.Key()] = struct{}{}
		r.registry.Reset(entry.Key())
	}
	entry.Index = int64(len(r.registry.Group(entry.Key())))
	if exec.Result != nil && exec.Result.Status != nil {
		entry.Status = string(*exec.Result.Status)
	}
	r.store(entry)
	return entry
}

// complete refreshes an entry from the finished execution.
func (r *executionRecorder) complete(entry executiondata.Execution, exec testkube.TestWorkflowExecution) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry.Outputs = executiondata.OutputsOf(&exec)
	entry.ErrorMessage, entry.StepErrors = executiondata.ErrorsOf(&exec)
	entry.StepAttempts = executiondata.AttemptsOf(&exec)
	entry.ErrorReason, entry.StepReasons = executiondata.ReasonsOf(&exec)
	if exec.Result != nil && exec.Result.Status != nil {
		entry.Status = string(*exec.Result.Status)
	}
	r.store(entry)
}

// store must be called with the lock held.
func (r *executionRecorder) store(entry executiondata.Execution) {
	r.registry.Add(entry)
	instructions.PrintOutput(config.Ref(), executiondata.ExecutionInstructionName(entry.Key()), r.registry.Group(entry.Key()))
}

// workflowExecutionRequest is a single test workflow execution the step will run.
// The spec is kept unresolved until the operation starts, so that its configuration
// may reference executions scheduled earlier by the same step.
type workflowExecutionRequest struct {
	spec     *testworkflowsv1.StepExecuteWorkflow
	machines []expressions.Machine
	alias    string
	async    bool
	recorder *executionRecorder
}

// executionOutcome is the result of one execution that the step started or tried to start.
type executionOutcome struct {
	// name is the execution name, or the workflow name when the step did not schedule an execution.
	name string
	// err is nil when the execution passed.
	err error
}

// operationResult is the result of one entry of the step.
type operationResult struct {
	outcomes []executionOutcome
	// err is a failure that is not about one execution, for example a failure to fetch the artifacts.
	err error
}

// failureSummary returns one line that names the failed executions and then the other failures.
// It returns an empty string when there is no failure.
func failureSummary(results []operationResult) string {
	total := 0
	var failed, others []string
	for _, r := range results {
		for _, o := range r.outcomes {
			total++
			if o.err != nil {
				failed = append(failed, fmt.Sprintf("%s (%s)", o.name, o.err.Error()))
			}
		}
		if r.err != nil {
			others = append(others, r.err.Error())
		}
	}

	var parts []string
	if len(failed) > 0 {
		parts = append(parts, fmt.Sprintf("%d of %d executions failed: %s", len(failed), total, strings.Join(failed, ", ")))
	}
	return strings.Join(append(parts, others...), "; ")
}

func buildWorkflowExecution(req workflowExecutionRequest) func() operationResult {
	return func() operationResult {
		workflow := *req.spec.DeepCopy()
		if err := expressions.Finalize(&workflow, req.machines...); err != nil {
			ui.Errf("failed to compute execution: %s: %s", req.spec.Name, err.Error())
			return notScheduled(req.spec.Name, errors.Wrap(err, "computing execution"))
		}

		// An output another workflow withheld resolves to a marker instead of the value
		// it was meant to carry. Scheduling this execution would configure it with the
		// marker, so stop while the cause is still visible.
		if markers := executiondata.WithheldMarkersIn(&workflow); len(markers) > 0 {
			err := executiondata.WithheldError("this execution", markers)
			ui.Errf("failed to compute execution: %s: %s", req.spec.Name, err.Error())
			return notScheduled(req.spec.Name, errors.Wrap(err, "computing execution"))
		}

		// A base that was configured but came out empty must not quietly become a
		// first run. The scheduled execution would resolve no "rerun" reference, so
		// a workflow branching on lineage takes its original branch and the run
		// looks like a working configuration instead of a broken one - which is
		// exactly how this is hard to notice from the outside.
		if req.spec.BaseExecutionId != "" && workflow.BaseExecutionId == "" {
			err := errors.Errorf("baseExecutionId %q resolved to nothing", req.spec.BaseExecutionId)
			ui.Errf("failed to compute execution: %s: %s", req.spec.Name, err.Error())
			return notScheduled(req.spec.Name, errors.Wrap(err, "computing execution"))
		}
		// Say it out loud, so a run that silently schedules a first run is
		// distinguishable from one that never carried a base at all.
		if workflow.BaseExecutionId != "" {
			fmt.Printf("%s • rerun of %s\n", ui.LightCyan(workflow.Name), workflow.BaseExecutionId)
		}

		async := req.async
		tags := config.ExecutionTags()
		target := common.MapPtr(workflow.Target, commonmapper.MapTargetKubeToAPI)

		// Schedule execution
		var execs []testkube.TestWorkflowExecution
		var err error
		for i := 0; i < CreateExecutionRetryOnFailureMaxAttempts; i++ {
			execs, err = execute.ExecuteTestWorkflow(workflow.Name, testkube.TestWorkflowExecutionRequest{
				Name:            workflow.ExecutionName,
				Config:          testworkflows.MapConfigValueKubeToAPI(workflow.Config),
				DisableWebhooks: config.ExecutionDisableWebhooks(),
				Tags:            tags,
				Target:          target,
			}, workflow.BaseExecutionId)
			if err == nil {
				break
			}
			if i+1 < CreateExecutionRetryOnFailureMaxAttempts {
				nextDelay := time.Duration(i+1) * CreateExecutionRetryOnFailureBaseDelay
				ui.Errf("failed to execute test workflow: retrying in %s (attempt %d/%d): %s: %s", nextDelay.String(), i+2, CreateExecutionRetryOnFailureMaxAttempts, workflow.Name, err.Error())
				time.Sleep(nextDelay)
			}
		}
		if err != nil {
			ui.Errf("failed to execute test workflow: %s: %s", workflow.Name, err.Error())
			return notScheduled(workflow.Name, errors.Wrap(err, "scheduling"))
		}

		// Print information about scheduled execution
		entries := make([]executiondata.Execution, len(execs))
		outcomes := make([]executionOutcome, len(execs))
		for i, exec := range execs {
			instructions.PrintOutput(config.Ref(), "testworkflow-start", &testWorkflowExecutionDetails{
				Id:               exec.Id,
				Name:             exec.Name,
				TestWorkflowName: exec.Workflow.Name,
				Description:      workflow.Description,
			})

			// Register it, so the following executions and steps may read its data
			entries[i] = req.recorder.schedule(req.alias, exec.Workflow.Name, exec)
			outcomes[i] = executionOutcome{name: exec.Name}

			description := ""
			if workflow.Description != "" {
				description = fmt.Sprintf(": %s", workflow.Description)
			}
			fmt.Printf("%s%s • scheduled %s\n", ui.LightCyan(exec.Name), description, ui.DarkGray("("+exec.Id+")"))
		}

		if async {
			if len(workflow.Fetch) > 0 {
				ui.Warn("skipping 'fetch': artifacts are only available after an execution finishes, and this step is asynchronous")
			}
			return operationResult{outcomes: outcomes}
		}

		// Monitor. Each goroutine writes only its own item of outcomes.
		var wg sync.WaitGroup
		wg.Add(len(execs))
		for i := range execs {
			go func(index int, entry executiondata.Execution, exec testkube.TestWorkflowExecution) {
				defer wg.Done()
				prevStatus := testkube.QUEUED_TestWorkflowStatus
				var gErr error
			loop:
				for {
					// TODO: Consider real-time Notifications without logs instead
					time.Sleep(ExecutionResultPollingTime)

					// Use go routine error variable
					for i := 0; i < GetExecutionRetryOnFailureMaxAttempts; i++ {
						var next *testkube.TestWorkflowExecution
						next, gErr = execute.GetExecution(exec.Id)
						if gErr == nil {
							exec = *next
							break
						}

						if i+1 < GetExecutionRetryOnFailureMaxAttempts {
							ui.Errf("error while getting execution result: retrying in %s (attempt %d/%d): %s: %s", GetExecutionRetryOnFailureDelay.String(), i+2, GetExecutionRetryOnFailureMaxAttempts, ui.LightCyan(exec.Name), gErr.Error())
							time.Sleep(GetExecutionRetryOnFailureDelay)
						}
					}

					// Check go routine error
					if gErr != nil {
						ui.Errf("error while getting execution result: %s: %s", ui.LightCyan(exec.Name), gErr.Error())
						outcomes[index].err = errors.Wrap(gErr, "getting the execution result")
						return
					}

					if exec.Result != nil && exec.Result.Status != nil {
						status := *exec.Result.Status
						// Ask the result whether it finished, rather than treating every status
						// other than queued/running as terminal: a child reporting `assigned`,
						// `starting` or `scheduling` is still on its way, and reading those as
						// terminal made the parent give up and report the child as failed.
						if exec.Result.IsFinished() {
							break loop
						}

						if prevStatus != status {
							instructions.PrintOutput(config.Ref(), "testworkflow-status", &executionResult{Id: exec.Id, Status: string(status)})
						}

						prevStatus = status
					}
				}

				// Publish the outputs the execution produced, so the following
				// executions and steps may read them with the execution() expression.
				// The execution record carries them once the status is terminal.
				req.recorder.complete(entry, exec)

				// Safe status access after loop
				if exec.Result == nil || exec.Result.Status == nil {
					outcomes[index].err = errors.New("completed but status unavailable")
					return
				}

				status := *exec.Result.Status
				color := ui.Green
				if status != testkube.PASSED_TestWorkflowStatus {
					outcomes[index].err = errors.New(string(status))
					color = ui.Red
				}

				instructions.PrintOutput(config.Ref(), "testworkflow-end", &executionResult{Id: exec.Id, Status: string(status)})
				fmt.Printf("%s • %s\n", color(exec.Name), string(status))
			}(i, entries[i], execs[i])
		}
		wg.Wait()

		result := operationResult{outcomes: outcomes}

		// Download the artifacts the executions produced, even when they failed -
		// a failed run's artifacts are usually the interesting ones.
		if fetchErr := fetchArtifacts(workflow.Fetch, entries, req.recorder.registry); fetchErr != nil {
			ui.Errf("failed to fetch artifacts: %s", fetchErr.Error())
			result.err = errors.Wrap(fetchErr, "fetching artifacts")
		}

		for _, o := range outcomes {
			if o.err != nil {
				ui.Errf("Execution error: execution %s failed: %s", o.name, o.err.Error())
			}
		}

		return result
	}
}

// notScheduled is the result of an entry that failed before the step scheduled an execution.
func notScheduled(workflowName string, err error) operationResult {
	return operationResult{outcomes: []executionOutcome{{name: workflowName, err: err}}}
}

// claimExecutionRefs reserves the references an entry will be addressed by. Two
// entries answering to the same reference would make execution() ambiguous, so ask
// for an explicit alias instead of silently picking a winner.
//
// An aliased entry claims one reference no matter how many workflows its selector
// matched - they form a single group, told apart by execution("<alias>", index).
// An entry without an alias claims each matched workflow by name.
//
// An aliased entry stays addressable by the workflow it ran, which this does not claim:
// two aliased entries may legitimately run the same workflow and be told apart by their
// aliases. That leaves the workflow name itself able to address more than one execution,
// which Registry.Lookup reports when it is used rather than forbidding here.
func claimExecutionRefs(claimed map[string]string, alias string, workflowNames []string) error {
	refs := workflowNames
	if alias != "" {
		refs = []string{alias}
	}
	for _, ref := range refs {
		if previous, ok := claimed[ref]; ok {
			return fmt.Errorf("duplicated execution reference %q, already used by %q: set a unique 'as' on one of them", ref, previous)
		}
		claimed[ref] = workflowNames[0]
	}
	return nil
}

// fetchArtifacts downloads artifacts produced by executed test workflows into the
// local file system, so large payloads can cross the workflow boundary without going
// through an expression.
//
// An entry without 'from' fetches from the executions it was declared on; otherwise the
// reference is resolved the way execution() resolves one: against what this workflow has
// executed so far, falling back to the control plane for a reference the registry does not
// know - a raw execution id handed down as configuration, or "parent".
func fetchArtifacts(fetch []testworkflowsv1.StepExecuteFetch, entries []executiondata.Execution, registry *executiondata.Registry) error {
	if len(fetch) == 0 {
		return nil
	}

	repository := data.ExecutionDataRepository()
	resolver := data.ExecutionResolver(registry)
	for i, f := range fetch {
		sources := entries
		if f.From != "" {
			source, err := resolver.Resolve(context.Background(), f.From, 0)
			if err != nil {
				return errors.Wrapf(err, "fetch.%d", i)
			}
			sources = []executiondata.Execution{source}
		}

		for _, source := range sources {
			result, err := executiondata.FetchArtifacts(context.Background(), repository, data.ArtifactClient(), source.Id, f.Paths, f.To)
			if err != nil {
				return errors.Wrapf(err, "fetch.%d", i)
			}
			fmt.Printf("%s • fetched %d files (%d bytes) into %s\n", ui.LightCyan(source.Name), result.Files, result.Bytes, f.To)
		}
	}
	return nil
}

func registerTransfer(transferSrv transfer.Server, request map[string]testworkflowsv1.TarballRequest, machines ...expressions.Machine) (expressions.Machine, error) {
	err := expressions.Finalize(&request, machines...)
	if err != nil {
		return nil, errors.Wrap(err, "computing tarball")
	}
	tarballs := make(map[string]transfer.Entry, len(request))
	for k, t := range request {
		patterns := []string{"**/*"}
		if t.Files != nil && !t.Files.Dynamic {
			patterns = spawn.MapDynamicListToStringList(t.Files.Static)
		} else if t.Files != nil && t.Files.Dynamic {
			patternsExpr, err := expressions.EvalExpression(t.Files.Expression, machines...)
			if err != nil {
				return nil, errors.Wrapf(err, "computing tarball: %s", k)
			}
			patternsList, err := patternsExpr.Static().SliceValue()
			if err != nil {
				return nil, errors.Wrapf(err, "computing tarball: %s", k)
			}
			patterns = make([]string, len(patternsList))
			for i, p := range patternsList {
				if s, ok := p.(string); ok {
					patterns[i] = s
				} else {
					p, err := json.Marshal(s)
					if err != nil {
						return nil, errors.Wrapf(err, "computing tarball: %s", k)
					}
					patterns[i] = string(p)
				}
			}
		}
		tarballs[k], err = transferSrv.Include(t.From, patterns)
		if err != nil {
			return nil, errors.Wrapf(err, "computing tarball: %s", k)
		}
	}
	return expressions.NewMachine().Register("tarball", tarballs), nil
}

func NewExecuteCmd() *cobra.Command {
	var (
		workflows     []string
		parallelism   int
		async         bool
		base64Encoded bool
	)

	cmd := &cobra.Command{
		Use:   "execute",
		Short: "Execute other resources",
		Args:  cobra.MaximumNArgs(1),

		Run: func(cmd *cobra.Command, args []string) {
			// Parse input based on encoding
			if base64Encoded && len(args) > 0 {
				// Decode base64 input. The processor base64-encodes execute specs to prevent
				// testworkflow-init from prematurely resolving expressions like {{ index + 1 }}.
				// We decode here where we have the proper context to evaluate these expressions.
				// Unmarshal the execute data
				type ExecuteData struct {
					Tests       []json.RawMessage `json:"tests,omitempty"`
					Workflows   []json.RawMessage `json:"workflows,omitempty"`
					Async       bool              `json:"async,omitempty"`
					Parallelism int               `json:"parallelism,omitempty"`
				}
				var executeData ExecuteData
				err := expressionstcl.DecodeBase64JSON(args[0], &executeData)
				if err != nil {
					toolkitcommon.Fail(errors.Wrap(err, "parsing execute data"))
				}

				workflows = make([]string, len(executeData.Workflows))
				for i, raw := range executeData.Workflows {
					workflows[i] = string(raw)
				}
				if executeData.Async {
					async = true
				}
				if executeData.Parallelism > 0 {
					parallelism = executeData.Parallelism
				}
			}

			// Initialize internal machine.
			// The registry is seeded with the test workflows executed by the previous steps,
			// and grows as this step schedules more of them.
			credMachine := credentials.NewCredentialMachine(data.Credentials())
			recorder := newExecutionRecorder(data.ExecutionRegistry())
			baseMachine := expressions.CombinedMachines(
				data.GetBaseTestWorkflowMachine(),
				data.ExecutionMachine(),
				data.ExecutionDataMachineFor(recorder.registry),
				credMachine,
			)

			// Initialize transfer server
			transferSrv := transfer.NewServer(constants.DefaultTransferDirPath, config.IP(), constants.DefaultTransferPort)

			// Build operations to run
			operations := make([]func() operationResult, 0)
			// aliases guards against two entries claiming the same execution() reference
			aliases := make(map[string]string)
			for _, s := range workflows {
				var w testworkflowsv1.StepExecuteWorkflow
				err := json.Unmarshal([]byte(s), &w)
				if err != nil {
					toolkitcommon.Fail(errors.Wrap(err, "unmarshal workflow definition"))
				}

				if w.Name == "" && w.Selector == nil {
					toolkitcommon.Fail(errors.New("either workflow name or selector should be specified"))
				}

				var testWorkflowNames []string
				if w.Name != "" {
					testWorkflowNames = []string{w.Name}
				}

				if w.Selector != nil {
					if len(w.Selector.MatchExpressions) > 0 {
						toolkitcommon.Fail(errors.New("error creating selector from test workflow selector: matchExpressions is not supported"))
					}
					testWorkflowsList, err := execute.ListTestWorkflows(w.Selector.MatchLabels)
					if err != nil {
						toolkitcommon.Fail(errors.Wrap(err, "error listing test workflows using selector"))
					}

					if len(testWorkflowsList) > 0 {
						ui.Info("List of test workflows found for selector specification:")
					} else {
						ui.Warn("No test workflows found for selector specification")
					}

					for _, item := range testWorkflowsList {
						testWorkflowNames = append(testWorkflowNames, item.Name)
						ui.Info("- " + item.Name)
					}
				}

				if len((testWorkflowNames)) == 0 {
					toolkitcommon.Fail(errors.New("no test workflows to run"))
				}

				// Resolve the params
				params, err := commontcl.GetParamsSpec(w.Matrix, w.Shards, w.Count, w.MaxCount, baseMachine)
				if err != nil {
					toolkitcommon.Fail(errors.Wrap(err, "matrix and sharding"))
				}

				// Resolve the reference this entry will be addressed by. It cannot depend
				// on any sibling execution, so it is safe to compute it up-front.
				alias, err := expressions.EvalTemplate(w.As, baseMachine)
				if err != nil {
					toolkitcommon.Fail(errors.Wrapf(err, "'%s' workflow: computing the 'as' reference", w.Name))
				}

				if err := claimExecutionRefs(aliases, alias, testWorkflowNames); err != nil {
					toolkitcommon.Fail(err)
				}

				for _, testWorkflowName := range testWorkflowNames {
					fmt.Printf("%s: %s\n", commontcl.ServiceLabel(testWorkflowName), params.Humanize())

					// Create operations for each expected execution
					for i := int64(0); i < params.Count; i++ {
						// Clone the spec
						spec := w.DeepCopy()
						spec.Name = testWorkflowName
						spec.As = alias

						// Build files for transfer
						tarballMachine, err := registerTransfer(transferSrv, spec.Tarball, baseMachine, params.MachineAt(i))
						if err != nil {
							toolkitcommon.Fail(errors.Wrapf(err, "'%s' workflow", spec.Name))
						}
						spec.Tarball = nil

						// Prepare the operation to run. The spec stays unresolved until the
						// operation starts, so that its configuration may reference the
						// executions scheduled before it.
						operations = append(operations, buildWorkflowExecution(workflowExecutionRequest{
							spec:     spec,
							machines: []expressions.Machine{baseMachine, tarballMachine, params.MachineAt(i)},
							alias:    alias,
							async:    async,
							recorder: recorder,
						}))
					}
				}
			}

			// Validate if there is anything to run
			if len(operations) == 0 {
				fmt.Printf("nothing to run\n")
				os.Exit(0)
			}

			// Initialize transfer server if expected
			if transferSrv.Count() > 0 {
				fmt.Printf("Starting transfer server for %d tarballs...\n", transferSrv.Count())
				if _, err := transferSrv.Listen(); err != nil {
					toolkitcommon.Fail(errors.Wrap(err, "failed to start transfer server"))
				}
				fmt.Printf("Transfer server started.\n")
			}

			// Calculate parallelism
			if parallelism <= 0 {
				parallelism = 100
			}
			if parallelism < len(operations) {
				fmt.Printf("Total: %d executions, %d parallel\n", len(operations), parallelism)
			} else {
				fmt.Printf("Total: %d executions, all in parallel\n", len(operations))
			}

			// Create channel for execution
			var wg sync.WaitGroup
			wg.Add(len(operations))
			ch := make(chan struct{}, parallelism)
			// Each goroutine writes only its own item of results.
			results := make([]operationResult, len(operations))

			// Execute all operations
			for i, op := range operations {
				ch <- struct{}{}
				go func(i int, op func() operationResult) {
					results[i] = op()
					<-ch
					wg.Done()
				}(i, op)
			}
			wg.Wait()

			// The summary becomes the step message, so it names the failed executions.
			if summary := failureSummary(results); summary != "" {
				toolkitcommon.Fail(errors.New(summary))
			}
		},
	}

	cmd.Flags().StringArrayVarP(&workflows, "workflow", "w", nil, "workflows to run")
	cmd.Flags().IntVarP(&parallelism, "parallelism", "p", 0, "how many items could be executed at once")
	cmd.Flags().BoolVar(&async, "async", false, "should it wait for results")
	cmd.Flags().BoolVar(&base64Encoded, "base64", false, "input is base64 encoded")

	return cmd
}
