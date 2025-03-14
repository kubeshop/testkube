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

	commontcl "github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/common"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/spawn"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/artifacts"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env/config"
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
			machine := expressions.CombinedMachines(data.AliasMachine, data.GetBaseTestWorkflowMachine())
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

			// Fetch the services when needed
			namespace := ""
			if len(logs) > 0 {
				items, err := spawn.ExecutionWorker().List(context.Background(), executionworkertypes.ListOptions{
					GroupId: groupRef,
				})
				ui.ExitOnError("listing service instances", err)

				if len(items) > 0 {
					namespace = items[0].Namespace
					for _, item := range items {
						if item.Namespace != namespace {
							namespace = ""
							break
						}
					}
				}

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

				for _, id := range ids {
					service, index := spawn.GetServiceByResourceId(id)
					count := index + 1
					if services[service] > count {
						count = services[service]
					}

					log := spawn.CreateLogger(service, "", index, count)
					notifications := spawn.ExecutionWorker().Notifications(context.Background(), id, executionworkertypes.NotificationsOptions{})
					if notifications.Err() != nil {
						log("error", "failed to connect to the service", notifications.Err().Error())
						continue
					}

					for l := range notifications.Channel() {
						if l.Result == nil || l.Result.Status == nil {
							continue
						}

						if l.Result.Status.Finished() {
							if l.Result.Initialization != nil && l.Result.Initialization.ErrorMessage != "" {
								log("warning", "initialization error", ui.Red(l.Result.Initialization.ErrorMessage))
							} else {
								for _, step := range l.Result.Steps {
									if step.ErrorMessage != "" {
										log("warning", "step error", ui.Red(step.ErrorMessage))
									}
								}
							}
						}
					}
				}

				// Inform about detected services
				for name, count := range services {
					fmt.Printf("%s: fetching logs of %d instances\n", commontcl.ServiceLabel(name), count)
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

					logsFilePath, err := spawn.SaveLogs(context.Background(), storage, config.Namespace(), id, service+"/", index)
					if err == nil {
						instructions.PrintOutput(config.Ref(), "service", ServiceInfo{Group: groupRef, Name: service, Index: index, Logs: storage.FullPath(logsFilePath)})
						log("saved logs")
					} else {
						log("warning", "problem saving the logs", err.Error())
					}
				}
			}

			err := spawn.ExecutionWorker().DestroyGroup(context.Background(), groupRef, executionworkertypes.DestroyOptions{
				Namespace: namespace,
			})
			ui.ExitOnError("cleaning up resources", err)
		},
	}

	cmd.Flags().StringArrayVarP(&logs, "logs", "l", nil, "fetch the logs for specific services - pair <name>=<expression>")

	return cmd
}
