// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package devutils

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/kubeshop/testkube/internal/common"
)

type Mongo struct {
	pod *PodObject
}

func NewMongo(pod *PodObject) *Mongo {
	return &Mongo{
		pod: pod,
	}
}

func (r *Mongo) Create(ctx context.Context) error {
	err := r.pod.Create(ctx, &corev1.Pod{
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: common.Ptr(int64(1)),
			Containers: []corev1.Container{
				{
					Name:            "mongo",
					Image:           "zcube/bitnami-compat-mongodb:6.0.5-debian-11-r64",
					ImagePullPolicy: corev1.PullIfNotPresent,
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							TCPSocket: &corev1.TCPSocketAction{
								Port: intstr.FromInt32(27017),
							},
						},
						PeriodSeconds: 1,
					},
				},
			},
		},
	})
	if err != nil {
		return err
	}
	return r.pod.CreateService(ctx, corev1.ServicePort{
		Name:       "api",
		Protocol:   "TCP",
		Port:       27017,
		TargetPort: intstr.FromInt32(27017),
	})
}

func (r *Mongo) WaitForReady(ctx context.Context) error {
	return r.pod.WaitForReady(ctx)
}
