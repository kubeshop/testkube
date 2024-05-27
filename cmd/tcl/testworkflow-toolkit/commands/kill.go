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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/artifacts"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/common"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/spawn"
	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowcontroller"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor/constants"
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
			machine := expressionstcl.CombinedMachines(data.AliasMachine, data.GetBaseTestWorkflowMachine())
			groupRef := args[0]
			clientSet := env.Kubernetes()

			conditions := make(map[string]expressionstcl.Expression)
			for _, l := range logs {
				name, condition, found := strings.Cut(l, "=")
				if !found {
					condition = "true"
				}
				expr, err := expressionstcl.CompileAndResolve(condition, machine)
				if err != nil {
					fmt.Printf("warning: service '%s': could not compile condition '%s': %s", name, condition, err.Error())
				} else {
					conditions[name] = expr
				}
			}

			// Fetch the services when needed
			if len(logs) > 0 {
				jobs, err := clientSet.BatchV1().Jobs(env.Namespace()).List(context.Background(), metav1.ListOptions{
					LabelSelector: fmt.Sprintf("%s=%s", constants.GroupIdLabelName, groupRef),
				})
				ui.ExitOnError("listing service resources", err)

				services := make(map[string]int64)
				ids := make([]string, 0)
				for _, job := range jobs.Items {
					service, index := spawn.GetServiceByResourceId(job.Name)
					if _, ok := conditions[service]; !ok {
						continue
					}
					serviceMachine := expressionstcl.NewMachine().
						Register("index", index).
						RegisterAccessorExt(func(name string) (interface{}, bool, error) {
							if name == "count" {
								expr, err := expressionstcl.CompileAndResolve(fmt.Sprintf("len(services.%s)", service))
								return expr, true, err
							}
							return nil, false, nil
						})
					log, err := expressionstcl.EvalExpression(conditions[service].String(), serviceMachine, machine)
					if err != nil {
						fmt.Printf("warning: service '%s': could not resolve condition '%s': %s", service, log.String(), err.Error())
					} else if v, _ := log.BoolValue(); v {
						services[service]++
						ids = append(ids, job.Name)
					}
				}

				// Inform about detected services
				for name, count := range services {
					fmt.Printf("%s: fetching logs of %d instances\n", common.ServiceLabel(name), count)
				}

				// Fetch logs for them
				storage := artifacts.InternalStorage()
				for _, id := range ids {
					service, index := spawn.GetServiceByResourceId(id)
					count := index + 1
					if services[service] > count {
						count = services[service]
					}
					log := spawn.CreateLogger(service, "", index, count)

					logsFilePath, err := spawn.SaveLogs(context.Background(), clientSet, storage, env.Namespace(), id, service+"/", index)
					if err == nil {
						data.PrintOutput(env.Ref(), "service", ServiceInfo{Group: groupRef, Name: service, Index: index, Logs: storage.FullPath(logsFilePath)})
						log("saved logs")
					} else {
						log("warning", "problem saving the logs", err.Error())
					}
				}
			}

			err := testworkflowcontroller.CleanupGroup(context.Background(), clientSet, env.Namespace(), groupRef)
			ui.ExitOnError("cleaning up resources", err)
		},
	}

	cmd.Flags().StringArrayVarP(&logs, "logs", "l", nil, "fetch the logs for specific services - pair <name>=<expression>")

	return cmd
}
