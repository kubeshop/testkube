// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package devbox

import (
	"context"
	"errors"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/cloud/client"
)

type agentObj struct {
	clientSet    kubernetes.Interface
	namespace    string
	cfg          AgentConfig
	pod          *corev1.Pod
	localPort    int
	localWebPort int
}

type AgentConfig struct {
	AgentImage   string
	ToolkitImage string
	InitImage    string
}

func NewAgent(clientSet kubernetes.Interface, namespace string, cfg AgentConfig) *agentObj {
	return &agentObj{
		clientSet: clientSet,
		namespace: namespace,
		cfg:       cfg,
	}
}

func (r *agentObj) Deploy(env client.Environment, cloud *cloudObj) (err error) {
	tlsInsecure := "false"
	if cloud.AgentInsecure() {
		tlsInsecure = "true"
	}
	r.pod, err = r.clientSet.CoreV1().Pods(r.namespace).Create(context.Background(), &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "devbox-agent",
			Labels: map[string]string{
				"testkube.io/devbox": "agent",
			},
		},
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: common.Ptr(int64(1)),
			Volumes: []corev1.Volume{
				{Name: "tmp", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
				{Name: "nats", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
			},
			ServiceAccountName: "devbox-account",
			Containers: []corev1.Container{
				{
					Name:  "server",
					Image: r.cfg.AgentImage,
					Env: []corev1.EnvVar{
						{Name: "NATS_EMBEDDED", Value: "true"},
						{Name: "APISERVER_PORT", Value: "8088"},
						{Name: "APISERVER_FULLNAME", Value: "devbox-agent"},
						{Name: "DISABLE_TEST_TRIGGERS", Value: "true"},
						{Name: "DISABLE_WEBHOOKS", Value: "true"},
						{Name: "DISABLE_DEPRECATED_TESTS", Value: "true"},
						{Name: "TESTKUBE_ANALYTICS_ENABLED", Value: "false"},
						{Name: "TESTKUBE_NAMESPACE", Value: r.namespace},
						{Name: "JOB_SERVICE_ACCOUNT_NAME", Value: "devbox-account"},
						{Name: "TESTKUBE_ENABLE_IMAGE_DATA_PERSISTENT_CACHE", Value: "true"},
						{Name: "TESTKUBE_IMAGE_DATA_PERSISTENT_CACHE_KEY", Value: "testkube-image-cache"},
						{Name: "TESTKUBE_TW_TOOLKIT_IMAGE", Value: r.cfg.ToolkitImage},
						{Name: "TESTKUBE_TW_INIT_IMAGE", Value: r.cfg.InitImage},
						{Name: "TESTKUBE_PRO_API_KEY", Value: env.AgentToken},
						{Name: "TESTKUBE_PRO_ORG_ID", Value: env.OrganizationId},
						{Name: "TESTKUBE_PRO_ENV_ID", Value: env.Id},
						{Name: "TESTKUBE_PRO_URL", Value: cloud.AgentURI()},
						{Name: "TESTKUBE_PRO_TLS_INSECURE", Value: tlsInsecure},
						{Name: "TESTKUBE_PRO_TLS_SKIP_VERIFY", Value: "true"},
					},
					VolumeMounts: []corev1.VolumeMount{
						{Name: "tmp", MountPath: "/tmp"},
						{Name: "nats", MountPath: "/app/nats"},
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
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Create the service
	_, err = r.clientSet.CoreV1().Services(r.namespace).Create(context.Background(), &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "devbox-agent",
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"testkube.io/devbox": "agent",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "api",
					Protocol:   "TCP",
					Port:       8088,
					TargetPort: intstr.FromInt32(8088),
				},
			},
		},
	}, metav1.CreateOptions{})

	return
}

func (r *agentObj) WaitForReady() (err error) {
	for {
		if r.pod != nil && len(r.pod.Status.ContainerStatuses) > 0 && r.pod.Status.ContainerStatuses[0].Ready {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
		pods, err := r.clientSet.CoreV1().Pods(r.namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: "testkube.io/devbox=agent",
		})
		if err != nil {
			return err
		}
		if len(pods.Items) == 0 {
			return errors.New("pod not found")
		}
		r.pod = &pods.Items[0]
	}
}

func (r *agentObj) IP() string {
	if r.pod == nil {
		return ""
	}
	return r.pod.Status.PodIP
}

func (r *agentObj) ClusterAddress() string {
	if r.IP() == "" {
		return ""
	}
	return fmt.Sprintf("devbox-agent:%d", 9000)
}

func (r *agentObj) Debug() {
	PrintHeader("Agent")
	if r.ClusterAddress() != "" {
		PrintItem("Cluster Address", r.ClusterAddress(), "")
	} else {
		PrintItem("Cluster Address", "unknown", "")
	}
}
