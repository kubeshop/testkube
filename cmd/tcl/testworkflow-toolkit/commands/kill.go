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
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	commontcl "github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/common"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/spawn"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/artifacts"
	toolkitcommon "github.com/kubeshop/testkube/cmd/testworkflow-toolkit/common"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/credentials"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
)

func NewKillCmd() *cobra.Command {
	var (
		logs []string
	)
	cmd := &cobra.Command{
		Use:   "kill <ref>",
		Short: "Kill accompanying service(s)",
		Args:  cobra.ExactArgs(1),

		Run: func(cmd *cobra.Command, args []string) {
			credMachine := credentials.NewCredentialMachine(data.Credentials())
			machine := expressions.CombinedMachines(data.AliasMachine, data.GetBaseTestWorkflowMachine(), data.ExecutionMachine(), credMachine)
			groupRef := args[0]

			conditions := make(map[string]expressions.Expression)
			for _, l := range logs {
				name, condition, found := strings.Cut(l, "=")
				if !found {
					condition = "true"
				}
				expr, err := expressions.CompileAndResolve(condition, machine)
				if err != nil {
					fmt.Printf("warning: service '%s': could not compile condition '%s': %s", name, condition, err.Error())
				} else {
					conditions[name] = expr
				}
			}

			worker := spawn.ExecutionWorker()
			namespace := config.Namespace()

			err := RunKill(cmd.Context(), worker, namespace, config.Ref(), groupRef, conditions, machine)
			if err == nil {
				return
			}
			// A service that the kernel stopped for its memory is an infrastructure failure, so the
			// step reports it apart from a failure of the test.
			var unhealthy *UnhealthyServicesError
			if errors.As(err, &unhealthy) && unhealthy.OOMKilled {
				toolkitcommon.FailWithReason(testkube.StopReasonOOMKilled, err)
			} else {
				toolkitcommon.Fail(err)
			}
		},
	}

	cmd.Flags().StringArrayVarP(&logs, "logs", "l", nil, "fetch the logs for specific services - pair <name>=<expression>")

	return cmd
}

// RunKillWithOptions stops services in a group: checks health, fetches logs,
// and destroys resources. Returns an error if any service has failed (e.g. OOMKilled).
func RunKillWithOptions(ctx context.Context, cfg *config.ConfigV2, groupRef string) error {
	worker := spawn.ParallelExecutionWorker(cfg)
	namespace := cfg.Namespace()
	return RunKill(ctx, worker, namespace, cfg.Internal().Resource.Id, groupRef, nil, nil)
}

