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

type Minio struct {
	pod *PodObject
}

func NewMinio(pod *PodObject) *Minio {
	return &Minio{
		pod: pod,
	}
}

// minioImage is the image the testkube chart deploys MinIO with. MinIO no longer
// publishes public images, so minio/minio and quay.io/minio/minio do not pull.
// It runs as a non-root user, so the data directory has to be a mounted volume.
const minioImage = "kubeshop/testkube-minio:2025.10"

func minioPod() *corev1.Pod {
	return &corev1.Pod{
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: common.Ptr(int64(1)),
			Volumes: []corev1.Volume{
				{Name: "data", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
			},
			Containers: []corev1.Container{
				{
					Name:            "minio",
					Image:           minioImage,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{"minio"},
					Args:            []string{"server", "/data", "--console-address", ":9090"},
					VolumeMounts: []corev1.VolumeMount{
						{Name: "data", MountPath: "/data"},
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							TCPSocket: &corev1.TCPSocketAction{
								Port: intstr.FromInt32(9000),
							},
						},
						PeriodSeconds: 1,
					},
				},
			},
		},
	}
}

func (r *Minio) Create(ctx context.Context) error {
	err := r.pod.Create(ctx, minioPod())
	if err != nil {
		return err
	}
	return r.pod.CreateService(ctx, corev1.ServicePort{
		Name:       "api",
		Protocol:   "TCP",
		Port:       9000,
		TargetPort: intstr.FromInt32(9000),
	})
}

func (r *Minio) WaitForReady(ctx context.Context) error {
	return r.pod.WaitForReady(ctx)
}
