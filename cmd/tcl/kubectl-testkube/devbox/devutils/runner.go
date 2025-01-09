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

type Runner struct {
	pod              *PodObject
	cloud            *CloudObject
	agentImage       string
	initProcessImage string
	toolkitImage     string
}

func NewRunner(pod *PodObject, cloud *CloudObject, agentImage, initProcessImage, toolkitImage string) *Runner {
	return &Runner{
		pod:              pod,
		cloud:            cloud,
		agentImage:       agentImage,
		initProcessImage: initProcessImage,
		toolkitImage:     toolkitImage,
	}
}

func (r *Runner) Create(ctx context.Context, env *client.Environment, runner *client.Agent) error {
	envVariables := []corev1.EnvVar{
		{Name: "NATS_EMBEDDED", Value: "true"},
		{Name: "APISERVER_PORT", Value: "8088"},
		{Name: "GRPC_PORT", Value: "8089"},
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
		{Name: "FEATURE_NEW_EXECUTIONS", Value: "true"},
		{Name: "FEATURE_TESTWORKFLOW_CLOUD_STORAGE", Value: "true"},
	}
	if env == nil || runner == nil {
		panic("runner is not supported in OSS")
	}
	tlsInsecure := "false"
	if r.cloud.AgentInsecure() {
		tlsInsecure = "true"
	}
	envVariables = append(envVariables, []corev1.EnvVar{
		{Name: "TESTKUBE_DISABLE_DEFAULT_AGENT", Value: "true"},
		{Name: "TESTKUBE_PRO_AGENT_ID", Value: runner.Name},
		{Name: "TESTKUBE_PRO_API_KEY", Value: runner.SecretKey},
		{Name: "TESTKUBE_PRO_ORG_ID", Value: env.OrganizationId},
		{Name: "TESTKUBE_PRO_ENV_ID", Value: env.Id},
		{Name: "TESTKUBE_PRO_URL", Value: r.cloud.AgentURI()},
		{Name: "TESTKUBE_PRO_TLS_INSECURE", Value: tlsInsecure},
		{Name: "TESTKUBE_PRO_TLS_SKIP_VERIFY", Value: "true"},
	}...)
	return r.pod.Create(ctx, &corev1.Pod{
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: common.Ptr(int64(1)),
			Volumes: []corev1.Volume{
				{Name: "tmp", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
				{Name: "nats", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
				{Name: "devbox", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
			},
			ServiceAccountName: "devbox-account",
			Containers: []corev1.Container{
				{
					Name:            "server",
					Image:           r.agentImage,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{"/bin/sh", "-c"},
					Args: []string{`
						wget -O /.tk-devbox/testkube-api-server http://devbox-binary:8080/testkube-api-server || exit 1
						chmod 777 /.tk-devbox/testkube-api-server
						exec /.tk-devbox/testkube-api-server`},
					Env: envVariables,
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
}

func (r *Runner) WaitForReady(ctx context.Context) error {
	return r.pod.WaitForReady(ctx)
}

func (r *Runner) Restart(ctx context.Context) error {
	return r.pod.Restart(ctx)
}
