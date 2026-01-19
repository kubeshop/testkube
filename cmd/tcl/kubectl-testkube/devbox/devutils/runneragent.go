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

	errors2 "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/cloud/client"
)

type RunnerAgent struct {
	pod                *PodObject
	cloud              *CloudObject
	agentImage         string
	initProcessImage   string
	toolkitImage       string
	executionNamespace string
	env                *client.Environment // Store environment for pod recreation
	runner             *client.Agent       // Store runner config for pod recreation
}

func NewRunnerAgent(pod *PodObject, cloud *CloudObject, agentImage, initProcessImage, toolkitImage, executionNamespace string) *RunnerAgent {
	return &RunnerAgent{
		pod:                pod,
		cloud:              cloud,
		agentImage:         agentImage,
		initProcessImage:   initProcessImage,
		toolkitImage:       toolkitImage,
		executionNamespace: executionNamespace,
	}
}

func (r *RunnerAgent) generatePodSpec(env *client.Environment, runner *client.Agent) *corev1.Pod {
	envVariables := []corev1.EnvVar{
		// Disabling the rest
		{Name: "NATS_EMBEDDED", Value: "true"},
		{Name: "TESTKUBE_ANALYTICS_ENABLED", Value: "false"},
		{Name: "DISABLE_TEST_TRIGGERS", Value: "true"},
		{Name: "DISABLE_WEBHOOKS", Value: "true"},
		{Name: "DISABLE_DEPRECATED_TESTS", Value: "true"},
		{Name: "DISABLE_DEFAULT_AGENT", Value: "true"},

		// Cloud connection
		{Name: "TESTKUBE_PRO_AGENT_ID", Value: runner.ID},
		{Name: "TESTKUBE_PRO_API_KEY", Value: runner.SecretKey},
		{Name: "TESTKUBE_PRO_ORG_ID", Value: env.OrganizationId},
		{Name: "TESTKUBE_PRO_ENV_ID", Value: env.Id},
		{Name: "TESTKUBE_PRO_URL", Value: r.cloud.AgentURI()},
		{Name: "TESTKUBE_PRO_TLS_INSECURE", Value: fmt.Sprintf("%v", r.cloud.AgentInsecure())},
		{Name: "TESTKUBE_PRO_TLS_SKIP_VERIFY", Value: "true"},

		// Runner configuration
		{Name: "APISERVER_FULLNAME", Value: "devbox-agent"},
		{Name: "TESTKUBE_NAMESPACE", Value: r.pod.Namespace()},
		{Name: "DEFAULT_EXECUTION_NAMESPACE", Value: r.executionNamespace},
		{Name: "JOB_SERVICE_ACCOUNT_NAME", Value: jobServiceAccountName},
		{Name: "TESTKUBE_ENABLE_IMAGE_DATA_PERSISTENT_CACHE", Value: "true"},
		{Name: "TESTKUBE_IMAGE_DATA_PERSISTENT_CACHE_KEY", Value: "testkube-image-cache"},
		{Name: "TESTKUBE_TW_TOOLKIT_IMAGE", Value: r.toolkitImage},
		{Name: "TESTKUBE_TW_INIT_IMAGE", Value: r.initProcessImage},

		// Feature flags
		{Name: "FEATURE_CLOUD_STORAGE", Value: "true"},
	}
	return &corev1.Pod{
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
	}
}

func (r *RunnerAgent) Create(ctx context.Context, env *client.Environment, runner *client.Agent) error {
	if env == nil || runner == nil {
		return errors2.New("runner agent requires cloud environment and runner configuration (not available in OSS mode)")
	}

	r.env = env
	r.runner = runner
	podSpec := r.generatePodSpec(env, runner)

	return r.pod.CreateWithFunc(ctx, podSpec, func() (*corev1.Pod, error) {
		return r.generatePodSpec(r.env, r.runner), nil
	})
}

func (r *RunnerAgent) WaitForReady(ctx context.Context) error {
	return r.pod.WaitForReady(ctx)
}

func (r *RunnerAgent) Restart(ctx context.Context) error {
	return r.pod.Restart(ctx)
}
