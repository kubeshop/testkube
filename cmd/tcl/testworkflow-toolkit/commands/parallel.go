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
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/artifacts"
	common2 "github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/common"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/spawn"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/transfer"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowcontroller"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor/constants"
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
			baseMachine := expressionstcl.CombinedMachines(
				data.GetBaseTestWorkflowMachine(),
				expressionstcl.NewMachine().RegisterStringMap("internal", map[string]string{
					"storage.url":        env.Config().ObjectStorage.Endpoint,
					"storage.accessKey":  env.Config().ObjectStorage.AccessKeyID,
					"storage.secretKey":  env.Config().ObjectStorage.SecretAccessKey,
					"storage.region":     env.Config().ObjectStorage.Region,
					"storage.bucket":     env.Config().ObjectStorage.Bucket,
					"storage.token":      env.Config().ObjectStorage.Token,
					"storage.ssl":        strconv.FormatBool(env.Config().ObjectStorage.Ssl),
					"storage.skipVerify": strconv.FormatBool(env.Config().ObjectStorage.SkipVerify),
					"storage.certFile":   env.Config().ObjectStorage.CertFile,
					"storage.keyFile":    env.Config().ObjectStorage.KeyFile,
					"storage.caFile":     env.Config().ObjectStorage.CAFile,

					"cloud.enabled":         strconv.FormatBool(env.Config().Cloud.ApiKey != ""),
					"cloud.api.key":         env.Config().Cloud.ApiKey,
					"cloud.api.tlsInsecure": strconv.FormatBool(env.Config().Cloud.TlsInsecure),
					"cloud.api.skipVerify":  strconv.FormatBool(env.Config().Cloud.SkipVerify),
					"cloud.api.url":         env.Config().Cloud.Url,

					"dashboard.url":   env.Config().System.DashboardUrl,
					"api.url":         env.Config().System.ApiUrl,
					"namespace":       env.Namespace(),
					"defaultRegistry": env.Config().System.DefaultRegistry,

					"images.init":                env.Config().Images.Init,
					"images.toolkit":             env.Config().Images.Toolkit,
					"images.persistence.enabled": strconv.FormatBool(env.Config().Images.InspectorPersistenceEnabled),
					"images.persistence.key":     env.Config().Images.InspectorPersistenceCacheKey,
				}),
			)

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
			params, err := common2.GetParamsSpec(parallel.Matrix, parallel.Shards, parallel.Count, parallel.MaxCount, baseMachine)
			ui.ExitOnError("compute matrix and sharding", err)

			// Clean up universal copy
			parallel.StepExecuteStrategy = testworkflowsv1.StepExecuteStrategy{}
			if len(parallel.Transfer) > 0 && parallel.Content == nil {
				parallel.Content = &testworkflowsv1.Content{}
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
			for i := int64(0); i < params.Count; i++ {
				machines := []expressionstcl.Machine{baseMachine, params.MachineAt(i)}
				// Clone the spec
				spec := parallel.DeepCopy()
				err = expressionstcl.Simplify(&spec, machines...)
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
			clientSet := env.Kubernetes()
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
			controllers := map[int64]testworkflowcontroller.Controller{}
			run := func(index int64, spec *testworkflowsv1.TestWorkflowSpec) bool {
				log := spawn.CreateLogger("worker", descriptions[index], index, params.Count)
				id, machine := spawn.CreateExecutionMachine(index)

				updates <- Update{index: index}

				// Build the resources bundle
				scheduledAt := time.Now()
				bundle, err := testworkflowprocessor.NewFullFeatured(inspector).
					Bundle(context.Background(), &testworkflowsv1.TestWorkflow{Spec: *spec}, machine, baseMachine, params.MachineAt(index))
				if err != nil {
					fmt.Printf("%d: failed to prepare resources: %s\n", index, err.Error())
					return false
				}

				// Compute the namespace where it's deployed to
				namespace := bundle.Job.Namespace
				if namespace == "" {
					namespace = env.Namespace()
				}

				defer func() {
					// Save logs
					logsFilePath, err := spawn.SaveLogs(context.Background(), clientSet, storage, namespace, id, index)
					if err == nil {
						data.PrintOutput(env.Ref(), "parallel", ParallelStatus{Index: int(index), Logs: storage.FullPath(logsFilePath)})
						log("saved logs")
					} else {
						log("warning", "problem saving the logs", err.Error())
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

				// Deploy the resources
				err = bundle.Deploy(context.Background(), clientSet, namespace)
				if err != nil {
					log("problem deploying", err.Error())
					return false
				}

				// Inform about the step structure
				sig := testworkflowprocessor.MapSignatureListToInternal(bundle.Signature)
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
				controllers[index] = ctrl
				ctx, ctxCancel := context.WithCancel(context.Background())
				log("created")

				prevStatus := testkube.QUEUED_TestWorkflowStatus
				prevStep := ""
				scheduled := false
				for v := range ctrl.Watch(ctx) {
					// Handle error
					if v.Error != nil {
						log("error", v.Error.Error())
						continue
					}

					// Inform about the node assignment
					if !scheduled {
						nodeName, err := ctrl.NodeName(ctx)
						if err == nil {
							scheduled = true
							log(fmt.Sprintf("assigned to %s node", ui.LightBlue(nodeName)))
						}
					}

					// Handle result change
					if v.Value.Result != nil {
						updates <- Update{index: index, result: v.Value.Result}
						current := v.Value.Result.Current(sig)
						status := testkube.QUEUED_TestWorkflowStatus
						if v.Value.Result.Status != nil {
							status = *v.Value.Result.Status
						}

						if status != prevStatus {
							log(string(status))
						}

						if v.Value.Result.IsFinished() {
							data.PrintOutput(env.Ref(), "parallel", ParallelStatus{Index: int(index), Status: status, Result: v.Value.Result})
							ctxCancel()
							return v.Value.Result.IsPassed()
						} else if status != prevStatus || current != prevStep {
							prevStatus = status
							prevStep = current
							data.PrintOutput(env.Ref(), "parallel", ParallelStatus{Index: int(index), Status: status, Current: current})
						}
					}
				}

				ctxCancel()
				return false
			}

			// Orchestrate resume
			go func() {
				statuses := map[int64]Update{}
				for update := range updates {
					statuses[update.index] = update

					// Delete obsolete data
					if update.done || update.err != nil {
						if _, ok := controllers[update.index]; ok {
							controllers[update.index].StopController()
						}
						delete(controllers, update.index)
						delete(statuses, update.index)
					}

					// Determine status
					total := len(statuses)
					paused := 0
					for _, u := range statuses {
						if u.result != nil && u.result.Status != nil && *u.result.Status == testkube.PAUSED_TestWorkflowStatus {
							paused++
						}
					}

					// Resume all at once
					if total != 0 && total == paused {
						fmt.Println("resuming all workers")
						var wg sync.WaitGroup
						wg.Add(paused)
						for index := range statuses {
							go func(index int64) {
								err := controllers[index].Resume(context.Background())
								if err != nil {
									fmt.Printf("%s: warning: failed to resume: %s\n", common2.InstanceLabel("worker", index, params.Count), err.Error())
								}
								wg.Done()
							}(index)
						}
						wg.Wait()
					}
				}
			}()

			// Create channel for execution
			var wg sync.WaitGroup
			wg.Add(int(params.Count))
			ch := make(chan struct{}, parallelism)
			success := atomic.Int64{}

			// Execute all operations
			for index := range specs {
				ch <- struct{}{}
				go func(index int) {
					if run(int64(index), &specs[index]) {
						success.Add(1)
					}
					<-ch
					wg.Done()
				}(index)
			}
			wg.Wait()

			if success.Load() == params.Count {
				fmt.Printf("Successfully finished %d workers.\n", params.Count)
			} else {
				fmt.Printf("Failed to finish %d out of %d expected workers.\n", params.Count-success.Load(), params.Count)
				os.Exit(1)
			}
		},
	}

	return cmd
}
