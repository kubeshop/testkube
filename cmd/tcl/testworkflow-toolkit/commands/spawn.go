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
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/spawn"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor"
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
			schedulablePods := make([][]*corev1.Pod, len(services))
			storage := testworkflowprocessor.NewConfigMapFiles(fmt.Sprintf("%s-%s-vol", env.ExecutionId(), podsRef), map[string]string{
				testworkflowprocessor.ExecutionIdLabelName:         env.ExecutionId(),
				testworkflowprocessor.ExecutionAssistingPodRefName: podsRef,
			})

			for svcIndex, svc := range services {
				combinations := spawn.CountCombinations(svc.Matrix)
				schedulablePods[svcIndex] = make([]*corev1.Pod, svc.Count*combinations)
				for i := int64(0); i < svc.Count*combinations; i++ {
					pod, err := svc.Pod(podsRef, i, baseMachine)
					if err != nil {
						fail(err.Error())
					}
					files, err := svc.Files(i, baseMachine)
					if err != nil {
						fail(err.Error())
					}
					for path, content := range files {
						// Apply file
						mount, volume, err := storage.AddTextFile(content)
						if err != nil {
							fail("%s: %s instance: file %s: %s", svc.Name, humanize.Ordinal(int(i)), path, err.Error())
						}

						// Append the volume mount
						mount.MountPath = path
						for i := range pod.Spec.InitContainers {
							pod.Spec.InitContainers[i].VolumeMounts = append(pod.Spec.InitContainers[i].VolumeMounts, mount)
						}
						for i := range pod.Spec.Containers {
							pod.Spec.Containers[i].VolumeMounts = append(pod.Spec.Containers[i].VolumeMounts, mount)
						}

						// Append the volume if it's not yet added
						if !slices.ContainsFunc(pod.Spec.Volumes, func(v corev1.Volume) bool {
							return v.Name == mount.Name
						}) {
							pod.Spec.Volumes = append(pod.Spec.Volumes, volume)
						}
					}

					schedulablePods[svcIndex][i] = pod
				}
			}

			// Watch events for all Pod modifications
			podWatch, err := clientSet.CoreV1().Pods(env.Namespace()).Watch(context.Background(), metav1.ListOptions{
				TypeMeta:      metav1.TypeMeta{Kind: "Pod"},
				LabelSelector: fmt.Sprintf("%s=%s", testworkflowprocessor.ExecutionAssistingPodRefName, podsRef),
			})
			if err != nil {
				fail("Couldn't watch Kubernetes for pod changes: %s", err.Error())
			}
			initialized := make(map[string]struct{})
			go func() {
				defer podWatch.Stop()

				for ev := range podWatch.ResultChan() {
					if pod, ok := ev.Object.(*corev1.Pod); ok {
						segments := strings.Split(pod.Name, "-")
						name := segments[2]
						index, err := strconv.ParseInt(segments[3], 10, 64)
						if err != nil {
							// Unknown shard
							continue
						}
						if _, ok := servicesMap[name]; !ok {
							// Unknown service
							continue
						}
						if _, ok := initialized[pod.Name]; ok {
							// Already initialized
							continue
						}
						updateState(name, index, pod)

						// Check the conditions
						state := getState(name, index)
						svc := servicesMap[name]

						podSuccess, err := svc.EvalReady(state, index)
						if err != nil {
							fmt.Printf("Warning: %s: parsing 'success' condition: %s\n", pod.Name, err.Error())
							continue
						}
						podError, err := svc.EvalError(state, index)
						if err != nil {
							fmt.Printf("Warning: %s: parsing 'error' condition: %s\n", pod.Name, err.Error())
							continue
						}

						if podError != nil && *podError {
							fmt.Printf("%s: pod %s (%d) failed\n", svc.Name, pod.Name, index+1)
							initialized[pod.Name] = struct{}{}
							(*serviceLocksMap[svc.Name])[index].Unlock()
						} else if podSuccess != nil && *podSuccess {
							fmt.Printf("%s: pod %s (%d) initialized successfully on %s\n", svc.Name, pod.Name, index+1, pod.Spec.NodeName)
							success.Add(1)
							initialized[pod.Name] = struct{}{}
							(*serviceLocksMap[svc.Name])[index].Unlock()
						}
					}
				}
			}()

			// Create required config maps
			if len(storage.ConfigMaps()) > 0 {
				fmt.Printf("Creating %d ConfigMaps for %d unique files.\n", len(storage.ConfigMaps()), storage.FilesCount())
			}
			for _, cfg := range storage.ConfigMaps() {
				_, err := clientSet.CoreV1().ConfigMaps(env.Namespace()).
					Create(context.Background(), &cfg, metav1.CreateOptions{})
				if err != nil {
					// TODO: Cleanup the rest of config maps
					fail("creating ConfigMap: %s", err.Error())
				}
			}

			// Prepare wait group to wait for all services
			var wg sync.WaitGroup
			wg.Add(len(services))

			// Initialize all the services
			// TODO: Consider dry-run as well
			// TODO: Decouple
			for i, v := range services {
				go func(svc spawn.Service, svcIndex int) {
					combinations := svc.Combinations()

					var swg sync.WaitGroup
					swg.Add(int(combinations * svc.Count))
					sema := make(chan struct{}, svc.Parallelism)

					for index, pod := range schedulablePods[svcIndex] {
						sema <- struct{}{}
						go func(index int64, pod *corev1.Pod) {
							defer func() {
								<-sema
								swg.Done()
							}()

							// Create the pod
							pod, err := clientSet.CoreV1().Pods(env.Namespace()).
								Create(context.Background(), pod, metav1.CreateOptions{})
							if err != nil {
								fail("[%d/%d] %s: error while creating pod: %s", index+1, combinations*svc.Count, svc.Name, err.Error())
							}

							// Inform about the pod creation
							fmt.Printf("[%d/%d] %s: created pod\n", index+1, combinations*svc.Count, svc.Name)

							// Update the initial data
							updateState(svc.Name, index, pod)

							// TODO: Support the timeout

							// Wait until it's ready
							serviceLocks[svcIndex][index].Lock()
						}(int64(index), pod)
					}

					swg.Wait()
					wg.Done()
				}(v, i)
			}

			// Wait until all pods will be ready to continue
			wg.Wait()

			saveState()

			if success.Load() == total {
				fmt.Printf("Successfully started %d pods.\n", total)
			} else {
				fmt.Printf("Failed to initialize %d out of %d expected pods.\n", total-success.Load(), total)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().StringArrayVarP(&instructionsStr, "instructions", "i", nil, "pod instructions to start")
	cmd.Flags().BoolVarP(&longRunning, "services", "s", false, "are these long-running services")

	return cmd
}
