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

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	commontcl "github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/common"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/spawn"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/artifacts"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env/config"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/transfer"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/ui"
)

const (
	ResumeRetryOnFailureDelay = 300 * time.Millisecond
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

func (s ParallelStatus) AsMap() (v map[string]interface{}) {
	serialized, _ := json.Marshal(s)
	_ = json.Unmarshal(serialized, &v)
	return
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
			transferSrv := transfer.NewServer(constants.DefaultTransferDirPath, config.IP(), constants.DefaultTransferPort)

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
			namespaces := make([]string, params.Count)
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

				// Determine the namespace
				namespace := config.Namespace()
				if spec.Job != nil && spec.Job.Namespace != "" {
					namespace = spec.Job.Namespace
				}

				// Prepare the workflow to run
				specs[i] = spec.TestWorkflowSpec
				namespaces[i] = namespace
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
				instructions.PrintOutput(config.Ref(), "parallel", ParallelStatus{
					Index:       index,
					Description: descriptions[index],
				})
			}

			// Load Kubernetes client and image inspector
			storage, err := artifacts.InternalStorage()
			if err != nil {
				ui.Failf("could not create internal storage client: %v", err)
			}

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
			run := func(index int64, namespace string, spec *testworkflowsv1.TestWorkflowSpec) bool {
				log := spawn.CreateLogger("worker", descriptions[index], index, params.Count)

				// Build the configuration
				cfg := *config.Config()
				cfg.Resource = spawn.CreateResourceConfig(config.Ref()+"-", index) // TODO: Think if it should be there
				cfg.Worker.Namespace = namespace
				machine := expressions.CombinedMachines(
					testworkflowconfig.CreateResourceMachine(&cfg.Resource),
					testworkflowconfig.CreateWorkerMachine(&cfg.Worker),
					baseMachine,
					testworkflowconfig.CreatePvcMachine(cfg.Execution.PvcNames),
					params.MachineAt(index),
				)

				// Simplify the workflow
				_ = expressions.Simplify(&spec, machine)

				// Register that there is some operation queued
				registry.SetStatus(index, nil)

				updates <- Update{index: index}

				// Deploy the resource
				scheduledAt := time.Now()
				result, err := spawn.ExecutionWorker().Execute(context.Background(), executionworkertypes.ExecuteRequest{
					ResourceId:          cfg.Resource.Id,
					Execution:           cfg.Execution,
					Workflow:            testworkflowsv1.TestWorkflow{ObjectMeta: metav1.ObjectMeta{Name: cfg.Workflow.Name, Labels: cfg.Workflow.Labels}, Spec: *spec},
					ScheduledAt:         &scheduledAt,
					ControlPlane:        cfg.ControlPlane,
					ArtifactsPathPrefix: spawn.CreateResourceConfig("", index).FsPrefix, // Omit duplicated reference for FS prefix
				})
				if err != nil {
					fmt.Printf("%d: failed to prepare resources: %s\n", index, err.Error())
					registry.Destroy(index)
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
						logsFilePath, err := spawn.SaveLogs(context.Background(), storage, namespace, cfg.Resource.Id, "", index)
						if err == nil {
							instructions.PrintOutput(config.Ref(), "parallel", ParallelStatus{Index: int(index), Logs: storage.FullPath(logsFilePath)})
							log("saved logs")
						} else {
							log("warning", "problem saving the logs", err.Error())
						}
					}

					// Clean up
					err = spawn.ExecutionWorker().Destroy(context.Background(), cfg.Resource.Id, executionworkertypes.DestroyOptions{
						Namespace: namespace,
					})
					if err == nil {
						log("cleaned resources")
					} else {
						log("warning", "problem cleaning up resources", err.Error())
					}
					updates <- Update{index: index, done: true, err: err}
				}()

				// Inform about the step structure
				instructions.PrintOutput(config.Ref(), "parallel", ParallelStatus{Index: int(index), Signature: result.Signature})

				// Control the execution
				// TODO: Consider aggregated controller to limit number of watchers
				ctx, ctxCancel := context.WithCancel(context.Background())
				defer ctxCancel()

				// TODO: Use more lightweight notifications
				notifications := spawn.ExecutionWorker().StatusNotifications(ctx, cfg.Resource.Id, executionworkertypes.StatusNotificationsOptions{
					Hints: executionworkertypes.Hints{
						Namespace:   result.Namespace,
						Signature:   result.Signature,
						ScheduledAt: common.Ptr(scheduledAt),
					},
				})
				if notifications.Err() != nil {
					log("error", "failed to connect to the parallel worker", notifications.Err().Error())
					return false
				}
				log("created")

				prevStatus := testkube.QUEUED_TestWorkflowStatus
				prevStep := ""
				scheduled := false
				ipAssigned := false
				for v := range notifications.Channel() {
					// Inform about the node assignment
					if !scheduled && v.NodeName != "" {
						scheduled = true
						log(fmt.Sprintf("assigned to %s node", ui.LightBlue(v.NodeName)))
					}

					// Inform about the IP assignment
					if !ipAssigned && v.PodIp != "" {
						ipAssigned = true
						registry.SetAddress(index, v.PodIp)
					}

					// Save the last result
					step := prevStep
					status := prevStatus
					if v.Result != nil {
						lastResult = *v.Result
						if lastResult.Status != nil {
							status = *lastResult.Status
						} else {
							status = testkube.QUEUED_TestWorkflowStatus
						}
						step = lastResult.Current(result.Signature)
					}

					// Handle result change
					if status != prevStatus || step != prevStep {
						if status != prevStatus {
							log(string(status))
						}
						updates <- Update{index: index, result: v.Result}
						prevStep = step
						prevStatus = status

						if lastResult.IsFinished() {
							instructions.PrintOutput(config.Ref(), "parallel", ParallelStatus{Index: int(index), Status: status, Result: v.Result})
							return v.Result.IsPassed()
						} else {
							instructions.PrintOutput(config.Ref(), "parallel", ParallelStatus{Index: int(index), Status: status, Current: step})
						}
					}
				}
				if notifications.Err() != nil {
					log("error", notifications.Err().Error())
				}

				// Fallback in case there is a problem with finishing
				log("could not determine status of the worker - aborting")
				instructions.PrintOutput(config.Ref(), "parallel", ParallelStatus{Index: int(index), Status: testkube.ABORTED_TestWorkflowStatus, Result: &lastResult})
				log(string(testkube.ABORTED_TestWorkflowStatus))
				lastResult.Status = common.Ptr(testkube.ABORTED_TestWorkflowStatus)
				if lastResult.FinishedAt.IsZero() {
					lastResult.FinishedAt = time.Now().UTC()
				}
				updates <- Update{index: index, result: lastResult.Clone()}

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
						ids := common.MapSlice(registry.Indexes(), func(index int64) string {
							return spawn.GetResourceId(config.Ref()+"-", index)
						})

						errs := spawn.ExecutionWorker().ResumeMany(context.Background(), ids, executionworkertypes.ControlOptions{})
						for _, err := range errs {
							if err.Id == "" {
								fmt.Printf("warn: %s\n", err.Error)
							} else {
								_, index := spawn.GetServiceByResourceId(err.Id)
								spawn.CreateLogger("worker", descriptions[index], index, params.Count)("warning", "failed to resume", err.Error.Error())
								_ = spawn.ExecutionWorker().Abort(context.Background(), err.Id, executionworkertypes.DestroyOptions{
									Namespace: namespaces[index],
								})
							}
						}
					}
				}
			}()

			// Create channel for execution
			failed := spawn.ExecuteParallel(run, specs, namespaces, parallelism)

			// Wait for the results
			if failed == 0 {
				fmt.Printf("Successfully finished %d workers.\n", params.Count)
				os.Exit(0)
			} else {
				fmt.Printf("Failed to finish %d out of %d expected workers.\n", failed, params.Count)
				os.Exit(1)
			}
		},
	}

	return cmd
}
