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
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-init/data"
	common2 "github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/common"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/env"
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
			infos := make([]string, 0)
			if params.MatrixCount > 1 {
				infos = append(infos, fmt.Sprintf("%d combinations", params.MatrixCount))
			}
			if params.ShardCount > 1 {
				infos = append(infos, fmt.Sprintf("sharded %d times", params.ShardCount))
			}
			parallelism := int64(parallel.Parallelism)
			if parallelism <= 0 {
				parallelism = 1000
			}
			if params.Count > 1 {
				if parallelism < params.Count {
					infos = append(infos, fmt.Sprintf("parallel: %d", parallelism))
				} else if parallelism >= params.Count {
					infos = append(infos, fmt.Sprintf("all in parallel"))
				}
			}
			if params.Count == 1 {
				fmt.Printf("1 instance requested\n")
			} else {
				fmt.Printf("%d instances requested: %s\n", params.Count, strings.Join(infos, ", "))
			}

			// Analyze instances to run
			specs := make([]testworkflowsv1.TestWorkflowSpec, params.Count)
			descriptions := make([]string, params.Count)
			for i := int64(0); i < params.Count; i++ {
				machines := []expressionstcl.Machine{baseMachine, params.MachineAt(i)}
				// Clone the spec
				spec := parallel.DeepCopy()
				err = expressionstcl.Simplify(&spec, machines...)

				// Prepare the transfer
				for ti, t := range spec.Transfer {
					// Parse 'from' clause
					from, err := expressionstcl.EvalTemplate(t.From, machines...)
					ui.ExitOnError(fmt.Sprintf("%d: transfer.%d.from", i, ti), err)

					// Parse 'to' clause
					to := from
					if t.To != "" {
						to, err = expressionstcl.EvalTemplate(t.To, machines...)
						ui.ExitOnError(fmt.Sprintf("%d: transfer.%d.to", i, ti), err)
					}

					// Parse 'files' clause
					patterns := []string{"**/*"}
					if t.Files != nil && !t.Files.Dynamic {
						patterns = t.Files.Static
					} else if t.Files != nil && t.Files.Dynamic {
						patternsExpr, err := expressionstcl.EvalExpression(t.Files.Expression, machines...)
						ui.ExitOnError(fmt.Sprintf("%d: transfer.%d.files", i, ti), err)
						patternsList, err := patternsExpr.Static().SliceValue()
						ui.ExitOnError(fmt.Sprintf("%d: transfer.%d.files", i, ti), err)
						patterns = make([]string, len(patternsList))
						for pi, p := range patternsList {
							if s, ok := p.(string); ok {
								patterns[pi] = s
							} else {
								p, err := json.Marshal(s)
								ui.ExitOnError(fmt.Sprintf("%d: transfer.%d.files.%d", i, ti, pi), err)
								patterns[pi] = string(p)
							}
						}
					}

					entry, err := transferSrv.Include(from, patterns)
					ui.ExitOnError(fmt.Sprintf("%d: transfer.%d", i, ti), err)
					spec.Content.Tarball = append(spec.Content.Tarball, testworkflowsv1.ContentTarball{
						Url:   entry.Url,
						Path:  to,
						Mount: t.Mount,
					})
				}

				// Prepare the workflow to run
				specs[i] = spec.TestWorkflowSpec
				descriptions[i] = spec.Description
			}

			// Initialize transfer server if expected
			if transferSrv.Count() > 0 {
				fmt.Printf("Starting transfer server for %d tarballs...\n", transferSrv.Count())
				if _, err = transferSrv.Listen(); err != nil {
					ui.Fail(errors.Wrap(err, "failed to start transfer server"))
				}
				fmt.Printf("Transfer server started.\n")
			}

			// Validate if there is anything to run
			if len(specs) == 0 {
				fmt.Printf("nothing to run\n")
				os.Exit(0)
			}

			// Send initial output
			for index := range specs {
				data.PrintOutput(env.Ref(), "parallel", ParallelStatus{
					Index:       index,
					Description: descriptions[index],
				})
			}
			descriptions = nil

			// Load Kubernetes client and image inspector
			clientSet := env.Kubernetes()
			inspector := env.ImageInspector()

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
				updates <- Update{index: index}

				// Build internal machine
				id := fmt.Sprintf("%s-%d", env.ExecutionId(), index)
				machine := expressionstcl.NewMachine().
					Register("execution.id", env.ExecutionId()).
					Register("resource.rootId", env.ExecutionId()).
					Register("resource.id", id).
					Register("workflow.name", env.WorkflowName())

				// Build the resources bundle
				scheduledAt := time.Now()
				bundle, err := testworkflowprocessor.NewFullFeatured(inspector).
					Bundle(context.Background(), &testworkflowsv1.TestWorkflow{Spec: *spec}, machine, baseMachine, params.MachineAt(index)) // TODO: params.MachineAt should be limited until sub-parallel/sub-execute
				if err != nil {
					fmt.Printf("%d: failed to prepare resources: %s\n", index, err.Error())
					return false
				}
				defer func() {
					err := testworkflowcontroller.Cleanup(context.Background(), clientSet, env.Namespace(), id)
					if err == nil {
						fmt.Printf("%s: cleaned resources\n", common2.InstanceLabel("worker", index, params.Count))
					} else {
						fmt.Printf("%s: warning: problem cleaning up resources: %s\n", common2.InstanceLabel("worker", index, params.Count), err.Error())
					}
					updates <- Update{index: index, done: true, err: err}
				}()

				// Deploy the resources
				for _, item := range bundle.Secrets {
					_, err = clientSet.CoreV1().Secrets(env.Namespace()).Create(context.Background(), &item, metav1.CreateOptions{})
					if err != nil {
						fmt.Printf("%s: failed to deploy secrets: %s\n", common2.InstanceLabel("worker", index, params.Count), err.Error())
						return false
					}
				}
				for _, item := range bundle.ConfigMaps {
					_, err = clientSet.CoreV1().ConfigMaps(env.Namespace()).Create(context.Background(), &item, metav1.CreateOptions{})
					if err != nil {
						fmt.Printf("%s: failed to deploy config maps: %s\n", common2.InstanceLabel("worker", index, params.Count), err.Error())
						return false
					}
				}
				_, err = clientSet.BatchV1().Jobs(env.Namespace()).Create(context.Background(), &bundle.Job, metav1.CreateOptions{})
				if err != nil {
					fmt.Printf("%s: failed to deploy job: %s\n", common2.InstanceLabel("worker", index, params.Count), err.Error())
					return false
				}

				// Inform about the step structure
				data.PrintOutput(env.Ref(), "parallel", ParallelStatus{Index: int(index), Signature: testworkflowprocessor.MapSignatureListToInternal(bundle.Signature)})

				// Control the execution
				// TODO: Consider aggregated controller to limit number of watchers
				ctrl, err := testworkflowcontroller.New(context.Background(), clientSet, env.Namespace(), id, scheduledAt, testworkflowcontroller.ControllerOptions{
					Timeout: 120 * time.Second,
				})
				if err != nil {
					fmt.Printf("%s: error: failed to deploy job: %s\n", common2.InstanceLabel("worker", index, params.Count), err.Error())
					return false
				}
				controllers[index] = ctrl

				fmt.Printf("%s: created\n", common2.InstanceLabel("worker", index, params.Count))

				// TODO: Add support for watching only result?
				prevStatus := testkube.QUEUED_TestWorkflowStatus
				for v := range ctrl.Watch(context.Background()) {
					if v.Error != nil {
						fmt.Printf("%s: error: %s\n", common2.InstanceLabel("worker", index, params.Count), v.Error.Error())
					} else if v.Value.Result != nil {
						updates <- Update{index: index, result: v.Value.Result}
						if v.Value.Result.Status != nil && *v.Value.Result.Status != prevStatus {
							prevStatus = *v.Value.Result.Status
							fmt.Printf("%s: %s\n", common2.InstanceLabel("worker", index, params.Count), prevStatus)

							if v.Value.Result.IsFinished() {
								data.PrintOutput(env.Ref(), "parallel", ParallelStatus{Index: int(index), Status: prevStatus, Result: v.Value.Result})
								return v.Value.Result.IsPassed()
							} else {
								data.PrintOutput(env.Ref(), "parallel", ParallelStatus{Index: int(index), Status: prevStatus})
							}
						}
					}
				}

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
