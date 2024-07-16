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

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	commontcl "github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/common"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/spawn"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/artifacts"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/transfer"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowcontroller"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/presets"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
	"github.com/kubeshop/testkube/pkg/ui"
)

type ParallelStatus struct {
	Index       int                              `json:"index"`
	Description string                           `json:"description,omitempty"`
	Current     string                           `json:"current,omitempty"`
	Logs        string                           `json:"logs,omitempty"`
	Status      testkube.TestWorkflowStatus      `json:"status,omitempty"`
	Signature   []testkube.TestWorkflowSignature `json:"signature,omitempty"`
	Result      *testkube.TestWorkflowResult     `json:"result,omitempty"`
}

func NewParallelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "parallel <spec>",
		Short: "Run parallel steps",
		Args:  cobra.ExactArgs(1),

		Run: func(cmd *cobra.Command, args []string) {
			// Initialize internal machine
			baseMachine := spawn.CreateBaseMachine()

			// Read the template
			var parallel *testworkflowsv1.StepParallel
			err := json.Unmarshal([]byte(args[0]), &parallel)
			ui.ExitOnError("parsing parallel spec", err)

			// Inject short syntax down
			if !reflect.ValueOf(parallel.StepControl).IsZero() || !reflect.ValueOf(parallel.StepOperations).IsZero() {
				parallel.Steps = append([]testworkflowsv1.Step{{
					StepControl:    parallel.StepControl,
					StepOperations: parallel.StepOperations,
				}}, parallel.Steps...)
				parallel.StepControl = testworkflowsv1.StepControl{}
				parallel.StepOperations = testworkflowsv1.StepOperations{}
			}

			// Initialize transfer server
			transferSrv := transfer.NewServer(constants.DefaultTransferDirPath, env.IP(), constants.DefaultTransferPort)

			// Resolve the params
			params, err := commontcl.GetParamsSpec(parallel.Matrix, parallel.Shards, parallel.Count, parallel.MaxCount, baseMachine)
			ui.ExitOnError("compute matrix and sharding", err)

			// Clean up universal copy
			parallel.StepExecuteStrategy = testworkflowsv1.StepExecuteStrategy{}
			if parallel.Content == nil {
				parallel.Content = &testworkflowsv1.Content{}
			}

			// Apply default service account
			if parallel.Pod == nil {
				parallel.Pod = &testworkflowsv1.PodConfig{}
			}
			if parallel.Pod.ServiceAccountName == "" {
				parallel.Pod.ServiceAccountName = "{{internal.serviceaccount.default}}"
			}

			// Print information about the computed request
			if params.Count == 0 {
				fmt.Printf("0 instances requested (combinations=%d, count=%d), skipping\n", params.MatrixCount, params.ShardCount)
				os.Exit(0)
			}

			// Print information
			parallelism := int64(parallel.Parallelism)
			if parallelism <= 0 {
				parallelism = spawn.DefaultParallelism
			}
			fmt.Println(params.String(parallelism))

			// Analyze instances to run
			specs := make([]testworkflowsv1.TestWorkflowSpec, params.Count)
			descriptions := make([]string, params.Count)
			logConditions := make([]*string, params.Count)
			for i := int64(0); i < params.Count; i++ {
				machines := []expressions.Machine{baseMachine, params.MachineAt(i)}

				// Copy the log condition
				if parallel.Logs != nil {
					logConditions[i] = common.Ptr(*parallel.Logs)
				}

				// Clone the spec
				spec := parallel.DeepCopy()
				err = expressions.Simplify(&spec, machines...)
				ui.ExitOnError(fmt.Sprintf("%d: error", i), err)

				// Prepare the transfer
				tarballs, err := spawn.ProcessTransfer(transferSrv, spec.Transfer, machines...)
				ui.ExitOnError(fmt.Sprintf("%d: error: transfer", i), err)
				spec.Content.Tarball = append(spec.Content.Tarball, tarballs...)

				// Prepare the fetch
				fetchStep, err := spawn.ProcessFetch(transferSrv, spec.Fetch, machines...)
				ui.ExitOnError(fmt.Sprintf("%d: error: fetch", i), err)
				if fetchStep != nil {
					spec.After = append(spec.After, *fetchStep)
				}

				// Prepare the workflow to run
				specs[i] = spec.TestWorkflowSpec
				descriptions[i] = spec.Description
			}

			// Initialize transfer server if expected
			if transferSrv.Count() > 0 || transferSrv.RequestsCount() > 0 {
				infos := make([]string, 0)
				if transferSrv.Count() > 0 {
					infos = append(infos, fmt.Sprintf("sending %d tarballs", transferSrv.Count()))
				}
				if transferSrv.RequestsCount() > 0 {
					infos = append(infos, fmt.Sprintf("fetching %d requests", transferSrv.RequestsCount()))
				}
				fmt.Printf("Starting transfer server for %s...\n", strings.Join(infos, " and "))
				if _, err = transferSrv.Listen(); err != nil {
					ui.Fail(errors.Wrap(err, "failed to start transfer server"))
				}
				fmt.Printf("Transfer server started.\n")
			}

			// Validate if there is anything to run
			if len(specs) == 0 {
				ui.SuccessAndExit("nothing to run")
			}

			// Send initial output
			for index := range specs {
				data.PrintOutput(env.Ref(), "parallel", ParallelStatus{
					Index:       index,
					Description: descriptions[index],
				})
			}

			// Load Kubernetes client and image inspector
			inspector := env.ImageInspector()
			storage := artifacts.InternalStorage()

			// Prepare runner
			// TODO: Share resources like configMaps?
			type Update struct {
				index  int64
				result *testkube.TestWorkflowResult
				done   bool
				err    error
			}
			updates := make(chan Update, 100)
			registry := spawn.NewRegistry()
			run := func(index int64, spec *testworkflowsv1.TestWorkflowSpec) bool {
				clientSet := env.Kubernetes()
				log := spawn.CreateLogger("worker", descriptions[index], index, params.Count)
				id, machine := spawn.CreateExecutionMachine("", index)

				updates <- Update{index: index}

				// Build the resources bundle
				scheduledAt := time.Now()
				bundle, err := presets.NewPro(inspector).
					Bundle(context.Background(), &testworkflowsv1.TestWorkflow{Spec: *spec}, machine, baseMachine, params.MachineAt(index))
				if err != nil {
					fmt.Printf("%d: failed to prepare resources: %s\n", index, err.Error())
					return false
				}

				// Compute the bundle instructions
				sig := stage.MapSignatureListToInternal(bundle.Signature)
				namespace := bundle.Job.Namespace
				if namespace == "" {
					namespace = env.Namespace()
				}

				// Deploy the resources
				err = bundle.Deploy(context.Background(), clientSet, namespace)
				if err != nil {
					log("problem deploying", err.Error())
					return false
				}

				// Final clean up
				var lastResult testkube.TestWorkflowResult
				defer func() {
					shouldSaveLogs := logConditions[index] == nil
					if !shouldSaveLogs {
						shouldSaveLogs, _ = spawn.EvalLogCondition(*logConditions[index], lastResult, machine, baseMachine, params.MachineAt(index))
						if err != nil {
							log("warning", "log condition", err.Error())
						}
					}

					// Save logs
					if shouldSaveLogs {
						logsFilePath, err := spawn.SaveLogs(context.Background(), clientSet, storage, namespace, id, "", index)
						if err == nil {
							data.PrintOutput(env.Ref(), "parallel", ParallelStatus{Index: int(index), Logs: storage.FullPath(logsFilePath)})
							log("saved logs")
						} else {
							log("warning", "problem saving the logs", err.Error())
						}
					}

					// Clean up
					err = testworkflowcontroller.Cleanup(context.Background(), clientSet, namespace, id)
					if err == nil {
						log("cleaned resources")
					} else {
						log("warning", "problem cleaning up resources", err.Error())
					}
					updates <- Update{index: index, done: true, err: err}
				}()

				// Inform about the step structure
				data.PrintOutput(env.Ref(), "parallel", ParallelStatus{Index: int(index), Signature: sig})

				// Control the execution
				// TODO: Consider aggregated controller to limit number of watchers
				ctrl, err := testworkflowcontroller.New(context.Background(), clientSet, namespace, id, scheduledAt, testworkflowcontroller.ControllerOptions{
					Timeout: spawn.ControllerTimeout,
				})
				if err != nil {
					log("error", "failed to connect to the job", err.Error())
					return false
				}
				registry.Set(index, ctrl)
				ctx, ctxCancel := context.WithCancel(context.Background())
				log("created")

				prevStatus := testkube.QUEUED_TestWorkflowStatus
				prevStep := ""
				scheduled := false
				for v := range ctrl.WatchLightweight(ctx) {
					// Handle error
					if v.Error != nil {
						log("error", v.Error.Error())
						continue
					}

					// Inform about the node assignment
					if !scheduled && v.NodeName != "" {
						scheduled = true
						log(fmt.Sprintf("assigned to %s node", ui.LightBlue(v.NodeName)))
					}

					// Save the last result
					if v.Result != nil {
						lastResult = *v.Result
					}

					// Handle result change
					if v.Status != prevStatus || v.Current != prevStep {
						if v.Status != prevStatus {
							log(string(v.Status))
						}
						updates <- Update{index: index, result: v.Result}
						prevStep = v.Current
						prevStatus = v.Status
						if v.Result.IsFinished() {
							data.PrintOutput(env.Ref(), "parallel", ParallelStatus{Index: int(index), Status: v.Status, Result: v.Result})
							ctxCancel()
							return v.Result.IsPassed()
						} else {
							data.PrintOutput(env.Ref(), "parallel", ParallelStatus{Index: int(index), Status: v.Status, Current: v.Current})
						}
					}
				}

				ctxCancel()
				return false
			}

			// Orchestrate resume
			go func() {
				for update := range updates {
					if update.result != nil {
						registry.SetStatus(update.index, update.result.Status)
					}

					// Delete obsolete data
					if update.done || update.err != nil {
						registry.Destroy(update.index)
					}

					// Resume all at once
					if registry.Count() > 0 && registry.AllPaused() {
						fmt.Println("resuming all workers")
						registry.EachAsync(func(index int64, ctrl testworkflowcontroller.Controller) {
							err := ctrl.Resume(context.Background())
							if err != nil {
								spawn.CreateLogger("worker", descriptions[index], index, params.Count)("warning", "failed to resume", err.Error())
							}
						})
					}
				}
			}()

			// Create channel for execution
			failed := spawn.ExecuteParallel(run, specs, parallelism)
			if failed == 0 {
				fmt.Printf("Successfully finished %d workers.\n", params.Count)
			} else {
				fmt.Printf("Failed to finish %d out of %d expected workers.\n", failed, params.Count)
				os.Exit(1)
			}
		},
	}

	return cmd
}
