// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package commands

import (
	"encoding/json"
	"fmt"
	"os"
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
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env/config"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/transfer"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/credentials"
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

type testExecutionDetails struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	TestName    string `json:"testName"`
	Description string `json:"description,omitempty"`
}

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

func buildTestExecution(test testworkflowsv1.StepExecuteTest, async bool) (func() error, error) {
	return func() (err error) {
		c := env.Testkube()

		if test.ExecutionRequest == nil {
			test.ExecutionRequest = &testworkflowsv1.TestExecutionRequest{}
		}

		exec, err := c.ExecuteTest(test.Name, test.ExecutionRequest.Name, client.ExecuteTestOptions{
			RunningContext: &testkube.RunningContext{
				Type_:   string(testkube.RunningContextTypeTestWorkflow),
				Context: fmt.Sprintf("%s/executions/%s", config.WorkflowName(), config.ExecutionId()),
			},
			IsVariablesFileUploaded:            test.ExecutionRequest.IsVariablesFileUploaded,
			ExecutionLabels:                    test.ExecutionRequest.ExecutionLabels,
			Command:                            test.ExecutionRequest.Command,
			Args:                               test.ExecutionRequest.Args,
			ArgsMode:                           string(test.ExecutionRequest.ArgsMode),
			HTTPProxy:                          test.ExecutionRequest.HttpProxy,
			HTTPSProxy:                         test.ExecutionRequest.HttpsProxy,
			Image:                              test.ExecutionRequest.Image,
			ArtifactRequest:                    common.MapPtr(test.ExecutionRequest.ArtifactRequest, testworkflows.MapTestArtifactRequestKubeToAPI),
			JobTemplate:                        test.ExecutionRequest.JobTemplate,
			PreRunScriptContent:                test.ExecutionRequest.PreRunScript,
			PostRunScriptContent:               test.ExecutionRequest.PostRunScript,
			ExecutePostRunScriptBeforeScraping: test.ExecutionRequest.ExecutePostRunScriptBeforeScraping,
			SourceScripts:                      test.ExecutionRequest.SourceScripts,
			ScraperTemplate:                    test.ExecutionRequest.ScraperTemplate,
			NegativeTest:                       test.ExecutionRequest.NegativeTest,
			EnvConfigMaps:                      common.MapSlice(test.ExecutionRequest.EnvConfigMaps, testworkflows.MapTestEnvReferenceKubeToAPI),
			EnvSecrets:                         common.MapSlice(test.ExecutionRequest.EnvSecrets, testworkflows.MapTestEnvReferenceKubeToAPI),
			ExecutionNamespace:                 test.ExecutionRequest.ExecutionNamespace,
			DisableWebhooks:                    config.ExecutionDisableWebhooks(),
		})
		execName := exec.Name
		if err != nil {
			ui.Errf("failed to execute test: %s: %s", test.Name, err)
			return
		}

		instructions.PrintOutput(config.Ref(), "test-start", &testExecutionDetails{
			Id:          exec.Id,
			Name:        exec.Name,
			TestName:    exec.TestName,
			Description: test.Description,
		})
		description := ""
		if test.Description != "" {
			description = fmt.Sprintf(": %s", test.Description)
		}
		fmt.Printf("%s%s • scheduled %s\n", ui.LightCyan(execName), description, ui.DarkGray("("+exec.Id+")"))

		if async {
			return
		}

		prevStatus := testkube.QUEUED_ExecutionStatus
	loop:
		for {
			time.Sleep(time.Second)
			exec, err = c.GetExecution(exec.Id)
			if err != nil {
				ui.Errf("error while getting execution result: %s: %s", ui.LightCyan(execName), err.Error())
				return
			}
			if exec.ExecutionResult != nil && exec.ExecutionResult.Status != nil {
				status := *exec.ExecutionResult.Status
				switch status {
				case testkube.QUEUED_ExecutionStatus, testkube.RUNNING_ExecutionStatus:
					break
				default:
					break loop
				}
				if prevStatus != status {
					instructions.PrintOutput(config.Ref(), "test-status", &executionResult{Id: exec.Id, Status: string(status)})
				}
				prevStatus = status
			}
		}

		status := *exec.ExecutionResult.Status
		color := ui.Green

		if status != testkube.PASSED_ExecutionStatus {
			err = errors.New("test failed")
			color = ui.Red
		}

		instructions.PrintOutput(config.Ref(), "test-end", &executionResult{Id: exec.Id, Status: string(status)})
		fmt.Printf("%s • %s\n", color(execName), string(status))
		return
	}, nil
}

