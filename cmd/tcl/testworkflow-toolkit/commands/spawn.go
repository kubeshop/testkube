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
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/rand"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor"
)

const MaxParallelism = 5000
const maxConfigMapFileSize = 950 * 1024

type Service struct {
	Name        string
	Count       int64
	Parallelism int64
	Timeout     string
	Matrix      map[string][]interface{}
	Shards      map[string][]interface{}
	Ready       string
	Error       string
	Content     *testworkflowsv1.SpawnContent
	PodTemplate corev1.PodTemplateSpec
}

func (svc *Service) ShardIndexAt(index int64) int64 {
	return index % svc.Count
}

func (svc *Service) CombinationIndexAt(index int64) int64 {
	return (index - svc.ShardIndexAt(index)) / svc.Count
}

func (svc *Service) Combinations() int64 {
	return countCombinations(svc.Matrix)
}

func (svc *Service) MatrixAt(index int64) map[string]interface{} {
	return getMatrixValues(svc.Matrix, svc.CombinationIndexAt(index))
}

func (svc *Service) ShardsAt(index int64) map[string][]interface{} {
	return getShardValues(svc.Matrix, svc.ShardIndexAt(index), svc.Count)
}

func (svc *Service) MachineAt(index int64) expressionstcl.Machine {
	// Get basic indices
	combinations := svc.Combinations()
	shardIndex := svc.ShardIndexAt(index)
	combinationIndex := svc.CombinationIndexAt(index)

	// Compute values for this instance
	matrixValues := svc.MatrixAt(combinationIndex)
	shardValues := getShardValues(svc.Shards, shardIndex, svc.Count)

	return expressionstcl.NewMachine().
		Register("index", index).
		Register("count", combinations*svc.Count).
		Register("matrixIndex", combinationIndex).
		Register("matrixCount", combinations).
		Register("matrix", matrixValues).
		Register("shardIndex", shardIndex).
		Register("shardsCount", svc.Count).
		Register("shard", shardValues)
}

func (svc *Service) Pod(ref string, longRunning bool, index int64, globalMachine expressionstcl.Machine) (*corev1.Pod, error) {
	// Get details for current position
	combinations := svc.Combinations()
	machine := svc.MachineAt(index)

	// Build a pod
	spec := svc.PodTemplate.DeepCopy()
	err := expressionstcl.FinalizeForce(&spec, machine, globalMachine)
	if err != nil {
		return nil, fmt.Errorf("[%d/%d] %s: error while resolving pod schema: %s", index+1, combinations*svc.Count, svc.Name, err.Error())
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-%s-%d", env.ExecutionId(), ref, svc.Name, index),
			Namespace: env.Namespace(),
			Labels: map[string]string{
				testworkflowprocessor.ExecutionIdLabelName:         env.ExecutionId(),
				testworkflowprocessor.ExecutionAssistingPodRefName: ref,
				testworkflowprocessor.AssistingPodServiceName:      "true",
			},
			Annotations: spec.Annotations,
		},
		Spec: spec.Spec,
	}
	if !longRunning && pod.Spec.RestartPolicy == "" {
		pod.Spec.RestartPolicy = corev1.RestartPolicyNever
	}
	if pod.Labels == nil {
		pod.Labels = map[string]string{}
	}
	pod.Labels[testworkflowprocessor.ExecutionIdLabelName] = env.ExecutionId()
	pod.Labels[testworkflowprocessor.ExecutionAssistingPodRefName] = ref
	pod.Labels[testworkflowprocessor.AssistingPodServiceName] = "true"
	if pod.Spec.Subdomain == "" {
		pod.Spec.Subdomain = testworkflowprocessor.AssistingPodServiceName
	}
	if pod.Spec.Hostname == "" {
		pod.Spec.Hostname = fmt.Sprintf("%s-%s-%d", env.ExecutionId(), svc.Name, index)
	}
	// Append random names to pod containers
	for i := range pod.Spec.InitContainers {
		if pod.Spec.InitContainers[i].Name == "" {
			pod.Spec.InitContainers[i].Name = fmt.Sprintf("c%s-%d", rand.String(5), i)
		}
	}
	for i := range pod.Spec.Containers {
		if pod.Spec.Containers[i].Name == "" {
			pod.Spec.Containers[i].Name = fmt.Sprintf("c%s-%d", rand.String(5), len(pod.Spec.InitContainers)+i)
		}
	}

	return pod, nil
}

func (svc *Service) Files(index int64, globalMachine expressionstcl.Machine) (map[string]string, error) {
	// Ignore when there are no files expected
	if svc.Content == nil || len(svc.Content.Files) == 0 {
		return nil, nil
	}

	// Prepare data for computation
	combinations := svc.Combinations()
	machine := svc.MachineAt(index)
	files := make(map[string]string, len(svc.Content.Files))

	// Compute all files
	var err error
	for fileIndex, file := range svc.Content.Files {
		files[file.Path], err = expressionstcl.EvalTemplate(file.Content, machine, globalMachine)
		if err != nil {
			return nil, fmt.Errorf("[%d/%d] %s: error while resolving %s (%d): %s", index+1, combinations*svc.Count, svc.Name, file.Path, fileIndex, err.Error())
		}
	}
	return files, nil
}

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

func countCombinations(matrix map[string][]interface{}) int64 {
	combinations := int64(1)
	for k := range matrix {
		combinations *= int64(len(matrix[k]))
	}
	return combinations
}

