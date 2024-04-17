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
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/artifacts"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/spawn"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/ui"
)

const MaxParallelism = 1000

func NewSpawnCmd() *cobra.Command {
	var (
		instructionsStr []string
		longRunning     bool
	)

	cmd := &cobra.Command{
		Use:   "spawn",
		Short: "Spawn assisting pods",
		Args:  cobra.ExactArgs(1),

		Run: func(cmd *cobra.Command, args []string) {
			podsRef := args[0]

			// Initialize internal machine
			baseMachine := data.GetBaseTestWorkflowMachine()

			// Initialize state
			states := make(map[string][]spawn.ServiceState)
			var statesMu sync.Mutex
			saveState := func() {
				if longRunning {
					for k := range states {
						data.PrintHintDetails(env.Ref(), fmt.Sprintf("services.%s", k), states[k])
					}
				}
			}
			getState := func(name string, index int64) spawn.ServiceState {
				defer statesMu.Unlock()
				statesMu.Lock()
				return states[name][index]
			}
			updateState := func(name string, index int64, pod *corev1.Pod) {
				defer statesMu.Unlock()
				statesMu.Lock()
				state := states[name][index]
				state.Update(pod)
				states[name][index] = state
			}
			fail := func(format string, a ...any) {
				saveState()
				fmt.Printf(format+"\n", a...)
				os.Exit(1)
			}

			// Unmarshal the instructions
			instructions := make(map[string]testworkflowsv1.SpawnInstructionBase, len(instructionsStr))
			for i := range instructionsStr {
				name, instruction, _ := strings.Cut(instructionsStr[i], "=")
				var v testworkflowsv1.SpawnInstructionBase
				err := json.Unmarshal([]byte(instruction), &v)
				instructions[name] = v
				if err != nil {
					fail("Problem processing the assisting pod spec: %s\n%s", err.Error(), instructionsStr[i])
				}
			}

			// Initialize list of services
			total := int64(0)
			success := atomic.Int64{}
			services := make([]spawn.Service, 0)
			servicesMap := make(map[string]spawn.Service)
			serviceLocks := make([][]sync.Mutex, 0)
			serviceLocksMap := make(map[string]*[]sync.Mutex)

			// Resolve the instructions
			for k, instruction := range instructions {
				// Apply defaults
				if longRunning && instruction.Ready == "" {
					instruction.Ready = "ready && containerStarted"
				}
				if !longRunning && instruction.Pod.Spec.RestartPolicy == "" {
					instruction.Pod.Spec.RestartPolicy = corev1.RestartPolicyNever
				}

				// Build the service
				svc, err := spawn.FromInstruction(k, instruction, baseMachine)
				svcCombinations := svc.Params.MatrixCount
				svcTotal := svc.Params.Count
				if err != nil {
					fail("%s: %s", spawn.ServiceLabel(k), err.Error())
				}

				// Apply empty state
				states[k] = make([]spawn.ServiceState, svcTotal)

				// Skip when empty
				if svcTotal == 0 {
					fmt.Printf("%s: 0 instances requested (combinations=%d, count=%d), skipping\n", spawn.ServiceLabel(k), svcCombinations, svc.Params.ShardCount)
					continue
				}

				// Print information
				infos := make([]string, 0)
				if svcCombinations > 1 {
					infos = append(infos, fmt.Sprintf("%d combinations", svcCombinations))
				}
				if svc.Params.ShardCount > 1 {
					infos = append(infos, fmt.Sprintf("sharded %d times", svc.Params.ShardCount))
				}
				if svc.Parallelism < svc.Params.ShardCount {
					infos = append(infos, fmt.Sprintf("parallelism: %d", svc.Parallelism))
				}
				if svcTotal == 1 {
					fmt.Printf("%s: 1 instance requested\n", spawn.ServiceLabel(k))
				} else {
					fmt.Printf("%s: %d instances requested: %s\n", spawn.ServiceLabel(k), svcTotal, strings.Join(infos, ", "))
				}

				// Limit parallelism
				if svc.Parallelism > MaxParallelism {
					svc.Parallelism = int64(MaxParallelism)
					fmt.Printf("   limited parallelism to %d for stability\n", MaxParallelism)
				}

				// Apply to the state
				total += svcTotal

				// Prepare locks for all instances
				locks := make([]sync.Mutex, svcTotal)
				for i := int64(0); i < svcTotal; i++ {
					locks[i] = sync.Mutex{}
					locks[i].Lock()
				}

				// Save the service
				services = append(services, svc)
				servicesMap[svc.Name] = svc
				serviceLocks = append(serviceLocks, locks)
				serviceLocksMap[svc.Name] = &serviceLocks[len(serviceLocks)-1]
			}

			// Fast-track when nothing is requested
			if total == 0 {
				saveState()
				fmt.Printf("No pods requested.\n")
				os.Exit(0)
			}

			// Initialize Kubernetes client
			clientSet := env.Kubernetes()
			artifacts := artifacts.InternalStorage()

			// Initialize list of pods to schedule
			fmt.Println("Computing and packaging resources...")
			schedulablePods, storage, transferServer, err := spawn.BuildResources(services, podsRef, baseMachine)
			if err != nil {
				fail(err.Error())
			}
			fmt.Println("Resources ready.")

			// Start transferring server when needed
			if transferServer.Count() > 0 {
				_, err = transferServer.Listen("tcp", ":9999")
				if err != nil {
					fail("failed to start transfer server")
				}
				fmt.Println("Initialized files transfer server.")
			}

			// Watch events for all Pod modifications
			initialized := make(map[string]struct{})
			started := make(map[string]struct{})
			timedOut := make(map[string]struct{})
			err = spawn.WatchPods(context.Background(), clientSet, podsRef, servicesMap, func(svc spawn.Service, index int64, pod *corev1.Pod) {
				updateState(svc.Name, index, pod)
				state := getState(svc.Name, index)
				if _, ok := initialized[pod.Name]; ok {
					return
				}

				var firstContainer corev1.ContainerStatus
				if len(pod.Status.InitContainerStatuses) > 0 {
					firstContainer = pod.Status.InitContainerStatuses[0]
				} else if len(pod.Status.ContainerStatuses) > 0 {
					firstContainer = pod.Status.ContainerStatuses[0]
				}
				if firstContainer.State.Running != nil || firstContainer.State.Terminated != nil {
					if _, ok := started[pod.Name]; !ok {
						started[pod.Name] = struct{}{}
						data.PrintOutput(env.Ref(), "service-status", spawn.ServiceStatus{
							Name:      svc.Name,
							Index:     index,
							StartedAt: common.Ptr(time.Now()),
						})
					}
				}

				podSuccess, err := svc.EvalReady(state, index, baseMachine)
				if err != nil {
					fmt.Printf("%s: warning: parsing 'success' condition: %s\n", spawn.InstanceLabel(svc.Name, index, svc.Params.Count), err.Error())
					return
				}
				podError, err := svc.EvalError(state, index, baseMachine)
				if err != nil {
					fmt.Printf("%s: warning: parsing 'error' condition: %s\n", spawn.InstanceLabel(svc.Name, index, svc.Params.Count), err.Error())
					return
				}

				// Get status
				_, timeout := timedOut[pod.Name]
				failed := (podError != nil && *podError) || timeout
				succeed := podSuccess != nil && *podSuccess

				// Inform about status
				if failed || succeed {
					status := "failed"
					if !failed {
						status = "success"
					}
					data.PrintOutput(env.Ref(), "service-status", spawn.ServiceStatus{
						Name:       svc.Name,
						Index:      index,
						Status:     status,
						FinishedAt: common.Ptr(time.Now()),
					})
				}

				// Delete when it is no longer needed
				if !longRunning && (failed || succeed) && pod.DeletionTimestamp == nil {
					var err error
					if svc.Logs {
						err = spawn.DeletePodAndSaveLogs(context.Background(), clientSet, artifacts, svc, pod, index)
					} else {
						err = spawn.DeletePod(context.Background(), clientSet, pod)
					}
					if err != nil {
						fmt.Printf("%s: warning: failed to delete obsolete pod: %s\n", spawn.InstanceLabel(svc.Name, index, svc.Params.Count), err.Error())
					}
				}

				if failed {
					if timeout || pod.Status.Reason == "DeadlineExceeded" {
						fmt.Printf("%s: timed out\n", spawn.InstanceLabel(svc.Name, index, svc.Params.Count))
					} else {
						fmt.Printf("%s: failed\n", spawn.InstanceLabel(svc.Name, index, svc.Params.Count))
					}
					initialized[pod.Name] = struct{}{}
					(*serviceLocksMap[svc.Name])[index].Unlock()
				} else if succeed {
					if longRunning {
						fmt.Printf("%s: initialized successfully on %s\n", spawn.InstanceLabel(svc.Name, index, svc.Params.Count), pod.Spec.NodeName)
					} else {
						fmt.Printf("%s: finished successfully on %s\n", spawn.InstanceLabel(svc.Name, index, svc.Params.Count), pod.Spec.NodeName)
					}
					initialized[pod.Name] = struct{}{}
					success.Add(1)
					(*serviceLocksMap[svc.Name])[index].Unlock()
				}
			})
			if err != nil {
				fail("Couldn't watch Kubernetes for pod changes: %s", err.Error())
			}

			// Create required config maps
			if len(storage.ConfigMaps()) > 0 {
				fmt.Printf("Creating %d ConfigMaps for %d unique files.\n", len(storage.ConfigMaps()), storage.FilesCount())
			}
			for _, cfg := range storage.ConfigMaps() {
				_, err := clientSet.CoreV1().ConfigMaps(env.Namespace()).
					Create(context.Background(), &cfg, metav1.CreateOptions{})
				if err != nil {
					fail("creating ConfigMap: %s", err.Error())
				}
			}

			// Make spacing
			fmt.Println()

			// Initialize all the services
			// TODO: Consider dry-run as well
			spawn.EachService(services, schedulablePods, func(svc spawn.Service, svcIndex int, pod *corev1.Pod, index int64, combinations int64) {
				// Compute timeout duration
				timeout, err := svc.TimeoutDuration(index, baseMachine)
				if err != nil {
					fail("%s: error while reading timeout: %s", spawn.InstanceLabel(svc.Name, index, svc.Params.Count), err.Error())
				}
				isTimeoutApplicable := timeout != nil && (pod.Spec.ActiveDeadlineSeconds == nil || float64(*pod.Spec.ActiveDeadlineSeconds) > timeout.Seconds())

				// Apply timeout for workers
				if !longRunning && isTimeoutApplicable {
					pod.Spec.ActiveDeadlineSeconds = common.Ptr(int64(math.Ceil(timeout.Seconds())))
				}

				// Build service instance description
				description, err := svc.DescriptionAt(index, baseMachine)
				if err != nil {
					fail("%s: error while reading service description: %s", spawn.InstanceLabel(svc.Name, index, svc.Params.Count), err.Error())
				}

				// Create the pod
				pod, err = clientSet.CoreV1().Pods(env.Namespace()).
					Create(context.Background(), pod, metav1.CreateOptions{})
				if err != nil {
					fail("%s: error while creating pod: %s", spawn.InstanceLabel(svc.Name, index, svc.Params.Count), err.Error())
				}

				// Inform about the pod creation
				fmt.Printf("%s: created pod %s\n", spawn.InstanceLabel(svc.Name, index, svc.Params.Count), ui.DarkGray("("+pod.Name+")"))
				data.PrintOutput(env.Ref(), "service-status", spawn.ServiceStatus{
					Name:        svc.Name,
					Description: description,
					Index:       index,
					CreatedAt:   common.Ptr(time.Now()),
					Status:      "running",
				})

				// Update the initial data
				updateState(svc.Name, index, pod)

				// Apply timeout for long-running services
				if longRunning && isTimeoutApplicable {
					go func() {
						time.Sleep(*timeout)
						if _, ok := initialized[pod.Name]; ok {
							return
						}
						timedOut[pod.Name] = struct{}{}
						fmt.Printf("%s: takes longer than expected %s\n", spawn.InstanceLabel(svc.Name, index, svc.Params.Count), timeout.String())
						_ = spawn.DeletePod(context.Background(), clientSet, pod)
					}()
				}

				// Wait until it's ready
				serviceLocks[svcIndex][index].Lock()
			})

			// Make spacing
			fmt.Println()

			saveState()

			if longRunning {
				if success.Load() == total {
					fmt.Printf("Successfully started %d pods.\n", total)
				} else {
					fmt.Printf("Failed to initialize %d out of %d expected pods.\n", total-success.Load(), total)
					os.Exit(1)
				}
			} else {
				if success.Load() == total {
					fmt.Printf("Successfully finished %d pods.\n", total)
				} else {
					fmt.Printf("Failed to finish %d out of %d expected pods.\n", total-success.Load(), total)
					os.Exit(1)
				}
			}
		},
	}

	cmd.Flags().StringArrayVarP(&instructionsStr, "instructions", "i", nil, "pod instructions to start")
	cmd.Flags().BoolVarP(&longRunning, "services", "s", false, "are these long-running services")

	return cmd
}
