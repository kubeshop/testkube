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
	"math"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	commontcl "github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/common"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/spawn"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/transfer"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowcontroller"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/presets"
	"github.com/kubeshop/testkube/pkg/ui"
)

type ServiceInstance struct {
	Index          int64
	Name           string
	Description    string
	Timeout        *time.Duration
	RestartPolicy  corev1.RestartPolicy
	ReadinessProbe *corev1.Probe
	Spec           testworkflowsv1.TestWorkflowSpec
}

type ServiceState struct {
	Ip          string `json:"ip"`
	Description string `json:"description"`
}

type ServiceStatus string

const (
	ServiceStatusQueued  ServiceStatus = "queued"
	ServiceStatusRunning ServiceStatus = "running"
	ServiceStatusReady   ServiceStatus = "passed"
	ServiceStatusFailed  ServiceStatus = "failed"
)

type ServiceInfo struct {
	Group       string        `json:"group"`
	Index       int64         `json:"index"`
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	Logs        string        `json:"logs,omitempty"`
	Status      ServiceStatus `json:"status,omitempty"`
}

func NewServicesCmd() *cobra.Command {
	var (
		groupRef string
	)
	cmd := &cobra.Command{
		Use:   "services <ref>",
		Short: "Start accompanying service(s)",

		Run: func(cmd *cobra.Command, pairs []string) {
			// Initialize basic adapters
			baseMachine := spawn.CreateBaseMachine()
			inspector := env.ImageInspector()
			transferSrv := transfer.NewServer(constants.DefaultTransferDirPath, env.IP(), constants.DefaultTransferPort)

			// Validate data
			if groupRef == "" {
				ui.Fail(errors.New("missing required --group for starting the services"))
			}

			// Read the services to start
			services := make(map[string]testworkflowsv1.ServiceSpec, len(pairs))
			for i := range pairs {
				name, v, found := strings.Cut(pairs[i], "=")
				if !found {
					ui.Fail(fmt.Errorf("invalid service declaration: %s", pairs[i]))
				}
				var svc *testworkflowsv1.ServiceSpec
				err := json.Unmarshal([]byte(v), &svc)
				ui.ExitOnError("parsing service spec", err)
				services[name] = *svc

				// Apply default service account
				if services[name].Pod == nil {
					svc := services[name]
					svc.Pod = &testworkflowsv1.PodConfig{}
					services[name] = svc
				}
				if services[name].Pod.ServiceAccountName == "" {
					services[name].Pod.ServiceAccountName = "{{internal.serviceaccount.default}}"
				}

				// Initialize empty array of details for each of the services
				data.PrintHintDetails(env.Ref(), fmt.Sprintf("services.%s", name), []ServiceState{})
			}

			// Analyze instances to run
			state := make(map[string][]ServiceState)
			instances := make([]ServiceInstance, 0)
			svcParams := make(map[string]*commontcl.ParamsSpec)
			for name, svc := range services {
				// Resolve the params
				params, err := commontcl.GetParamsSpec(svc.Matrix, svc.Shards, svc.Count, svc.MaxCount, baseMachine)
				ui.ExitOnError(fmt.Sprintf("%s: compute matrix and sharding", commontcl.ServiceLabel(name)), err)
				svcParams[name] = params

				// Ignore no instances
				if params.Count == 0 {
					fmt.Printf("%s: 0 instances requested (combinations=%d, count=%d), skipping\n", commontcl.ServiceLabel(name), params.MatrixCount, params.ShardCount)
					continue
				}

				// Print information about the number of params
				fmt.Printf("%s: %s\n", commontcl.ServiceLabel(name), params.String(math.MaxInt64))

				svcInstances := make([]ServiceInstance, params.Count)
				for index := int64(0); index < params.Count; index++ {
					machines := []expressions.Machine{baseMachine, params.MachineAt(index)}

					// Clone the spec
					svcSpec := svc.DeepCopy()
					err = expressions.Simplify(&svcSpec, machines...)
					ui.ExitOnError(fmt.Sprintf("%s: %d: error", commontcl.ServiceLabel(name), index), err)

					// Build the spec
					spec := testworkflowsv1.TestWorkflowSpec{
						TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
							Content:   svcSpec.Content,
							Container: common.Ptr(svcSpec.ContainerConfig),
							Pod:       svcSpec.Pod,
						},
						Steps: []testworkflowsv1.Step{
							{StepOperations: testworkflowsv1.StepOperations{Run: common.Ptr(svcSpec.StepRun)}},
						},
					}
					spec.Steps[0].Run.ContainerConfig = testworkflowsv1.ContainerConfig{}

					// Transfer the data
					if spec.Content == nil {
						spec.Content = &testworkflowsv1.Content{}
					}
					tarballs, err := spawn.ProcessTransfer(transferSrv, svcSpec.Transfer, machines...)
					ui.ExitOnError(fmt.Sprintf("%s: %d: error: transfer", commontcl.ServiceLabel(name), index), err)
					spec.Content.Tarball = append(spec.Content.Tarball, tarballs...)

					// Save the instance
					svcInstances[index] = ServiceInstance{
						Index:          index,
						Name:           name,
						Description:    svcSpec.Description,
						RestartPolicy:  corev1.RestartPolicy(svcSpec.RestartPolicy),
						ReadinessProbe: svcSpec.ReadinessProbe,
						Spec:           spec,
					}

					// Save the timeout
					if svcSpec.Timeout != "" {
						v, err := expressions.EvalTemplate(svcSpec.Timeout, machines...)
						ui.ExitOnError(fmt.Sprintf("%s: %d: error: timeout expression", commontcl.ServiceLabel(name), index), err)
						d, err := time.ParseDuration(strings.ReplaceAll(v, " ", ""))
						ui.ExitOnError(fmt.Sprintf("%s: %d: error: invalid timeout: %s:", commontcl.ServiceLabel(name), index, v), err)
						svcInstances[index].Timeout = &d
					}
				}
				instances = append(instances, svcInstances...)

				// Update the state
				state[name] = make([]ServiceState, len(svcInstances))
				for i := range svcInstances {
					state[name][i].Description = svcInstances[i].Description
				}
				data.PrintHintDetails(env.Ref(), fmt.Sprintf("services.%s", name), state)
			}

			// Inform about each service instance
			for _, instance := range instances {
				data.PrintOutput(env.Ref(), "service", ServiceInfo{
					Group:       groupRef,
					Index:       instance.Index,
					Name:        instance.Name,
					Description: instance.Description,
					Status:      ServiceStatusQueued,
				})
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
				if _, err := transferSrv.Listen(); err != nil {
					ui.Fail(errors.Wrap(err, "failed to start transfer server"))
				}
				fmt.Printf("Transfer server started.\n")
			}

			// Validate if there is anything to run
			if len(instances) == 0 {
				ui.SuccessAndExit("nothing to run")
			}

			run := func(_ int64, instance *ServiceInstance) bool {
				info := ServiceInfo{
					Group:       groupRef,
					Index:       instance.Index,
					Name:        instance.Name,
					Description: instance.Description,
					Status:      ServiceStatusQueued,
				}
				index := instance.Index
				id, machine := spawn.CreateExecutionMachine(instance.Name+"-", index)
				params := svcParams[instance.Name]
				log := spawn.CreateLogger(instance.Name, instance.Description, index, params.Count)
				clientSet := env.Kubernetes()

				// Build the resources bundle
				scheduledAt := time.Now()
				bundle, err := presets.NewPro(inspector).
					Bundle(context.Background(), &testworkflowsv1.TestWorkflow{Spec: instance.Spec}, machine, baseMachine, params.MachineAt(index))
				if err != nil {
					log("error", "failed to build the service", err.Error())
					return false
				}
				ui.ExitOnError(fmt.Sprintf("%s: %d: failed to prepare resources", commontcl.InstanceLabel(instance.Name, index, params.Count), index), err)

				// Apply the service specific data
				// TODO: Handle RestartPolicy: Always?
				if instance.RestartPolicy == "Never" {
					bundle.Job.Spec.BackoffLimit = common.Ptr(int32(0))
					bundle.Job.Spec.Template.Spec.RestartPolicy = "Never"
				} else {
					// TODO: Throw errors from the pod containers? Atm it will just end with "Success"...
					bundle.Job.Spec.BackoffLimit = nil
					bundle.Job.Spec.Template.Spec.RestartPolicy = "OnFailure"
				}
				bundle.Job.Spec.Template.Spec.Containers[0].ReadinessProbe = instance.ReadinessProbe

				// Add group recognition
				testworkflowprocessor.AnnotateGroupId(&bundle.Job, groupRef)
				for i := range bundle.ConfigMaps {
					testworkflowprocessor.AnnotateGroupId(&bundle.ConfigMaps[i], groupRef)
				}
				for i := range bundle.Secrets {
					testworkflowprocessor.AnnotateGroupId(&bundle.Secrets[i], groupRef)
				}

				// Compute the bundle instructions
				namespace := bundle.Job.Namespace
				if namespace == "" {
					namespace = env.Namespace()
				}

				var instructions [][]actiontypes.Action
				mainRef := ""
				err = json.Unmarshal([]byte(bundle.Job.Spec.Template.Annotations[constants.SpecAnnotationName]), &instructions)
				if err != nil {
					panic(fmt.Sprintf("invalid instructions: %v", err))
				} else {
					lastGroup := instructions[len(instructions)-1]
					for i := range lastGroup {
						if lastGroup[i].Type() == lite.ActionTypeStart {
							mainRef = *lastGroup[i].Start
						}
					}
				}

				// Deploy the resources
				// TODO: Avoid using Job
				err = bundle.Deploy(context.Background(), clientSet, namespace)
				if err != nil {
					log("problem deploying", err.Error())
					return false
				}

				// Handle timeout
				timeoutCtx, timeoutCtxCancel := context.WithCancel(context.Background())
				defer timeoutCtxCancel()
				if instance.Timeout != nil {
					go func() {
						select {
						case <-timeoutCtx.Done():
						case <-time.After(*instance.Timeout):
							log("timed out", instance.Timeout.String()+" elapsed")
							timeoutCtxCancel()
						}
					}()
				}

				// Control the execution
				// TODO: Consider aggregated controller to limit number of watchers
				ctx, ctxCancel := context.WithCancel(timeoutCtx)
				defer ctxCancel()
				ctrl, err := testworkflowcontroller.New(ctx, clientSet, namespace, id, scheduledAt, testworkflowcontroller.ControllerOptions{
					Timeout: spawn.ControllerTimeout,
				})
				if err != nil {
					log("error", "failed to connect to the job", err.Error())
					return false
				}
				log("created")
				scheduled := false
				started := false
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

					if state[instance.Name][index].Ip == "" && v.PodIP != "" {
						state[instance.Name][index].Ip = v.PodIP
						log(fmt.Sprintf("assigned to %s IP", ui.LightBlue(v.PodIP)))
						info.Status = ServiceStatusRunning
						data.PrintOutput(env.Ref(), "service", info)
					}

					if v.Current == mainRef {
						started = true
						if instance.ReadinessProbe == nil {
							log("container started")
						} else {
							log("container started, waiting for readiness")
						}
						ctxCancel()
						break
					}
				}
				ctrl.StopController()

				// Fail if the container has not started
				if !started {
					info.Status = ServiceStatusFailed
					log("container failed")
					data.PrintOutput(env.Ref(), "service", info)
					return false
				}

				// Watch for container readiness
				ready := instance.ReadinessProbe == nil
				if !ready {
					podWatcher := testworkflowcontroller.WatchMainPod(timeoutCtx, clientSet, namespace, id, 0)
					for pod := range podWatcher.Channel() {
						if pod.Error != nil {
							log("error", pod.Error.Error())
							continue
						}

						ready = pod.Value.Status.ContainerStatuses[0].Ready
						if ready {
							break
						}
					}
				}

				if !ready {
					log("container did not reach readiness")
					info.Status = ServiceStatusFailed
				} else {
					log("container ready")
					info.Status = ServiceStatusReady
				}
				data.PrintOutput(env.Ref(), "service", info)

				return ready
			}

			// Start all the services
			failed := spawn.ExecuteParallel(run, instances, int64(len(instances)))

			// Inform about the services state
			for k := range state {
				data.PrintHintDetails(env.Ref(), fmt.Sprintf("services.%s", k), state[k])
			}

			// Notify the results
			if failed == 0 {
				fmt.Printf("Successfully started %d workers.\n", len(instances))
			} else {
				fmt.Printf("Failed to start %d out of %d expected workers.\n", failed, len(instances))
				os.Exit(1)
			}
		},
	}

	cmd.Flags().StringVarP(&groupRef, "group", "g", "", "services group reference")

	return cmd
}