func getMatrixValues(matrix map[string][]interface{}, index int64) map[string]interface{} {
	// Compute modulo for each matrix parameter
	keys := maps.Keys(matrix)
	modulo := map[string]int64{}
	floor := map[string]int64{}
	for i, k := range keys {
		modulo[k] = int64(len(matrix[k]))
		floor[k] = 1
		for j := i + 1; j < len(keys); j++ {
			floor[k] *= int64(len(matrix[keys[j]]))
		}
	}

	// Compute values for selected index
	result := make(map[string]interface{})
	for _, k := range keys {
		kIdx := (index / floor[k]) % modulo[k]
		result[k] = matrix[k][kIdx]
	}
	return result
}

func getShardValues(values map[string][]interface{}, index int64, count int64) map[string][]interface{} {
	result := make(map[string][]interface{})
	for k := range values {
		if index > int64(len(values[k])) {
			result[k] = []interface{}{}
			continue
		}
		size := count / int64(len(values[k]))
		if size == 0 {
			size = 1
		}
		start := index * size
		end := start + size
		if end > int64(len(values[k])) {
			result[k] = values[k][start:]
		} else {
			result[k] = values[k][start:end]
		}
	}
	return result
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
			services := make([]Service, 0)
			servicesMap := make(map[string]Service)
			serviceLocks := make([][]sync.Mutex, 0)
			serviceLocksMap := make(map[string]*[]sync.Mutex)

			// Resolve the instructions
			for k := range instructions {
				// Resolve the shards and matrices
				shards, err := readParams(instructions[k].Shards, instructions[k].ShardExpressions)
				if err != nil {
					fail("[%s]: shards: %s", k, err)
				}
				matrix, err := readParams(instructions[k].Matrix, instructions[k].MatrixExpressions)
				if err != nil {
					fail("[%s]: shards: %s", k, err)
				}
				minShards := int64(math.MaxInt64)
				for key := range shards {
					if int64(len(shards[key])) < minShards {
						minShards = int64(len(shards[key]))
					}
				}

				// Calculate number of matrix combinations
				combinations := countCombinations(matrix)

				// Resolve the count
				var count, maxCount *int64
				if instructions[k].Count != nil {
					countVal, err := readCount(*instructions[k].Count)
					if err != nil {
						fail("[%s].count: %s", k, err)
					}
					count = &countVal
				}
				if instructions[k].MaxCount != nil {
					countVal, err := readCount(*instructions[k].MaxCount)
					if err != nil {
						fail("[%s].maxCount: %s\n", k, err)
					}
					maxCount = &countVal
				}
				if count == nil && maxCount == nil {
					count = common.Ptr(int64(1))
				}
				if count != nil && maxCount != nil && *maxCount < *count {
					count = maxCount
					maxCount = nil
				}
				if maxCount != nil && *maxCount < minShards {
					count = &minShards
					maxCount = nil
				} else if maxCount != nil {
					count = maxCount
					maxCount = nil
				}
				total += *count * combinations

				// Initialize the service state
				states[k] = make([]ServiceState, total)

				// Skip service if it has no instances requested
				if *count == 0 {
					fmt.Printf("[%s] 0 instances requested (combinations=%d, count=%d), skipping\n", k, combinations, *count)
					continue
				}

				// Compute parallelism
				var parallelism *int64
				if instructions[k].Parallelism != nil {
					parallelismVal, err := readCount(*instructions[k].Parallelism)
					if err != nil {
						fail("[%s].parallelism: %s", k, err)
					}
					parallelism = &parallelismVal
				}
				if parallelism == nil {
					parallelism = common.Ptr(int64(math.MaxInt64))
				}
				if *parallelism > *count*combinations {
					parallelism = common.Ptr(*count * combinations)
				}
				if *parallelism > MaxParallelism {
					parallelism = common.Ptr(int64(MaxParallelism))
					fmt.Printf("   limited parallelism to %d for stability\n", MaxParallelism)
				}

				// Build the service
				svc := Service{
					Name:        k,
					Count:       *count,
					Parallelism: *parallelism,
					Timeout:     instructions[k].Timeout,
					Matrix:      matrix,
					Shards:      shards,
					Ready:       instructions[k].Ready,
					Error:       instructions[k].Error,
					PodTemplate: instructions[k].Pod,
					Content:     instructions[k].Content,
				}

				// Define the default success/error clauses
				if svc.Ready == "" {
					if longRunning {
						svc.Ready = "ready && containerStarted"
					} else {
						svc.Ready = "success"
					}
				}
				if svc.Error == "" {
					svc.Error = "deleted || failed"
				}

				// Prepare locks for all instances
				locks := make([]sync.Mutex, combinations*svc.Count)
				for i := int64(0); i < combinations*svc.Count; i++ {
					locks[i] = sync.Mutex{}
					locks[i].Lock()
				}

				// Save the service
				services = append(services, svc)
				servicesMap[svc.Name] = svc
				serviceLocks = append(serviceLocks, locks)
				serviceLocksMap[svc.Name] = &serviceLocks[len(serviceLocks)-1]
				fmt.Printf("[%s] %d instances requested: %d combinations, sharded %d times (parallelism: %d)\n", k, svc.Count*combinations, combinations, svc.Count, svc.Parallelism)
			}

			// Ensure the services are valid
			for i := range services {
				if len(services[i].PodTemplate.Spec.Containers) == 0 {
					fail("[%s].pod.spec.containers: no containers provided", services[i].Name)
				}
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
				combinations := countCombinations(svc.Matrix)
				schedulablePods[svcIndex] = make([]*corev1.Pod, svc.Count*combinations)
				for i := int64(0); i < svc.Count*combinations; i++ {
					pod, err := svc.Pod(podsRef, longRunning, i, internalMachine)
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
				go func(svc Service, svcIndex int) {
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
							pod, err = clientSet.CoreV1().Pods(env.Namespace()).
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
