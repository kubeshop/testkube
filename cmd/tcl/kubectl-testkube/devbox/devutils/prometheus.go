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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/kubeshop/testkube/internal/common"
)

type Prometheus struct {
	pod *PodObject
}

func NewPrometheus(pod *PodObject) *Prometheus {
	return &Prometheus{
		pod: pod,
	}
}

const (
	prometheusConfig = `
global:
  scrape_interval: 1s
  scrape_timeout: 500ms
  evaluation_interval: 1s

scrape_configs:
- job_name: 'Agent'
  honor_labels: true
  metrics_path: /metrics
  static_configs:
  - targets: [ 'devbox-agent:8088' ]`
)

func (r *Prometheus) Create(ctx context.Context) error {
	_, err := r.pod.ClientSet().CoreV1().ConfigMaps(r.pod.Namespace()).Create(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "devbox-prometheus-config"},
		Data:       map[string]string{"prometheus.yml": prometheusConfig},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	err = r.pod.Create(ctx, &corev1.Pod{
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: common.Ptr(int64(1)),
			Volumes: []corev1.Volume{
				{Name: "data", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
				{Name: "config", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: "devbox-prometheus-config"},
				}}},
			},
			Containers: []corev1.Container{
				{
					Name:            "prometheus",
					Image:           "prom/prometheus:v2.53.3",
					ImagePullPolicy: corev1.PullIfNotPresent,
					VolumeMounts: []corev1.VolumeMount{
						{Name: "data", MountPath: "/prometheus"},
						{Name: "config", MountPath: "/etc/prometheus/prometheus.yml", SubPath: "prometheus.yml"},
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							TCPSocket: &corev1.TCPSocketAction{
								Port: intstr.FromInt32(9090),
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
		Port:       9090,
		TargetPort: intstr.FromInt32(9090),
	})
}

func (r *Prometheus) WaitForReady(ctx context.Context) error {
	return r.pod.WaitForReady(ctx)
}
