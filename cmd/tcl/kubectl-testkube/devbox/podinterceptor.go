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
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/kballard/go-shellquote"
	errors2 "github.com/pkg/errors"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/artifacts"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

type podInterceptorObj struct {
	clientSet        *kubernetes.Clientset
	kubernetesConfig *rest.Config
	namespace        string
	pod              *corev1.Pod
	caPem            []byte
	localPort        int
	localSslPort     int
}

func NewPodInterceptor(clientSet *kubernetes.Clientset, kubernetesConfig *rest.Config, namespace string) *podInterceptorObj {
	return &podInterceptorObj{
		clientSet:        clientSet,
		namespace:        namespace,
		kubernetesConfig: kubernetesConfig,
	}
}

func (r *podInterceptorObj) Deploy(binaryPath, initImage, toolkitImage string) (err error) {
	caPem, certPem, keyPem, err := CreateCertificate(x509.Certificate{
		DNSNames: []string{
			fmt.Sprintf("devbox-interceptor.%s", r.namespace),
			fmt.Sprintf("devbox-interceptor.%s.svc", r.namespace),
		},
	})
	if err != nil {
		return err
	}
	r.caPem = caPem

	// Deploy certificate
	_, err = r.clientSet.CoreV1().Secrets(r.namespace).Create(context.Background(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "devbox-interceptor-cert",
		},
		Data: map[string][]byte{
			"ca.crt":  caPem,
			"tls.crt": certPem,
			"tls.key": keyPem,
		},
	}, metav1.CreateOptions{})

	// Deploy Pod
	r.pod, err = r.clientSet.CoreV1().Pods(r.namespace).Create(context.Background(), &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "devbox-interceptor",
			Labels: map[string]string{
				"testkube.io/devbox": "interceptor",
			},
		},
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: common.Ptr(int64(1)),
			Volumes: []corev1.Volume{
				{Name: "server", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
				{Name: "certs", VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{
					SecretName: "devbox-interceptor-cert",
				}}},
			},
			Containers: []corev1.Container{
				{
					Name:    "interceptor",
					Image:   "busybox:1.36.1-musl",
					Command: []string{"/bin/sh", "-c", fmt.Sprintf("while [ ! -f /app/server-ready ]; do sleep 1; done\n/app/server %s", shellquote.Join(initImage, toolkitImage))},
					VolumeMounts: []corev1.VolumeMount{
						{Name: "server", MountPath: "/app"},
						{Name: "certs", MountPath: "/certs"},
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/health",
								Port:   intstr.FromInt32(8443),
								Scheme: corev1.URISchemeHTTPS,
							},
						},
						PeriodSeconds: 1,
					},
				},
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return
	}

	// Create the service
	_, err = r.clientSet.CoreV1().Services(r.namespace).Create(context.Background(), &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "devbox-interceptor",
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"testkube.io/devbox": "interceptor",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "api",
					Protocol:   "TCP",
					Port:       8443,
					TargetPort: intstr.FromInt32(8443),
				},
			},
		},
	}, metav1.CreateOptions{})

	// Wait for the container to be started
	err = r.WaitForContainerStarted()
	if err != nil {
		return
	}

	// Apply the binary
	req := r.clientSet.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(r.pod.Name).
		Namespace(r.namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: "interceptor",
			Command:   []string{"tar", "-xzf", "-", "-C", "/app"},
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(r.kubernetesConfig, "POST", req.URL())
	if err != nil {
		return errors2.Wrap(err, "failed to create spdy executor")
	}

	os.WriteFile("/tmp/flag", []byte{1}, 0777)
	flagFile, err := os.Open("/tmp/flag")
	if err != nil {
		return errors2.Wrap(err, "failed to open flag file")
	}
	defer flagFile.Close()
	flagFileStat, err := flagFile.Stat()
	if err != nil {
		return
	}

	file, err := os.Open(binaryPath)
	if err != nil {
		return
	}
	defer file.Close()
	fileStat, err := file.Stat()
	if err != nil {
		return
	}

	tarStream := artifacts.NewTarStream()
	go func() {
		defer tarStream.Close()
		tarStream.Add("server", file, fileStat)
		tarStream.Add("server-ready", flagFile, flagFileStat)
	}()

	reader, writer := io.Pipe()
	var buf []byte
	var bufMu sync.Mutex
	go func() {
		bufMu.Lock()
		defer bufMu.Unlock()
		buf, _ = io.ReadAll(reader)
	}()
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  tarStream,
		Stdout: writer,
		Stderr: writer,
		Tty:    false,
	})
	if err != nil {
		writer.Close()
		bufMu.Lock()
		defer bufMu.Unlock()
		return fmt.Errorf("failed to stream: %s: %s", err.Error(), string(buf))
	}
	writer.Close()

	return
}

func (r *podInterceptorObj) WaitForContainerStarted() (err error) {
	for {
		if r.pod != nil && len(r.pod.Status.ContainerStatuses) > 0 && r.pod.Status.ContainerStatuses[0].Started != nil && *r.pod.Status.ContainerStatuses[0].Started {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
		pods, err := r.clientSet.CoreV1().Pods(r.namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: "testkube.io/devbox=interceptor",
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

func (r *podInterceptorObj) WaitForReady() (err error) {
	for {
		if r.pod != nil && len(r.pod.Status.ContainerStatuses) > 0 && r.pod.Status.ContainerStatuses[0].Ready {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
		pods, err := r.clientSet.CoreV1().Pods(r.namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: "testkube.io/devbox=interceptor",
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

func (r *podInterceptorObj) Enable() (err error) {
	_ = r.Disable()

	_, err = r.clientSet.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(context.Background(), &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("devbox-interceptor-webhook-%s", r.namespace),
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			{
				Name: "devbox.kb.io",
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Name:      "devbox-interceptor",
						Namespace: r.namespace,
						Path:      common.Ptr("/mutate"),
						Port:      common.Ptr(int32(8443)),
					},
					CABundle: r.caPem,
				},
				Rules: []admissionregistrationv1.RuleWithOperations{
					{
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"pods"},
							Scope:       common.Ptr(admissionregistrationv1.NamespacedScope),
						},
						Operations: []admissionregistrationv1.OperationType{
							admissionregistrationv1.Create,
						},
					},
				},
				NamespaceSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "kubernetes.io/metadata.name",
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{r.namespace},
						},
					},
				},
				ObjectSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      constants.ResourceIdLabelName,
							Operator: metav1.LabelSelectorOpExists,
						},
					},
				},
				SideEffects:             common.Ptr(admissionregistrationv1.SideEffectClassNone),
				AdmissionReviewVersions: []string{"v1"},
			},
		},
	}, metav1.CreateOptions{})
	return
}

func (r *podInterceptorObj) Disable() (err error) {
	return r.clientSet.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(
		context.Background(),
		fmt.Sprintf("devbox-interceptor-webhook-%s", r.namespace),
		metav1.DeleteOptions{})
}

func (r *podInterceptorObj) IP() string {
	if r.pod == nil {
		return ""
	}
	return r.pod.Status.PodIP
}
