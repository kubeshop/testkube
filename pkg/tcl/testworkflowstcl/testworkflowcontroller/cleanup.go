// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowcontroller

import (
	"context"
	"errors"
	"fmt"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor/constants"
)

func cleanupConfigMaps(labelName string) func(ctx context.Context, clientSet kubernetes.Interface, namespace, id string) error {
	return func(ctx context.Context, clientSet kubernetes.Interface, namespace, id string) error {
		return clientSet.CoreV1().ConfigMaps(namespace).DeleteCollection(ctx, metav1.DeleteOptions{
			GracePeriodSeconds: common.Ptr(int64(0)),
			PropagationPolicy:  common.Ptr(metav1.DeletePropagationBackground),
		}, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", labelName, id),
		})
	}
}

func cleanupSecrets(labelName string) func(ctx context.Context, clientSet kubernetes.Interface, namespace, id string) error {
	return func(ctx context.Context, clientSet kubernetes.Interface, namespace, id string) error {
		return clientSet.CoreV1().Secrets(namespace).DeleteCollection(ctx, metav1.DeleteOptions{
			GracePeriodSeconds: common.Ptr(int64(0)),
			PropagationPolicy:  common.Ptr(metav1.DeletePropagationBackground),
		}, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", labelName, id),
		})
	}
}

func cleanupPods(labelName string) func(ctx context.Context, clientSet kubernetes.Interface, namespace, id string) error {
	return func(ctx context.Context, clientSet kubernetes.Interface, namespace, id string) error {
		return clientSet.CoreV1().Pods(namespace).DeleteCollection(ctx, metav1.DeleteOptions{
			GracePeriodSeconds: common.Ptr(int64(0)),
			PropagationPolicy:  common.Ptr(metav1.DeletePropagationBackground),
		}, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", labelName, id),
		})
	}
}

func cleanupJobs(labelName string) func(ctx context.Context, clientSet kubernetes.Interface, namespace, id string) error {
	return func(ctx context.Context, clientSet kubernetes.Interface, namespace, id string) error {
		return clientSet.BatchV1().Jobs(namespace).DeleteCollection(ctx, metav1.DeleteOptions{
			GracePeriodSeconds: common.Ptr(int64(0)),
			PropagationPolicy:  common.Ptr(metav1.DeletePropagationBackground),
		}, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", labelName, id),
		})
	}
}

func Cleanup(ctx context.Context, clientSet kubernetes.Interface, namespace, id string) error {
	var errs []error
	var errsMu sync.Mutex
	var wg sync.WaitGroup
	ops := []func(context.Context, kubernetes.Interface, string, string) error{
		cleanupJobs(constants.RootResourceIdLabelName),
		cleanupJobs(constants.ResourceIdLabelName),
		cleanupPods(constants.RootResourceIdLabelName),
		cleanupPods(constants.ResourceIdLabelName),
		cleanupConfigMaps(constants.RootResourceIdLabelName),
		cleanupConfigMaps(constants.ResourceIdLabelName),
		cleanupSecrets(constants.RootResourceIdLabelName),
		cleanupSecrets(constants.ResourceIdLabelName),
	}
	wg.Add(len(ops))
	for _, op := range ops {
		go func(op func(context.Context, kubernetes.Interface, string, string) error) {
			err := op(ctx, clientSet, namespace, id)
			if err != nil {
				errsMu.Lock()
				errs = append(errs, err)
				errsMu.Unlock()
			}
			wg.Done()
		}(op)
	}
	wg.Wait()
	return errors.Join(errs...)
}
