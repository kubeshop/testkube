// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowprocessor

import (
	"context"

	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Bundle struct {
	Secrets    []corev1.Secret
	ConfigMaps []corev1.ConfigMap
	Job        batchv1.Job
	Signature  []Signature
}

func (b *Bundle) Deploy(ctx context.Context, clientSet kubernetes.Interface, namespace string) (err error) {
	if b.Job.Namespace != "" {
		namespace = b.Job.Namespace
	}
	for _, item := range b.Secrets {
		_, err = clientSet.CoreV1().Secrets(namespace).Create(ctx, &item, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to deploy secrets")
		}
	}
	for _, item := range b.ConfigMaps {
		_, err = clientSet.CoreV1().ConfigMaps(namespace).Create(ctx, &item, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to deploy config maps")
		}
	}
	_, err = clientSet.BatchV1().Jobs(namespace).Create(ctx, &b.Job, metav1.CreateOptions{})
	return errors.Wrap(err, "failed to deploy job")
}
