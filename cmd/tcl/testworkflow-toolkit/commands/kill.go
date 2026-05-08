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
	"fmt"
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
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env/config"
	"github.com/kubeshop/testkube/pkg/credentials"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/ui"
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
			ui.ExitOnError("stopping services", err)
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

	var healthErrors []string
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
			healthErrors = append(healthErrors, fmt.Sprintf("%s (index %d): %s", commontcl.ServiceLabel(service), index, strings.Join(issues, "; ")))
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

	if len(healthErrors) > 0 {
		return fmt.Errorf("unhealthy services detected: %s", strings.Join(healthErrors, "; "))
	}

	return nil
}

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

	pod := &pods.Items[0]
	var issues []string
	for _, cs := range pod.Status.ContainerStatuses {
		if term := cs.LastTerminationState.Terminated; term != nil && term.Reason != "Completed" && term.Reason != "" {
			issues = append(issues, fmt.Sprintf("container %q previously terminated: %s", cs.Name, term.Reason))
		}
		if term := cs.State.Terminated; term != nil && term.Reason != "Completed" && term.Reason != "" {
			issues = append(issues, fmt.Sprintf("container %q terminated: %s", cs.Name, term.Reason))
		}
		if waiting := cs.State.Waiting; waiting != nil && (waiting.Reason == "CrashLoopBackOff" || waiting.Reason == "Error") {
			issues = append(issues, fmt.Sprintf("container %q in state: %s", cs.Name, waiting.Reason))
		}
	}

	if pod.Status.Phase == corev1.PodFailed {
		reason := pod.Status.Reason
		if reason == "" {
			reason = "pod failed"
		}
		issues = append(issues, reason)
	}

	return issues
}