// RunKill is the core kill logic, separated from the Cobra command for testability.
// It lists service instances, checks their health, fetches logs for matching
// services, destroys the group, and returns an error if any service has failed.
func RunKill(ctx context.Context, worker executionworkertypes.Worker, namespace string, ref string, groupRef string, conditions map[string]expressions.Expression, machine expressions.Machine) error {
	items, err := worker.List(ctx, executionworkertypes.ListOptions{
		GroupId: groupRef,
	})
	if err != nil {
		return fmt.Errorf("listing service instances: %w", err)
	}

	if len(items) > 0 {
		namespace = items[0].Namespace
		for _, item := range items {
			if item.Namespace != namespace {
				namespace = ""
				break
			}
		}
	}

	instanceCounts := make(map[string]int64)
	for _, item := range items {
		service, _ := spawn.GetServiceByResourceId(item.Resource.Id)
		instanceCounts[service]++
	}

	unhealthy := &UnhealthyServicesError{}
	clientSet := env.Kubernetes()
	fmt.Printf("checking health of %d services\n", len(items))
	for _, item := range items {
		service, index := spawn.GetServiceByResourceId(item.Resource.Id)

		if len(conditions) > 0 {
			if _, ok := conditions[service]; !ok {
				instructions.PrintOutput(ref, "service", ServiceInfo{Group: groupRef, Name: service, Index: index, Done: true})
			}
		}

		if issues := getServiceHealth(ctx, clientSet, item.Namespace, item.Resource.Id); len(issues) > 0 {
			name := instanceName(service, index, instanceCounts[service])
			fmt.Printf("%s: not healthy: %s\n", commontcl.ServiceLabel(name), strings.Join(issues, ", "))
			unhealthy.add(name, issues)
		}
	}

	if len(conditions) > 0 && machine != nil {
		services := make(map[string]int64)
		ids := make([]string, 0)
		for _, item := range items {
			service, index := spawn.GetServiceByResourceId(item.Resource.Id)
			if _, ok := conditions[service]; !ok {
				continue
			}
			serviceMachine := expressions.NewMachine().
				Register("index", index).
				RegisterAccessorExt(func(name string) (interface{}, bool, error) {
					if name == "count" {
						expr, err := expressions.CompileAndResolve(fmt.Sprintf("len(%s)", data.ServicesPrefix+service))
						return expr, true, err
					}
					return nil, false, nil
				})
			log, err := expressions.EvalExpression(conditions[service].String(), serviceMachine, machine)
			if err != nil {
				fmt.Printf("warning: service '%s': could not resolve condition '%s': %s", service, log.String(), err.Error())
			} else if v, _ := log.BoolValue(); v {
				services[service]++
				ids = append(ids, item.Resource.Id)
			}
		}

		for name, count := range services {
			fmt.Printf("%s: fetching logs of %d instances\n", commontcl.ServiceLabel(name), count)
		}

		storage, err := artifacts.InternalStorage()
		if err != nil {
			return fmt.Errorf("could not create internal storage client: %w", err)
		}
		for _, id := range ids {
			service, index := spawn.GetServiceByResourceId(id)
			count := index + 1
			if services[service] > count {
				count = services[service]
			}
			log := spawn.CreateLogger(service, "", index, count)

			logsFilePath, err := spawn.SaveLogs(context.Background(), storage, namespace, id, service+"/", index)
			if err == nil {
				instructions.PrintOutput(ref, "service", ServiceInfo{Group: groupRef, Name: service, Index: index, Logs: storage.FullPath(logsFilePath), Done: true})
				log("saved logs")
			} else {
				log("warning", "problem saving the logs", err.Error())
			}
		}
	}

	err = worker.DestroyGroup(ctx, groupRef, executionworkertypes.DestroyOptions{
		Namespace: namespace,
	})
	if err != nil {
		return fmt.Errorf("cleaning up resources: %w", err)
	}

	if len(unhealthy.services) > 0 {
		return unhealthy
	}

	return nil
}

// UnhealthyServicesError names each service instance that was not healthy when the step stopped
// the services, with the reasons that Kubernetes reported for it.
type UnhealthyServicesError struct {
	// OOMKilled is true when Kubernetes stopped a container of a service for its memory.
	OOMKilled bool
	services  []string
}

func (e *UnhealthyServicesError) add(name string, issues []string) {
	e.services = append(e.services, fmt.Sprintf("The service %q is not healthy: %s.", name, strings.Join(issues, ", ")))
	e.OOMKilled = e.OOMKilled || slices.Contains(issues, oomKilledReason)
}

func (e *UnhealthyServicesError) Error() string {
	return strings.Join(e.services, " ")
}

// oomKilledReason is the reason that Kubernetes gives a container that it stopped for its memory.
const oomKilledReason = "OOMKilled"

func getServiceHealth(ctx context.Context, clientSet kubernetes.Interface, namespace, resourceId string) []string {
	pods, err := clientSet.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "testkube.io/resource=" + resourceId,
		Limit:         1,
	})
	if err != nil {
		fmt.Printf("warning: could not list pods for service health check in namespace %s: %v\n", namespace, err)
		return nil
	}
	if len(pods.Items) == 0 {
		fmt.Printf("warning: no pod found for service health check: namespace=%s resource=%s\n", namespace, resourceId)
		return nil
	}

	return podHealthIssues(&pods.Items[0])
}

// podHealthIssues returns the reasons that Kubernetes reported for a pod of a service that is not
// healthy, each one time. The names of the containers are indexes of internal groups, so the
// reasons do not name them.
func podHealthIssues(pod *corev1.Pod) []string {
	var issues []string
	add := func(reason string) {
		if reason != "" && !slices.Contains(issues, reason) {
			issues = append(issues, reason)
		}
	}
	for _, cs := range pod.Status.ContainerStatuses {
		if term := cs.LastTerminationState.Terminated; term != nil && term.Reason != "Completed" {
			add(term.Reason)
		}
		if term := cs.State.Terminated; term != nil && term.Reason != "Completed" {
			add(term.Reason)
		}
		if waiting := cs.State.Waiting; waiting != nil && (waiting.Reason == "CrashLoopBackOff" || waiting.Reason == "Error") {
			add(waiting.Reason)
		}
	}

	if pod.Status.Phase == corev1.PodFailed {
		reason := pod.Status.Reason
		if reason == "" {
			reason = "the pod failed"
		}
		add(reason)
	}

	return issues
}
