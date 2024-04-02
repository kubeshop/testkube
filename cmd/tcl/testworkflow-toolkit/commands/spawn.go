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
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/spawn"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor"
)

const MaxParallelism = 5000
const maxConfigMapFileSize = 950 * 1024

type ServiceState struct {
	Name             string     `json:"name"`
	Host             string     `json:"host"`
	Ip               string     `json:"ip"`
	Started          bool       `json:"started"`
	ContainerStarted bool       `json:"containerStarted"`
	Ready            bool       `json:"ready"`
	Deleted          bool       `json:"deleted"`
	Success          bool       `json:"success"`
	Failed           bool       `json:"failed"`
	Finished         bool       `json:"finished"`
	Pod              corev1.Pod `json:"pod"`
}

func readCount(s intstr.IntOrString) (int64, error) {
	countExpr, err := expressionstcl.Compile(s.String())
	if err != nil {
		return 0, fmt.Errorf("%s: invalid: %s", s.String(), err)
	}
	if countExpr.Static() == nil {
		return 0, fmt.Errorf("%s: could not resolve: %s", s.String(), err)
	}
	countVal, err := countExpr.Static().IntValue()
	if err != nil {
		return 0, fmt.Errorf("%s: could not convert to int: %s", s.String(), err)
	}
	if countVal < 0 {
		return 0, fmt.Errorf("%s: should not be lower than zero", s.String())
	}
	return countVal, nil
}

