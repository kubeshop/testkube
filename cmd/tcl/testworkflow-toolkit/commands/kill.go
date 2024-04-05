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
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/artifacts"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/spawn"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor/constants"
)

func NewKillCmd() *cobra.Command {
	var (
		logsRequest []string
	)
	cmd := &cobra.Command{
		Use:   "kill",
		Short: "Kill assisting pods",
		Args:  cobra.ExactArgs(1),

		Run: func(cmd *cobra.Command, args []string) {
			podsRef := args[0]
			failed := false

			// Initialize Kubernetes client
			clientSet := env.Kubernetes()
			artifacts := artifacts.NewInternalArtifactStorage()

			// Find all pods to kill
			pods, err := clientSet.CoreV1().Pods(env.Namespace()).List(context.Background(), metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=%s,%s=%s",
					constants.ExecutionIdLabelName, env.ExecutionId(),
					constants.ExecutionAssistingPodRefName, podsRef),
			})
			if err != nil {
				fmt.Printf("failed to delete assisting pods: %s\n", err.Error())
				failed = true
			}

			// Process and delete pods
			if err == nil {
				for _, pod := range pods.Items {
					segments := strings.Split(pod.Name, "-")
					name := segments[2]
					index, err := strconv.ParseInt(segments[3], 10, 64)
					svc := spawn.Service{Name: name}

					if err == nil && slices.Contains(logsRequest, name) {
						err = spawn.DeletePodAndSaveLogs(context.Background(), clientSet, artifacts, svc, &pod, podsRef, index)
					} else {
						err = spawn.DeletePod(context.Background(), clientSet, &pod)
					}

					if err == nil {
						fmt.Printf("%s: deleted pod successfully\n", spawn.InstanceLabel(name, index, index))
					} else {
						fmt.Printf("%s: failed to delete pod: %s: %s\n", spawn.InstanceLabel(name, index, index), pod.Name, err.Error())
						failed = true
					}
				}
			}

			// Delete config maps
			err = clientSet.CoreV1().ConfigMaps(env.Namespace()).DeleteCollection(context.Background(), metav1.DeleteOptions{
				GracePeriodSeconds: common.Ptr(int64(0)),
				PropagationPolicy:  common.Ptr(metav1.DeletePropagationBackground),
			}, metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=%s,%s=%s",
					constants.ExecutionIdLabelName, env.ExecutionId(),
					constants.ExecutionAssistingPodRefName, podsRef),
			})
			if err != nil {
				fmt.Printf("failed to delete configmaps of assisting pods: %s\n", err.Error())
				failed = true
			}

			if failed {
				os.Exit(1)
			}
		},
	}

	cmd.Flags().StringArrayVarP(&logsRequest, "logs", "l", nil, "service names to fetch logs for")

	return cmd
}
