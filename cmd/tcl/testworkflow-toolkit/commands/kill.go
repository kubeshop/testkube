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

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor"
)

func NewKillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kill",
		Short: "Kill assisting pods",
		Args:  cobra.ExactArgs(1),

		Run: func(cmd *cobra.Command, args []string) {
			podsRef := args[0]
			failed := false

			// Initialize Kubernetes client
			clientSet := env.Kubernetes()

			// Delete pods
			err := clientSet.CoreV1().Pods(env.Namespace()).DeleteCollection(context.Background(), metav1.DeleteOptions{
				GracePeriodSeconds: common.Ptr(int64(0)),
				PropagationPolicy:  common.Ptr(metav1.DeletePropagationBackground),
			}, metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=%s,%s=%s",
					testworkflowprocessor.ExecutionIdLabelName, env.ExecutionId(),
					testworkflowprocessor.ExecutionAssistingPodRefName, podsRef),
			})
			if err != nil {
				fmt.Printf("failed to delete assisting pods: %s\n", err.Error())
				failed = true
			}

			// Delete config maps
			err = clientSet.CoreV1().ConfigMaps(env.Namespace()).DeleteCollection(context.Background(), metav1.DeleteOptions{
				GracePeriodSeconds: common.Ptr(int64(0)),
				PropagationPolicy:  common.Ptr(metav1.DeletePropagationBackground),
			}, metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=%s,%s=%s",
					testworkflowprocessor.ExecutionIdLabelName, env.ExecutionId(),
					testworkflowprocessor.ExecutionAssistingPodRefName, podsRef),
			})
			if err != nil {
				fmt.Printf("failed to delete configmaps of assisting pods: %s\n", err.Error())
				failed = true
			}

			if failed {
				os.Exit(1)
			}
			fmt.Printf("deleted assisting pods successfully\n")
		},
	}

	return cmd
}
