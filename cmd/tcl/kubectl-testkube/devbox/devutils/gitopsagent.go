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
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/cloud/client"
)

type GitOpsAgent struct {
	pod                      *PodObject
	cloud                    *CloudObject
	agentImage               string
	cloudToKubernetesEnabled bool
	kubernetesToCloudEnabled bool
	cloudPattern             string
	kubernetesPattern        string
}

func NewGitOpsAgent(
	pod *PodObject,
	cloud *CloudObject,
	agentImage string,
	cloudToKubernetesEnabled, kubernetesToCloudEnabled bool,
	cloudPattern, kubernetesPattern string,
) *GitOpsAgent {
	return &GitOpsAgent{
		pod:                      pod,
		cloud:                    cloud,
		agentImage:               agentImage,
		cloudToKubernetesEnabled: cloudToKubernetesEnabled,
		kubernetesToCloudEnabled: kubernetesToCloudEnabled,
		cloudPattern:             cloudPattern,
		kubernetesPattern:        kubernetesPattern,
	}
}

func (r *GitOpsAgent) Create(ctx context.Context, env *client.Environment, agent *client.Agent) error {
	if env == nil || agent == nil {
		panic("crd sync is not supported in OSS")
	}

	envVariables := []corev1.EnvVar{
		// Disabling the rest
		{Name: "NATS_EMBEDDED", Value: "true"},
		{Name: "TESTKUBE_ANALYTICS_ENABLED", Value: "false"},
		{Name: "DISABLE_TEST_TRIGGERS", Value: "true"},
		{Name: "DISABLE_WEBHOOKS", Value: "true"},
		{Name: "DISABLE_DEPRECATED_TESTS", Value: "true"},
		{Name: "DISABLE_RUNNER", Value: "true"},
		{Name: "DISABLE_DEFAULT_AGENT", Value: "true"},

		// Cloud connection
		{Name: "TESTKUBE_PRO_AGENT_ID", Value: agent.ID},
		{Name: "TESTKUBE_PRO_API_KEY", Value: agent.SecretKey},
		{Name: "TESTKUBE_PRO_ORG_ID", Value: env.OrganizationId},
		{Name: "TESTKUBE_PRO_ENV_ID", Value: env.Id},
		{Name: "TESTKUBE_PRO_URL", Value: r.cloud.AgentURI()},
		{Name: "TESTKUBE_PRO_TLS_INSECURE", Value: fmt.Sprintf("%v", r.cloud.AgentInsecure())},
		{Name: "TESTKUBE_PRO_TLS_SKIP_VERIFY", Value: "true"},

		// CRD Sync configuration
		{Name: "TESTKUBE_NAMESPACE", Value: r.pod.Namespace()},
		{Name: "GITOPS_KUBERNETES_TO_CLOUD_ENABLED", Value: fmt.Sprintf("%v", r.kubernetesToCloudEnabled)},
		{Name: "GITOPS_CLOUD_TO_KUBERNETES_ENABLED", Value: fmt.Sprintf("%v", r.cloudToKubernetesEnabled)},
		{Name: "GITOPS_CLOUD_NAME_PATTERN", Value: r.cloudPattern},
		{Name: "GITOPS_KUBERNETES_NAME_PATTERN", Value: r.kubernetesPattern},

		// Feature flags
		{Name: "FEATURE_NEW_ARCHITECTURE", Value: "true"},
		{Name: "FEATURE_CLOUD_STORAGE", Value: "true"},
	}
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

func (r *GitOpsAgent) WaitForReady(ctx context.Context) error {
	return r.pod.WaitForReady(ctx)
}

func (r *GitOpsAgent) Restart(ctx context.Context) error {
	return r.pod.Restart(ctx)
}