func readParams(base map[string][]intstr.IntOrString, expressions map[string]string) (map[string][]interface{}, error) {
	result := make(map[string][]interface{})
	for key, list := range base {
		result[key] = make([]interface{}, len(list))
		for i := range list {
			result[key][i] = list[i].String()
		}
	}
	for key, exprStr := range expressions {
		if _, ok := result[key]; !ok {
			result[key] = make([]interface{}, 0)
		}
		expr, err := expressionstcl.Compile(exprStr)
		if err != nil {
			return nil, fmt.Errorf("%s: %s: %s\n", key, exprStr, err)
		}
		if expr.Static() == nil {
			return nil, fmt.Errorf("%s: %s: could not resolve\n", key, exprStr)
		}
		list, err := expr.Static().SliceValue()
		if err != nil {
			return nil, fmt.Errorf("%s: %s: could not parse as list: %s\n", key, exprStr, err)
		}
		result[key] = append(result[key], list...)
	}
	for key := range expressions {
		if len(expressions[key]) == 0 {
			delete(expressions, key)
		}
	}
	return result, nil
}

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

			// Initialize internal machine TODO: Think if it is fine
			data.LoadState()
			internalMachine := expressionstcl.CombinedMachines(data.EnvMachine, data.StateMachine, data.FileMachine)

			// Initialize state
			states := make(map[string][]ServiceState)
			var statesMu sync.Mutex
			saveState := func() {
				if longRunning {
					for k := range states {
						data.PrintHintDetails(env.Ref(), fmt.Sprintf("services.%s", k), states[k])
					}
				}
			}
			getState := func(name string, index int64) ServiceState {
				defer statesMu.Unlock()
				statesMu.Lock()
				return states[name][index]
			}
			updateState := func(name string, index int64, fn func(s ServiceState) ServiceState) {
				defer statesMu.Unlock()
				statesMu.Lock()
				states[name][index] = fn(states[name][index])
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

			// Ensure all instructions have a container
			for k := range instructions {
				if len(instructions[k].Pod.Spec.Containers) == 0 {
					fail("Problem processing the assisting pod '%s': spec.containers: pod needs to have any containers specified", k)
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
				if longRunning && instruction.Pod.Spec.RestartPolicy == "" {
					instruction.Pod.Spec.RestartPolicy = corev1.RestartPolicyNever
				}

				// Build the service
				svc, err := spawn.FromInstruction(k, instructions[k])
				svcTotal := svc.Total()
				if err != nil {
					fail("%s: %w", k, err)
				}

				// Apply empty state
				states[k] = make([]ServiceState, svcTotal)

				// Skip when empty
				if svcTotal == 0 {
					fmt.Printf("[%s] 0 instances requested (combinations=%d, count=%d), skipping\n", k, svc.Combinations(), svc.Count)
					continue
				}

				// Print information
				fmt.Printf("[%s] %d instances requested: %d combinations, sharded %d times (parallelism: %d)\n", k, svcTotal, svc.Combinations(), svc.Count, svc.Parallelism)

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
			// TODO: Reduce number of config maps (not one per file)
			// TODO: Dedupe files
			schedulablePods := make([][]*corev1.Pod, len(services))
			fileMounts := make(map[string]corev1.VolumeMount)
			fileVolumes := make(map[string]corev1.Volume)
			configMaps := make([]*corev1.ConfigMap, 0)

			for svcIndex, svc := range services {
				combinations := spawn.CountCombinations(svc.Matrix)
				schedulablePods[svcIndex] = make([]*corev1.Pod, svc.Count*combinations)
				for i := int64(0); i < svc.Count*combinations; i++ {
					pod, err := svc.Pod(podsRef, i, internalMachine)
					if err != nil {
						fail(err.Error())
					}
					files, err := svc.Files(i, internalMachine)
					if err != nil {
						fail(err.Error())
					}
					for path, content := range files {
						hash := fmt.Sprintf("%x", sha256.Sum256([]byte(content)))

						// Detect or create volume/mount
						mount, ok := fileMounts[hash]
						if !ok {
							var cfgMap *corev1.ConfigMap
							// Find config map which has enough space for this file
							for _, cfg := range configMaps {
								size := 0
								for _, v := range cfg.Data {
									size += len(v)
								}
								if size+len(content) < maxConfigMapFileSize {
									cfgMap = cfg
									break
								}
							}
							// Create new config map if needed
							if cfgMap == nil {
								cfgName := fmt.Sprintf("%s-%s-vol-%d", env.ExecutionId(), podsRef, len(configMaps))
								cfgMap = &corev1.ConfigMap{
									ObjectMeta: metav1.ObjectMeta{
										Name: cfgName,
										Labels: map[string]string{
											testworkflowprocessor.ExecutionIdLabelName:         env.ExecutionId(),
											testworkflowprocessor.ExecutionAssistingPodRefName: podsRef,
										},
									},
									Data:      map[string]string{},
									Immutable: common.Ptr(true),
								}
								configMaps = append(configMaps, cfgMap)
								fileVolumes[cfgName] = corev1.Volume{
									Name: cfgName,
									VolumeSource: corev1.VolumeSource{
										ConfigMap: &corev1.ConfigMapVolumeSource{
											LocalObjectReference: corev1.LocalObjectReference{Name: cfgName},
										},
									},
								}
							}
							key := fmt.Sprintf("%d", len(cfgMap.Data))
							cfgMap.Data[key] = content
							mount = corev1.VolumeMount{
								Name:     cfgMap.Name,
								ReadOnly: true,
								SubPath:  key,
							}
							fileMounts[hash] = mount
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
							pod.Spec.Volumes = append(pod.Spec.Volumes, fileVolumes[mount.Name])
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
						updateState(name, index, func(s ServiceState) ServiceState {
							s.Pod = *pod
							s.Started = pod.Status.StartTime != nil
							s.Deleted = pod.DeletionTimestamp != nil
							s.Success = pod.Status.Phase == "Succeeded"
							s.Failed = pod.Status.Phase == "Failed"
							s.Finished = s.Deleted || s.Success || s.Failed
							s.Ip = pod.Status.PodIP
							for _, c := range pod.Status.ContainerStatuses {
								if c.State.Running != nil || c.State.Terminated != nil {
									s.ContainerStarted = true
								}
							}
							for _, cond := range pod.Status.Conditions {
								if cond.Type == "Ready" && cond.Status == "True" {
									s.Ready = true
								}
							}
							return s
						})

						// Check the conditions
						state := getState(name, index)
						svc := servicesMap[name]
						machine := expressionstcl.NewMachine().
							Register("started", state.Started).
							Register("containerStarted", state.ContainerStarted).
							Register("deleted", state.Deleted).
							Register("success", state.Success).
							Register("failed", state.Failed).
							Register("finished", state.Finished).
							Register("ready", state.Ready).
							Register("ip", state.Ip).
							Register("host", state.Host).
							Register("pod", state.Pod).
							Register("index", index)
						// TODO: ignore "should be static" error
						successExpr, err := expressionstcl.EvalExpressionPartial(svc.Ready, machine)
						if err != nil {
							fmt.Printf("Warning: %s: parsing success condition: %s\n", pod.Name, err.Error())
							continue
						}
						failedExpr, err := expressionstcl.EvalExpressionPartial(svc.Error, machine)
						if err != nil {
							fmt.Printf("Warning: %s: parsing failed condition: %s\n", pod.Name, err.Error())
							continue
						}

						if failedExpr.Static() != nil {
							v, _ := failedExpr.Static().BoolValue()
							if v {
								fmt.Printf("%s: pod %s (%d) failed\n", svc.Name, pod.Name, index+1)
								initialized[pod.Name] = struct{}{}
								(*serviceLocksMap[svc.Name])[index].Unlock()
							}
						}
						if successExpr.Static() != nil {
							v, _ := successExpr.Static().BoolValue()
							if v {
								fmt.Printf("%s: pod %s (%d) initialized successfully on %s\n", svc.Name, pod.Name, index+1, pod.Spec.NodeName)
								success.Add(1)
								initialized[pod.Name] = struct{}{}
								(*serviceLocksMap[svc.Name])[index].Unlock()
							}
						}
					}
				}
			}()

			// Create required config maps
			if len(configMaps) > 0 {
				fmt.Printf("Creating %d ConfigMaps for %d unique files.\n", len(configMaps), len(fileMounts))
			}
			for _, cfg := range configMaps {
				_, err := clientSet.CoreV1().ConfigMaps(env.Namespace()).
					Create(context.Background(), cfg, metav1.CreateOptions{})
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
							updateState(svc.Name, index, func(s ServiceState) ServiceState {
								s.Name = pod.Name
								s.Host = fmt.Sprintf("%s.%s.%s.svc.cluster.local", pod.Spec.Hostname, pod.Spec.Subdomain, pod.Namespace)
								return s
							})

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
