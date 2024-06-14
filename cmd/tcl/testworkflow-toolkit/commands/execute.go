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

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-init/data"
	common2 "github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/common"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/spawn"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/transfer"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/tcl/mapperstcl/testworkflows"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/ui"
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

func buildTestExecution(test testworkflowsv1.StepExecuteTest, async, disableWebhooks bool) (func() error, error) {
	return func() (err error) {
		c := env.Testkube()

		if test.ExecutionRequest == nil {
			test.ExecutionRequest = &testworkflowsv1.TestExecutionRequest{}
		}

		exec, err := c.ExecuteTest(test.Name, test.ExecutionRequest.Name, client.ExecuteTestOptions{
			RunningContext: &testkube.RunningContext{
				Type_:   "testworkflow",
				Context: fmt.Sprintf("%s/executions/%s", env.WorkflowName(), env.ExecutionId()),
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
			DisableWebhooks:                    disableWebhooks,
		})
		execName := exec.Name
		if err != nil {
			ui.Errf("failed to execute test: %s: %s", test.Name, err)
			return
		}

		data.PrintOutput(env.Ref(), "test-start", &testExecutionDetails{
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
					data.PrintOutput(env.Ref(), "test-status", &executionResult{Id: exec.Id, Status: string(status)})
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

		data.PrintOutput(env.Ref(), "test-end", &executionResult{Id: exec.Id, Status: string(status)})
		fmt.Printf("%s • %s\n", color(execName), string(status))
		return
	}, nil
}

func buildWorkflowExecution(workflow testworkflowsv1.StepExecuteWorkflow, async bool) (func() error, error) {
	return func() (err error) {
		c := env.Testkube()

		exec, err := c.ExecuteTestWorkflow(workflow.Name, testkube.TestWorkflowExecutionRequest{
			Name:   workflow.ExecutionName,
			Config: testworkflows.MapConfigValueKubeToAPI(workflow.Config),
		})
		execName := exec.Name
		if err != nil {
			ui.Errf("failed to execute test workflow: %s: %s", workflow.Name, err.Error())
			return
		}

		data.PrintOutput(env.Ref(), "testworkflow-start", &testWorkflowExecutionDetails{
			Id:               exec.Id,
			Name:             exec.Name,
			TestWorkflowName: exec.Workflow.Name,
			Description:      workflow.Description,
		})
		description := ""
		if workflow.Description != "" {
			description = fmt.Sprintf(": %s", workflow.Description)
		}
		fmt.Printf("%s%s • scheduled %s\n", ui.LightCyan(execName), description, ui.DarkGray("("+exec.Id+")"))

		if async {
			return
		}

		prevStatus := testkube.QUEUED_TestWorkflowStatus
	loop:
		for {
			time.Sleep(100 * time.Millisecond)
			exec, err = c.GetTestWorkflowExecution(exec.Id)
			if err != nil {
				ui.Errf("error while getting execution result: %s: %s", ui.LightCyan(execName), err.Error())
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
					data.PrintOutput(env.Ref(), "testworkflow-status", &executionResult{Id: exec.Id, Status: string(status)})
				}
				prevStatus = status
			}
		}

		status := *exec.Result.Status
		color := ui.Green

		if status != testkube.PASSED_TestWorkflowStatus {
			err = errors.New("test workflow failed")
			color = ui.Red
		}

		data.PrintOutput(env.Ref(), "testworkflow-end", &executionResult{Id: exec.Id, Status: string(status)})
		fmt.Printf("%s • %s\n", color(execName), string(status))
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
		tests           []string
		workflows       []string
		parallelism     int
		async           bool
		disableWebhooks bool
	)

	cmd := &cobra.Command{
		Use:   "execute",
		Short: "Execute other resources",
		Args:  cobra.ExactArgs(0),

		Run: func(cmd *cobra.Command, _ []string) {
			// Initialize internal machine
			baseMachine := data.GetBaseTestWorkflowMachine()

			// Initialize transfer server
			transferSrv := transfer.NewServer(constants.DefaultTransferDirPath, env.IP(), constants.DefaultTransferPort)

			// Build operations to run
			operations := make([]func() error, 0)
			for _, s := range tests {
				var t testworkflowsv1.StepExecuteTest
				err := json.Unmarshal([]byte(s), &t)
				if err != nil {
					ui.Fail(errors.Wrap(err, "unmarshal test definition"))
				}

				// Resolve the params
				params, err := common2.GetParamsSpec(t.Matrix, t.Shards, t.Count, t.MaxCount, baseMachine)
				if err != nil {
					ui.Fail(errors.Wrap(err, "matrix and sharding"))
				}
				fmt.Printf("%s: %s\n", common2.ServiceLabel(t.Name), params.Humanize())

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
					fn, err := buildTestExecution(*spec, async, disableWebhooks)
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

				// Resolve the params
				params, err := common2.GetParamsSpec(w.Matrix, w.Shards, w.Count, w.MaxCount, baseMachine)
				if err != nil {
					ui.Fail(errors.Wrap(err, "matrix and sharding"))
				}
				fmt.Printf("%s: %s\n", common2.ServiceLabel(w.Name), params.Humanize())

				// Create operations for each expected execution
				for i := int64(0); i < params.Count; i++ {
					// Clone the spec
					spec := w.DeepCopy()

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
	cmd.Flags().BoolVar(&disableWebhooks, "disableWebhooks", false, "should it disable webhooks")

	return cmd
}
