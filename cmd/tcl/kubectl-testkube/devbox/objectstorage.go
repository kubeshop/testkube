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
	"k8s.io/client-go/rest"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/storage/minio"
)

type objectStorageObj struct {
	clientSet        kubernetes.Interface
	kubernetesConfig *rest.Config
	namespace        string
	pod              *corev1.Pod
	localPort        int
	localWebPort     int
}

func NewObjectStorage(clientSet kubernetes.Interface, kubernetesConfig *rest.Config, namespace string) *objectStorageObj {
	return &objectStorageObj{
		clientSet:        clientSet,
		namespace:        namespace,
		kubernetesConfig: kubernetesConfig,
	}
}

func (r *objectStorageObj) Deploy() (err error) {
	r.pod, err = r.clientSet.CoreV1().Pods(r.namespace).Create(context.Background(), &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "devbox-storage",
			Labels: map[string]string{
				"testkube.io/devbox": "storage",
			},
		},
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: common.Ptr(int64(1)),
			Containers: []corev1.Container{
				{
					Name:  "minio",
					Image: "minio/minio:RELEASE.2024-10-13T13-34-11Z",
					Args:  []string{"server", "/data", "--console-address", ":9090"},
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
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Create the service
	_, err = r.clientSet.CoreV1().Services(r.namespace).Create(context.Background(), &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "devbox-storage",
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"testkube.io/devbox": "storage",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "api",
					Protocol:   "TCP",
					Port:       9000,
					TargetPort: intstr.FromInt32(9000),
				},
			},
		},
	}, metav1.CreateOptions{})

	return
}

func (r *objectStorageObj) WaitForReady() (err error) {
	for {
		if r.pod != nil && len(r.pod.Status.ContainerStatuses) > 0 && r.pod.Status.ContainerStatuses[0].Ready {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
		pods, err := r.clientSet.CoreV1().Pods(r.namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: "testkube.io/devbox=storage",
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

func (r *objectStorageObj) IP() string {
	if r.pod == nil {
		return ""
	}
	return r.pod.Status.PodIP
}

func (r *objectStorageObj) ClusterAddress() string {
	ip := r.IP()
	if ip == "" {
		return ""
	}
	return fmt.Sprintf("%s:%d", ip, 9000)
}

func (r *objectStorageObj) LocalAddress() string {
	if r.localPort == 0 {
		return ""
	}
	return fmt.Sprintf("0.0.0.0:%d", r.localPort)
}

func (r *objectStorageObj) LocalWebAddress() string {
	if r.localWebPort == 0 {
		return ""
	}
	return fmt.Sprintf("127.0.0.1:%d", r.localWebPort)
}

func (r *objectStorageObj) Forward() error {
	if r.pod == nil {
		return errors.New("pod not found")
	}
	if r.localPort != 0 {
		return nil
	}
	port, err := GetFreePort()
	if r.localWebPort != 0 {
		return nil
	}
	webPort, err := GetFreePort()
	if err != nil {
		return err
	}
	err = ForwardPodPort(r.kubernetesConfig, r.pod.Namespace, r.pod.Name, 9000, port)
	if err != nil {
		return err
	}
	r.localPort = port
	err = ForwardPodPort(r.kubernetesConfig, r.pod.Namespace, r.pod.Name, 9090, webPort)
	if err != nil {
		return err
	}
	r.localWebPort = webPort
	return nil
}

func (r *objectStorageObj) Connect() (*minio.Client, error) {
	minioClient := minio.NewClient(
		r.LocalAddress(),
		"minioadmin",
		"minioadmin",
		"",
		"",
		"devbox",
	)
	err := minioClient.Connect()
	return minioClient, err
}

func (r *objectStorageObj) Debug() {
	PrintHeader("Object Storage")
	if r.ClusterAddress() != "" {
		PrintItem("Cluster Address", r.ClusterAddress(), "")
	} else {
		PrintItem("Cluster Address", "unknown", "")
	}
	if r.LocalAddress() != "" {
		PrintItem("Local Address", r.LocalAddress(), "")
	} else {
		PrintItem("Local Address", "not forwarded", "")
	}
	if r.LocalWebAddress() != "" {
		PrintItem("Console", "http://"+r.LocalWebAddress(), "minioadmin / minioadmin")
	} else {
		PrintItem("Console", "not forwarded", "")
	}
}
