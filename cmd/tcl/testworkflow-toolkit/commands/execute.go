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
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

type testExecutionDetails struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	TestName string `json:"testName"`
}

type testWorkflowExecutionDetails struct {
	Id               string `json:"id"`
	Name             string `json:"name"`
	TestWorkflowName string `json:"testWorkflowName"`
}

type executionResult struct {
	Id     string `json:"id"`
	Status string `json:"status"`
}

func buildTestExecution(test string, async bool) (func() error, error) {
	name, req, _ := strings.Cut(test, "=")
	request := testkube.ExecutionRequest{}
	if req != "" {
		err := json.Unmarshal([]byte(req), &request)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to unmarshal execution request: %s: %s", name, req))
		}
	}
	if request.ExecutionLabels == nil {
		request.ExecutionLabels = map[string]string{}
	}

	return func() (err error) {
		c := env.Testkube()

		exec, err := c.ExecuteTest(name, request.Name, client.ExecuteTestOptions{
			RunningContext: &testkube.RunningContext{
				Type_:   "testworkflow",
				Context: fmt.Sprintf("%s/executions/%s", env.WorkflowName(), env.ExecutionId()),
			},
			IsVariablesFileUploaded:            request.IsVariablesFileUploaded,
			ExecutionLabels:                    request.ExecutionLabels,
			Command:                            request.Command,
			Args:                               request.Args,
			ArgsMode:                           request.ArgsMode,
			Envs:                               request.Envs,
			SecretEnvs:                         request.SecretEnvs,
			HTTPProxy:                          request.HttpProxy,
			HTTPSProxy:                         request.HttpsProxy,
			Image:                              request.Image,
			Uploads:                            request.Uploads,
			BucketName:                         request.BucketName,
			ArtifactRequest:                    request.ArtifactRequest,
			JobTemplate:                        request.JobTemplate,
			JobTemplateReference:               request.JobTemplateReference,
			ContentRequest:                     request.ContentRequest,
			PreRunScriptContent:                request.PreRunScript,
			PostRunScriptContent:               request.PostRunScript,
			ExecutePostRunScriptBeforeScraping: request.ExecutePostRunScriptBeforeScraping,
			SourceScripts:                      request.SourceScripts,
			ScraperTemplate:                    request.ScraperTemplate,
			ScraperTemplateReference:           request.ScraperTemplateReference,
			PvcTemplate:                        request.PvcTemplate,
			PvcTemplateReference:               request.PvcTemplateReference,
			NegativeTest:                       request.NegativeTest,
			IsNegativeTestChangedOnRun:         request.IsNegativeTestChangedOnRun,
			EnvConfigMaps:                      request.EnvConfigMaps,
			EnvSecrets:                         request.EnvSecrets,
			SlavePodRequest:                    request.SlavePodRequest,
			ExecutionNamespace:                 request.ExecutionNamespace,
		})
		execName := exec.Name
		if err != nil {
			ui.Errf("failed to execute test: %s: %s", name, err)
			return
		}

		data.PrintOutput(env.Ref(), "test-start", &testExecutionDetails{
			Id:       exec.Id,
			Name:     exec.Name,
			TestName: exec.TestName,
		})
		fmt.Printf("%s • scheduled %s\n", ui.LightCyan(execName), ui.DarkGray("("+exec.Id+")"))

		if async {
			return
		}

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
					continue
				default:
					break loop
				}
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

func buildWorkflowExecution(workflow string, async bool) (func() error, error) {
	name, req, _ := strings.Cut(workflow, "=")
	request := testkube.TestWorkflowExecutionRequest{}
	if req != "" {
		err := json.Unmarshal([]byte(req), &request)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to unmarshal execution request: %s: %s", name, req))
		}
	}

	return func() (err error) {
		c := env.Testkube()

		exec, err := c.ExecuteTestWorkflow(name, request)
		execName := exec.Name
		if err != nil {
			ui.Errf("failed to execute test workflow: %s: %s", name, err.Error())
			return
		}

		data.PrintOutput(env.Ref(), "testworkflow-start", &testWorkflowExecutionDetails{
			Id:               exec.Id,
			Name:             exec.Name,
			TestWorkflowName: exec.Workflow.Name,
		})
		fmt.Printf("%s • scheduled %s\n", ui.LightCyan(execName), ui.DarkGray("("+exec.Id+")"))

		if async {
			return
		}

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
					continue
				default:
					break loop
				}
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

func NewExecuteCmd() *cobra.Command {
	var (
		tests       []string
		workflows   []string
		parallelism int
		async       bool
	)

	cmd := &cobra.Command{
		Use:   "execute",
		Short: "Execute other resources",
		Args:  cobra.ExactArgs(0),

		Run: func(cmd *cobra.Command, _ []string) {
			// Calculate parallelism
			if parallelism <= 0 {
				parallelism = 20
			}

			// Build operations to run
			operations := make([]func() error, 0)
			for _, t := range tests {
				fn, err := buildTestExecution(t, async)
				if err != nil {
					ui.Fail(err)
				}
				operations = append(operations, fn)
			}
			for _, w := range workflows {
				fn, err := buildWorkflowExecution(w, async)
				if err != nil {
					ui.Fail(err)
				}
				operations = append(operations, fn)
			}

			// Validate if there is anything to run
			if len(operations) == 0 {
				fmt.Printf("nothing to run\n")
				os.Exit(0)
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
	cmd.Flags().StringArrayVarP(&tests, "test", "t", nil, "tests to run; either test name, or test-name=json-execution-request")
	cmd.Flags().StringArrayVarP(&workflows, "workflow", "w", nil, "workflows to run; either workflow name, or workflow-name=json-execution-request")
	cmd.Flags().IntVarP(&parallelism, "parallelism", "p", 0, "how many items could be executed at once")
	cmd.Flags().BoolVar(&async, "async", false, "should it wait for results")

	return cmd
}
