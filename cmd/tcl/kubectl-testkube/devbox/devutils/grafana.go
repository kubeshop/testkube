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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/kubeshop/testkube/internal/common"
)

type Grafana struct {
	pod       *PodObject
	localPort int
}

func NewGrafana(pod *PodObject) *Grafana {
	return &Grafana{
		pod: pod,
	}
}

const (
	grafanaProvisioningPrometheusDataSource = `
apiVersion: 1

deleteDatasources:
  - name: Prometheus
    orgId: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    orgId: 1
    url: http://devbox-prometheus:9090`
)

func (r *Grafana) Create(ctx context.Context) error {
	_, err := r.pod.ClientSet().CoreV1().ConfigMaps(r.pod.Namespace()).Create(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "devbox-grafana-provisioning-datasources"},
		Data: map[string]string{
			"prometheus.yml": grafanaProvisioningPrometheusDataSource,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	err = r.pod.Create(ctx, &corev1.Pod{
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: common.Ptr(int64(1)),
			Volumes: []corev1.Volume{
				{Name: "data", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
				{Name: "provisioning-datasources", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: "devbox-grafana-provisioning-datasources"},
				}}},
			},
			Containers: []corev1.Container{
				{
					Name:            "prometheus",
					Image:           "grafana/grafana-oss:11.3.1",
					ImagePullPolicy: corev1.PullIfNotPresent,
					Env: []corev1.EnvVar{
						{Name: "GF_USERS_ALLOW_SIGN_UP", Value: "false"},
						{Name: "GF_AUTH_ANONYMOUS_ENABLED", Value: "true"},
						{Name: "GF_AUTH_ANONYMOUS_ORG_ROLE", Value: "Admin"},
						{Name: "GF_AUTH_DISABLE_LOGIN_FORM", Value: "true"},
					},
					VolumeMounts: []corev1.VolumeMount{
						{Name: "data", MountPath: "/var/lib/grafana"},
						{Name: "provisioning-datasources", MountPath: "/etc/grafana/provisioning/datasources"},
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							TCPSocket: &corev1.TCPSocketAction{
								Port: intstr.FromInt32(3000),
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

	err = r.pod.CreateService(ctx, corev1.ServicePort{
		Name:       "web",
		Protocol:   "TCP",
		Port:       3000,
		TargetPort: intstr.FromInt32(3000),
	})
	if err != nil {
		return err
	}

	err = r.pod.WaitForReady(ctx)
	if err != nil {
		return err
	}

	r.localPort = GetFreePort()
	err = r.pod.Forward(ctx, 3000, r.localPort, true)
	if err != nil {
		return err
	}

	return nil
}

func (r *Grafana) LocalAddress() string {
	return fmt.Sprintf("http://localhost:%d", r.localPort)
}

func (r *Grafana) WaitForReady(ctx context.Context) error {
	return r.pod.WaitForReady(ctx)
}
