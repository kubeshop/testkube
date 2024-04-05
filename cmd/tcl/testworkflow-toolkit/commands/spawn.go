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
	"strings"
	"sync"
	"sync/atomic"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/spawn"
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
				svc, err := spawn.FromInstruction(k, instruction)
				svcCombinations := svc.Combinations()
				svcTotal := svc.Total()
				if err != nil {
					fail("%s: %s", k, err.Error())
				}

				// Apply empty state
				states[k] = make([]spawn.ServiceState, svcTotal)

				// Skip when empty
				if svcTotal == 0 {
					fmt.Printf("[%s] 0 instances requested (combinations=%d, count=%d), skipping\n", k, svcCombinations, svc.Count)
					continue
				}

				// Print information
				infos := make([]string, 0)
				if svcCombinations > 1 {
					infos = append(infos, fmt.Sprintf("%d combinations", svcCombinations))
				}
				if svc.Count > 1 {
					infos = append(infos, fmt.Sprintf("sharded %d times", svc.Count))
				}
				if svc.Parallelism < svc.Count {
					infos = append(infos, fmt.Sprintf("parallelism: %d", svc.Parallelism))
				}
				fmt.Printf("[%s] %d instances requested: %s\n", k, svcTotal, strings.Join(infos, ", "))

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

			// Initialize list of pods to schedule
			schedulablePods, storage, err := spawn.BuildResources(services, podsRef, baseMachine)
			if err != nil {
				fail(err.Error())
			}

			// Watch events for all Pod modifications
			initialized := make(map[string]struct{})
			err = spawn.WatchPods(context.Background(), clientSet, podsRef, servicesMap, func(svc spawn.Service, index int64, pod *corev1.Pod) {
				updateState(svc.Name, index, pod)
				state := getState(svc.Name, index)
				if _, ok := initialized[pod.Name]; ok {
					return
				}

				podSuccess, err := svc.EvalReady(state, index, baseMachine)
				if err != nil {
					fmt.Printf("Warning: %s: parsing 'success' condition: %s\n", pod.Name, err.Error())
					return
				}
				podError, err := svc.EvalError(state, index, baseMachine)
				if err != nil {
					fmt.Printf("Warning: %s: parsing 'error' condition: %s\n", pod.Name, err.Error())
					return
				}

				// Delete when it is no longer needed
				if !longRunning && ((podError != nil && *podError) || (podSuccess != nil && *podSuccess)) && pod.DeletionTimestamp == nil {
					err := spawn.DeletePod(context.Background(), clientSet, svc, podsRef, index)
					if err != nil {
						fmt.Printf("Warning: %s: failed to delete obsolete pod: %s\n", pod.Name, err.Error())
					}
				}

				if podError != nil && *podError {
					if pod.Status.Reason == "DeadlineExceeded" {
						fmt.Printf("%s: pod %s (%d) timed out\n", svc.Name, pod.Name, index+1)
					} else {
						fmt.Printf("%s: pod %s (%d) failed\n", svc.Name, pod.Name, index+1)
					}
					initialized[pod.Name] = struct{}{}
					(*serviceLocksMap[svc.Name])[index].Unlock()
				} else if podSuccess != nil && *podSuccess {
					if longRunning {
						fmt.Printf("%s: pod %s (%d) initialized successfully on %s\n", svc.Name, pod.Name, index+1, pod.Spec.NodeName)
					} else {
						fmt.Printf("%s: pod %s (%d) finished successfully on %s\n", svc.Name, pod.Name, index+1, pod.Spec.NodeName)
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

			// Initialize all the services
			// TODO: Consider dry-run as well
			spawn.EachService(services, schedulablePods, func(svc spawn.Service, svcIndex int, pod *corev1.Pod, index int64, combinations int64) {
				// Create the pod
				pod, err = clientSet.CoreV1().Pods(env.Namespace()).
					Create(context.Background(), pod, metav1.CreateOptions{})
				if err != nil {
					fail("[%d/%d] %s: error while creating pod: %s", index+1, combinations*svc.Count, svc.Name, err.Error())
				}

				// Inform about the pod creation
				fmt.Printf("[%d/%d] %s: created pod\n", index+1, combinations*svc.Count, svc.Name)

				// Update the initial data
				updateState(svc.Name, index, pod)

				// Wait until it's ready
				serviceLocks[svcIndex][index].Lock()
			})

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
