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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	commontcl "github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/common"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/spawn"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env/config"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/transfer"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowresolver"
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
	Done        bool          `json:"done,omitempty"`
}

func (s ServiceInfo) AsMap() (v map[string]interface{}) {
	serialized, _ := json.Marshal(s)
	_ = json.Unmarshal(serialized, &v)
	return
}

func NewServicesCmd() *cobra.Command {
	var (
		groupRef      string
		base64Encoded bool
	)
	cmd := &cobra.Command{
		Use:   "services <ref>",
		Short: "Start accompanying service(s)",

		Run: func(cmd *cobra.Command, args []string) {
			// Initialize basic adapters
			baseMachine := spawn.CreateBaseMachine()
			transferSrv := transfer.NewServer(constants.DefaultTransferDirPath, config.IP(), constants.DefaultTransferPort)

			// Validate data
			if groupRef == "" {
				ui.Fail(errors.New("missing required --group for starting the services"))
			}

			// Read the services to start
			services := make(map[string]testworkflowsv1.ServiceSpec)

			if base64Encoded && len(args) > 0 {
				// Decode base64 input. The processor base64-encodes service specs to prevent
				// testworkflow-init from prematurely resolving expressions like {{ matrix.browser.driver }}.
				// We decode here where we have the proper context to evaluate these expressions.
				// Decode the services map
				var servicesMap map[string]json.RawMessage
				err := expressionstcl.DecodeBase64JSON(args[0], &servicesMap)
				ui.ExitOnError("decoding services", err)

				// Parse each service spec
				for name, raw := range servicesMap {
					var svc testworkflowsv1.ServiceSpec
					err := json.Unmarshal(raw, &svc)
					ui.ExitOnError(fmt.Sprintf("parsing service spec for %s", name), err)
					services[name] = svc
				}
			} else {
				// Legacy format: name=spec pairs (kept for backward compatibility)
				for i := range args {
					name, v, found := strings.Cut(args[i], "=")
					if !found {
						ui.Fail(fmt.Errorf("invalid service declaration: %s", args[i]))
					}
					var svc *testworkflowsv1.ServiceSpec
					err := json.Unmarshal([]byte(v), &svc)
					ui.ExitOnError("parsing service spec", err)
					services[name] = *svc
				}
			}

			// Apply default service account and initialize details for each service
			for name := range services {
				if services[name].Pod == nil {
					svc := services[name]
					svc.Pod = &testworkflowsv1.PodConfig{}
					services[name] = svc
				}
				if services[name].Pod.ServiceAccountName == "" {
					services[name].Pod.ServiceAccountName = "{{internal.serviceaccount.default}}"
				}

				// Initialize empty array of details for each of the services
				instructions.PrintHintDetails(config.Ref(), data.ServicesPrefix+name, []ServiceState{})
			}

			// Analyze instances to run
			state := make(map[string][]ServiceState)
			instances := make([]ServiceInstance, 0)
			namespaces := make([]string, 0)
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
				svcNamespaces := make([]string, params.Count)
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
						Pvcs: svcSpec.Pvcs,
					}
					spec.Steps[0].Run.ContainerConfig = testworkflowsv1.ContainerConfig{}
					spec.Container.Env = testworkflowresolver.DedupeEnvVars(append(config.Config().Execution.GlobalEnv, spec.Container.Env...))

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
					svcNamespaces[index] = config.Namespace()
					if spec.Job != nil && spec.Job.Namespace != "" {
						svcNamespaces[index] = spec.Job.Namespace
					}

					// Save the timeout
					if svcSpec.Timeout != "" {
						v, err := expressions.EvalTemplate(svcSpec.Timeout, machines...)
						ui.ExitOnError(fmt.Sprintf("%s: %d: error: timeout expression", commontcl.ServiceLabel(name), index), err)
						d, err := time.ParseDuration(strings.ReplaceAll(v, " ", ""))
						ui.ExitOnError(fmt.Sprintf("%s: %d: error: invalid timeout: %s", commontcl.ServiceLabel(name), index, v), err)
						svcInstances[index].Timeout = &d
					}
				}
				instances = append(instances, svcInstances...)
				namespaces = append(namespaces, svcNamespaces...)

				// Update the state
				state[name] = make([]ServiceState, len(svcInstances))
				for i := range svcInstances {
					state[name][i].Description = svcInstances[i].Description
				}
				instructions.PrintHintDetails(config.Ref(), data.ServicesPrefix+name, state)
			}

			// Inform about each service instance
			for _, instance := range instances {
				instructions.PrintOutput(config.Ref(), "service", ServiceInfo{
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

			run := func(_ int64, _ string, instance *ServiceInstance) bool {
				info := ServiceInfo{
					Group:       groupRef,
					Index:       instance.Index,
					Name:        instance.Name,
					Description: instance.Description,
					Status:      ServiceStatusQueued,
				}
				index := instance.Index

				// Determine the namespace
				namespace := config.Namespace()
				if instance.Spec.Job != nil && instance.Spec.Job.Namespace != "" {
					namespace = instance.Spec.Job.Namespace
				}

				params := svcParams[instance.Name]
				log := spawn.CreateLogger(instance.Name, instance.Description, index, params.Count)

				// Build the configuration
				cfg := *config.Config()
				cfg.Resource = spawn.CreateResourceConfig(instance.Name+"-", index) // TODO: Think if it should be there
				cfg.Worker.Namespace = namespace
				machine := expressions.CombinedMachines(
					testworkflowconfig.CreateResourceMachine(&cfg.Resource),
					testworkflowconfig.CreateWorkerMachine(&cfg.Worker),
					baseMachine,
					testworkflowconfig.CreatePvcMachine(cfg.Execution.PvcNames),
					params.MachineAt(index),
				)

				// Simplify the workflow
				_ = expressions.Simplify(&instance.Spec, machine)

				// Build the resources bundle
				scheduledAt := time.Now()
				result, err := spawn.ExecutionWorker().Service(context.Background(), executionworkertypes.ServiceRequest{
					ResourceId:     cfg.Resource.Id,
					GroupId:        groupRef,
					Execution:      cfg.Execution,
					Workflow:       testworkflowsv1.TestWorkflow{ObjectMeta: metav1.ObjectMeta{Name: cfg.Workflow.Name, Labels: cfg.Workflow.Labels}, Spec: instance.Spec},
					ScheduledAt:    &scheduledAt,
					RestartPolicy:  string(instance.RestartPolicy),
					ReadinessProbe: common.MapPtr(instance.ReadinessProbe, testworkflows.MapProbeKubeToAPI),

					ControlPlane:        cfg.ControlPlane,
					ArtifactsPathPrefix: cfg.Resource.FsPrefix,
				})
				if err != nil {
					fmt.Printf("%d: failed to prepare resources: %s\n", index, err.Error())
					return false
				}

				// Compute the bundle instructions
				signatureSeq := stage.MapSignatureToSequence(stage.MapSignatureList(result.Signature))
				mainRef := signatureSeq[len(signatureSeq)-1].Ref()

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

				// TODO: Use more lightweight notifications
				notifications := spawn.ExecutionWorker().StatusNotifications(ctx, cfg.Resource.Id, executionworkertypes.StatusNotificationsOptions{
					Hints: executionworkertypes.Hints{
						Namespace:   result.Namespace,
						Signature:   result.Signature,
						ScheduledAt: common.Ptr(scheduledAt),
					},
				})
				if notifications.Err() != nil {
					log("error", "failed to connect to the service", notifications.Err().Error())
					return false
				}
				log("created")

				scheduled := false
				started := false
				ready := instance.ReadinessProbe == nil
				for v := range notifications.Channel() {
					// Inform about the node assignment
					if !scheduled && v.NodeName != "" {
						scheduled = true
						log(fmt.Sprintf("assigned to %s node", ui.LightBlue(v.NodeName)))
					}

					if state[instance.Name][index].Ip == "" && v.PodIp != "" {
						state[instance.Name][index].Ip = v.PodIp
						log(fmt.Sprintf("assigned to %s IP", ui.LightBlue(v.PodIp)))
						info.Status = ServiceStatusRunning
						instructions.PrintOutput(config.Ref(), "service", info)
					}

					ready = v.Ready
					if !started && v.Ref == mainRef && state[instance.Name][index].Ip != "" {
						started = true
						if instance.ReadinessProbe == nil {
							log("container started")
						} else {
							log("container started, waiting for readiness")
						}
					}

					if started && ready {
						break
					}
				}
				ctxCancel()
				if !ready {
					log("container did not reach readiness")
					info.Status = ServiceStatusFailed
				} else {
					log("container ready")
					info.Status = ServiceStatusReady
				}
				if notifications.Err() != nil {
					log("error", notifications.Err().Error())
				}

				// Fail if the container has not started
				if !started {
					info.Status = ServiceStatusFailed
					log("container failed")
					instructions.PrintOutput(config.Ref(), "service", info)
					return false
				}

				instructions.PrintOutput(config.Ref(), "service", info)

				return ready
			}

			// Start all the services
			failed := spawn.ExecuteParallel(run, instances, namespaces, int64(len(instances)))

			// Inform about the services state
			for k := range state {
				instructions.PrintHintDetails(config.Ref(), data.ServicesPrefix+k, state[k])
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
	cmd.Flags().BoolVar(&base64Encoded, "base64", false, "input is base64 encoded")

	return cmd
}
