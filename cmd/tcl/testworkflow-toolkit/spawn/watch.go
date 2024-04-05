// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package spawn

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor/constants"
)

func WatchPods(ctx context.Context, clientSet kubernetes.Interface, ref string, servicesMap map[string]Service, fn func(svc Service, index int64, pod *corev1.Pod)) error {
	podWatch, err := clientSet.CoreV1().Pods(env.Namespace()).Watch(ctx, metav1.ListOptions{
		TypeMeta:      metav1.TypeMeta{Kind: "Pod"},
		LabelSelector: fmt.Sprintf("%s=%s", constants.ExecutionAssistingPodRefName, ref),
	})
	if err != nil {
		return err
	}

	// Prepare semaphores for each service instance
	sema := make(map[string][]chan struct{}, len(servicesMap))
	for _, svc := range servicesMap {
		arr := make([]chan struct{}, svc.Total())
		for i := range arr {
			arr[i] = make(chan struct{}, 1)
		}
		sema[svc.Name] = arr
	}

	// Iterate over all pod updates
	go func() {
		defer podWatch.Stop()

		select {
		case <-ctx.Done():
			return
		default:
			for {
				select {
				case <-ctx.Done():
					return
				case ev, ok := <-podWatch.ResultChan():
					if !ok {
						return
					}
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
						sema[name][index] <- struct{}{}
						fn(servicesMap[name], index, pod)
						<-sema[name][index]
					}
				}
			}
		}
	}()

	return nil
}

func DeletePod(ctx context.Context, clientSet kubernetes.Interface, svc Service, ref string, index int64) error {
	podName := fmt.Sprintf("%s-%s-%s-%d", env.ExecutionId(), ref, svc.Name, index)
	err := clientSet.CoreV1().Pods(env.Namespace()).Delete(ctx, podName, metav1.DeleteOptions{
		GracePeriodSeconds: common.Ptr(int64(0)),
		PropagationPolicy:  common.Ptr(metav1.DeletePropagationBackground),
	})
	if err != nil && errors.IsNotFound(err) {
		err = nil
	}
	return err
}
