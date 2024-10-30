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
	"github.com/kubeshop/testkube/pkg/cloud/client"
)

type Agent struct {
	pod              *PodObject
	cloud            *cloudObj
	agentImage       string
	initProcessImage string
	toolkitImage     string
}

func NewAgent(pod *PodObject, cloud *cloudObj, agentImage, initProcessImage, toolkitImage string) *Agent {
	return &Agent{
		pod:              pod,
		cloud:            cloud,
		agentImage:       agentImage,
		initProcessImage: initProcessImage,
		toolkitImage:     toolkitImage,
	}
}

func (r *Agent) Create(ctx context.Context, env *client.Environment) error {
	tlsInsecure := "false"
	if r.cloud.AgentInsecure() {
		tlsInsecure = "true"
	}
	err := r.pod.Create(ctx, &corev1.Pod{
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: common.Ptr(int64(1)),
			Volumes: []corev1.Volume{
				{Name: "tmp", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
				{Name: "nats", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
				{Name: "devbox", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
			},
			ServiceAccountName: "devbox-account",
			InitContainers: []corev1.Container{{
				Name:            "devbox-init",
				Image:           "busybox:1.36.1-musl",
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/bin/sh", "-c"},
				Args: []string{`
				/bin/wget -O /.tk-devbox/testkube-api-server http://devbox-binary:8080/testkube-api-server || exit 1
				chmod 777 /.tk-devbox/testkube-api-server
				chmod +x /.tk-devbox/testkube-api-server
				ls -lah /.tk-devbox`},
				VolumeMounts: []corev1.VolumeMount{
					{Name: "devbox", MountPath: "/.tk-devbox"},
				},
			}},
			Containers: []corev1.Container{
				{
					Name:            "server",
					Image:           r.agentImage,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{"/.tk-devbox/testkube-api-server"},
					Env: []corev1.EnvVar{
						{Name: "NATS_EMBEDDED", Value: "true"},
						{Name: "APISERVER_PORT", Value: "8088"},
						{Name: "APISERVER_FULLNAME", Value: "devbox-agent"},
						{Name: "DISABLE_TEST_TRIGGERS", Value: "true"},
						{Name: "DISABLE_WEBHOOKS", Value: "true"},
						{Name: "DISABLE_DEPRECATED_TESTS", Value: "true"},
						{Name: "TESTKUBE_ANALYTICS_ENABLED", Value: "false"},
						{Name: "TESTKUBE_NAMESPACE", Value: r.pod.Namespace()},
						{Name: "JOB_SERVICE_ACCOUNT_NAME", Value: "devbox-account"},
						{Name: "TESTKUBE_ENABLE_IMAGE_DATA_PERSISTENT_CACHE", Value: "true"},
						{Name: "TESTKUBE_IMAGE_DATA_PERSISTENT_CACHE_KEY", Value: "testkube-image-cache"},
						{Name: "TESTKUBE_TW_TOOLKIT_IMAGE", Value: r.toolkitImage},
						{Name: "TESTKUBE_TW_INIT_IMAGE", Value: r.initProcessImage},
						{Name: "TESTKUBE_PRO_API_KEY", Value: env.AgentToken},
						{Name: "TESTKUBE_PRO_ORG_ID", Value: env.OrganizationId},
						{Name: "TESTKUBE_PRO_ENV_ID", Value: env.Id},
						{Name: "TESTKUBE_PRO_URL", Value: r.cloud.AgentURI()},
						{Name: "TESTKUBE_PRO_TLS_INSECURE", Value: tlsInsecure},
						{Name: "TESTKUBE_PRO_TLS_SKIP_VERIFY", Value: "true"},
					},
					VolumeMounts: []corev1.VolumeMount{
						{Name: "tmp", MountPath: "/tmp"},
						{Name: "nats", MountPath: "/app/nats"},
						{Name: "devbox", MountPath: "/.tk-devbox"},
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/health",
								Port:   intstr.FromInt32(8088),
								Scheme: corev1.URISchemeHTTP,
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
	err = r.pod.WaitForContainerStarted(ctx)
	if err != nil {
		return err
	}
	return r.pod.CreateService(ctx, corev1.ServicePort{
		Name:       "api",
		Protocol:   "TCP",
		Port:       8088,
		TargetPort: intstr.FromInt32(8088),
	})
}

func (r *Agent) WaitForReady(ctx context.Context) error {
	return r.pod.WaitForReady(ctx)
}

func (r *Agent) Restart(ctx context.Context) error {
	return r.pod.Restart(ctx)
}