func buildWorkflowExecution(workflow testworkflowsv1.StepExecuteWorkflow, async bool) (func() error, error) {
	return func() (err error) {
		tags := config.ExecutionTags()
		target := common.MapPtr(workflow.Target, commonmapper.MapTargetKubeToAPI)

		// Schedule execution
		var execs []testkube.TestWorkflowExecution
		for i := 0; i < CreateExecutionRetryOnFailureMaxAttempts; i++ {
			execs, err = execute.ExecuteTestWorkflow(workflow.Name, testkube.TestWorkflowExecutionRequest{
				Name:            workflow.ExecutionName,
				Config:          testworkflows.MapConfigValueKubeToAPI(workflow.Config),
				DisableWebhooks: config.ExecutionDisableWebhooks(),
				Tags:            tags,
				Target:          target,
			})
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
			return
		}

		// Print information about scheduled execution
		for _, exec := range execs {
			instructions.PrintOutput(config.Ref(), "testworkflow-start", &testWorkflowExecutionDetails{
				Id:               exec.Id,
				Name:             exec.Name,
				TestWorkflowName: exec.Workflow.Name,
				Description:      workflow.Description,
			})

			description := ""
			if workflow.Description != "" {
				description = fmt.Sprintf(": %s", workflow.Description)
			}
			fmt.Printf("%s%s • scheduled %s\n", ui.LightCyan(exec.Name), description, ui.DarkGray("("+exec.Id+")"))
		}

		if async {
			return
		}

		// Monitor
		var wg sync.WaitGroup
		var mu sync.Mutex
		var errs []error // Collect errors safely

		wg.Add(len(execs))
		for i := range execs {
			go func(exec testkube.TestWorkflowExecution) {
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
						mu.Lock()
						errs = append(errs, gErr)
						mu.Unlock()
						return
					}

					if exec.Result != nil && exec.Result.Status != nil {
						status := *exec.Result.Status
						switch status {
						case testkube.QUEUED_TestWorkflowStatus, testkube.RUNNING_TestWorkflowStatus:
							break
						default:
							break loop
						}

						if prevStatus != status {
							instructions.PrintOutput(config.Ref(), "testworkflow-status", &executionResult{Id: exec.Id, Status: string(status)})
						}

						prevStatus = status
					}
				}

				// Safe status access after loop
				if exec.Result == nil || exec.Result.Status == nil {
					mu.Lock()
					errs = append(errs, fmt.Errorf("execution %s completed but status unavailable", exec.Name))
					mu.Unlock()
					return
				}

				status := *exec.Result.Status
				color := ui.Green
				if status != testkube.PASSED_TestWorkflowStatus {
					mu.Lock()
					errs = append(errs, fmt.Errorf("execution %s failed", exec.Name))
					mu.Unlock()
					color = ui.Red
				}

				instructions.PrintOutput(config.Ref(), "testworkflow-end", &executionResult{Id: exec.Id, Status: string(status)})
				fmt.Printf("%s • %s\n", color(exec.Name), string(status))
			}(execs[i])
		}
		wg.Wait()

		// Handle collected errors
		if len(errs) > 0 {
			for _, lErr := range errs {
				ui.Errf("Execution error: %s", lErr.Error())
			}

			return fmt.Errorf("one or more executions failed")
		}

		return
	}, nil
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
		tests         []string
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
					ui.Fail(errors.Wrap(err, "parsing execute data"))
				}

				// Extract individual test/workflow specs from the decoded data
				tests = make([]string, len(executeData.Tests))
				for i, raw := range executeData.Tests {
					tests[i] = string(raw)
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

			// Initialize internal machine
			credMachine := credentials.NewCredentialMachine(data.Credentials())
			baseMachine := expressions.CombinedMachines(data.GetBaseTestWorkflowMachine(), data.ExecutionMachine(), credMachine)

			// Initialize transfer server
			transferSrv := transfer.NewServer(constants.DefaultTransferDirPath, config.IP(), constants.DefaultTransferPort)

			// Build operations to run
			operations := make([]func() error, 0)
			for _, s := range tests {
				var t testworkflowsv1.StepExecuteTest
				err := json.Unmarshal([]byte(s), &t)
				if err != nil {
					ui.Fail(errors.Wrap(err, "unmarshal test definition"))
				}

				// Resolve the params
				params, err := commontcl.GetParamsSpec(t.Matrix, t.Shards, t.Count, t.MaxCount, baseMachine)
				if err != nil {
					ui.Fail(errors.Wrap(err, "matrix and sharding"))
				}
				fmt.Printf("%s: %s\n", commontcl.ServiceLabel(t.Name), params.Humanize())

				// Create operations for each expected execution
				for i := int64(0); i < params.Count; i++ {
					// Clone the spec
					spec := t.DeepCopy()

					// Build files for transfer
					tarballMachine, err := registerTransfer(transferSrv, spec.Tarball, baseMachine, params.MachineAt(i))
					if err != nil {
						ui.Fail(errors.Wrapf(err, "'%s' workflow", spec.Name))
					}
					spec.Tarball = nil

					// Prepare the operation to run
					err = expressions.Finalize(&spec, baseMachine, tarballMachine, params.MachineAt(i))
					if err != nil {
						ui.Fail(errors.Wrapf(err, "'%s' test: computing execution", spec.Name))
					}
					fn, err := buildTestExecution(*spec, async)
					if err != nil {
						ui.Fail(err)
					}
					operations = append(operations, fn)
				}
			}

			for _, s := range workflows {
				var w testworkflowsv1.StepExecuteWorkflow
				err := json.Unmarshal([]byte(s), &w)
				if err != nil {
					ui.Fail(errors.Wrap(err, "unmarshal workflow definition"))
				}

				if w.Name == "" && w.Selector == nil {
					ui.Fail(errors.New("either workflow name or selector should be specified"))
				}

				var testWorkflowNames []string
				if w.Name != "" {
					testWorkflowNames = []string{w.Name}
				}

				if w.Selector != nil {
					if len(w.Selector.MatchExpressions) > 0 {
						ui.Fail(errors.New("error creating selector from test workflow selector: matchExpressions is not supported"))
					}
					testWorkflowsList, err := execute.ListTestWorkflows(w.Selector.MatchLabels)
					if err != nil {
						ui.Fail(errors.Wrap(err, "error listing test workflows using selector"))
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
					ui.Fail(errors.New("no test workflows to run"))
				}

				// Resolve the params
				params, err := commontcl.GetParamsSpec(w.Matrix, w.Shards, w.Count, w.MaxCount, baseMachine)
				if err != nil {
					ui.Fail(errors.Wrap(err, "matrix and sharding"))
				}

				for _, testWorkflowName := range testWorkflowNames {
					fmt.Printf("%s: %s\n", commontcl.ServiceLabel(testWorkflowName), params.Humanize())

					// Create operations for each expected execution
					for i := int64(0); i < params.Count; i++ {
						// Clone the spec
						spec := w.DeepCopy()
						spec.Name = testWorkflowName

						// Build files for transfer
						tarballMachine, err := registerTransfer(transferSrv, spec.Tarball, baseMachine, params.MachineAt(i))
						if err != nil {
							ui.Fail(errors.Wrapf(err, "'%s' workflow", spec.Name))
						}
						spec.Tarball = nil

						// Prepare the operation to run
						err = expressions.Finalize(&spec, baseMachine, tarballMachine, params.MachineAt(i))
						if err != nil {
							ui.Fail(errors.Wrapf(err, "'%s' workflow: computing execution", spec.Name))
						}
						fn, err := buildWorkflowExecution(*spec, async)
						if err != nil {
							ui.Fail(err)
						}
						operations = append(operations, fn)
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
					ui.Fail(errors.Wrap(err, "failed to start transfer server"))
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
			success := true

			// Execute all operations
			for _, op := range operations {
				ch <- struct{}{}
				go func(op func() error) {
					if op() != nil {
						success = false
					}
					<-ch
					wg.Done()
				}(op)
			}
			wg.Wait()

			if !success {
				os.Exit(1)
			}
		},
	}

	// TODO: Support test suites too
	cmd.Flags().StringArrayVarP(&tests, "test", "t", nil, "tests to run")
	cmd.Flags().StringArrayVarP(&workflows, "workflow", "w", nil, "workflows to run")
	cmd.Flags().IntVarP(&parallelism, "parallelism", "p", 0, "how many items could be executed at once")
	cmd.Flags().BoolVar(&async, "async", false, "should it wait for results")
	cmd.Flags().BoolVar(&base64Encoded, "base64", false, "input is base64 encoded")

	return cmd
}
