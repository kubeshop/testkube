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
	"crypto/x509"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/kballard/go-shellquote"
	errors2 "github.com/pkg/errors"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/artifacts"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

type Interceptor struct {
	pod                  *PodObject
	caPem                []byte
	initProcessImageName string
	toolkitImageName     string
	binary               *Binary
}

func NewInterceptor(pod *PodObject, initProcessImageName, toolkitImageName string, binary *Binary) *Interceptor {
	return &Interceptor{
		pod:                  pod,
		initProcessImageName: initProcessImageName,
		toolkitImageName:     toolkitImageName,
		binary:               binary,
	}
}

func (r *Interceptor) Create(ctx context.Context) error {
	if r.binary.Hash() == "" {
		return errors2.New("interceptor binary is not built")
	}

	certSet, err := CreateCertificate(x509.Certificate{
		DNSNames: []string{
			fmt.Sprintf("%s.%s", r.pod.Name(), r.pod.Namespace()),
			fmt.Sprintf("%s.%s.svc", r.pod.Name(), r.pod.Namespace()),
		},
	})
	if err != nil {
		return err
	}

	// Deploy certificate
	certSecretName := fmt.Sprintf("%s-cert", r.pod.Name())
	_, err = r.pod.ClientSet().CoreV1().Secrets(r.pod.Namespace()).Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: certSecretName,
		},
		Data: map[string][]byte{
			"ca.crt":  certSet.CaPEM,
			"tls.crt": certSet.CrtPEM,
			"tls.key": certSet.KeyPEM,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
		secret, err := r.pod.ClientSet().CoreV1().Secrets(r.pod.Namespace()).Get(ctx, certSecretName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		certSet.CaPEM = secret.Data["ca.crt"]
		certSet.CrtPEM = secret.Data["tls.crt"]
		certSet.KeyPEM = secret.Data["tls.key"]
	}
	r.caPem = certSet.CaPEM

	// Deploy Pod
	err = r.pod.Create(ctx, &corev1.Pod{
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: common.Ptr(int64(1)),
			Volumes: []corev1.Volume{
				{Name: "server", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
				{Name: "certs", VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{
					SecretName: certSecretName,
				}}},
			},
			Containers: []corev1.Container{
				{
					Name:            "interceptor",
					Image:           "busybox:1.36.1-musl",
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{"/bin/sh", "-c", fmt.Sprintf("while [ ! -f /app/server-ready ]; do sleep 1; done\n/app/server %s", shellquote.Join(r.initProcessImageName, r.toolkitImageName))},
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
	})
	if err != nil {
		return err
	}

	// Wait for the container to be started
	err = r.pod.WaitForContainerStarted(ctx)
	if err != nil {
		return err
	}

	// Deploy Service
	err = r.pod.CreateService(ctx, corev1.ServicePort{
		Name:       "api",
		Protocol:   "TCP",
		Port:       8443,
		TargetPort: intstr.FromInt32(8443),
	})
	if err != nil {
		return err
	}

	// TODO: Move transfer utilities to *PodObject
	// Apply the binary
	req := r.pod.ClientSet().CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(r.pod.Name()).
		Namespace(r.pod.Namespace()).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: "interceptor",
			Command:   []string{"tar", "-xzf", "-", "-C", "/app"},
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(r.pod.RESTConfig(), "POST", req.URL())
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
		return err
	}

	file, err := os.Open(r.binary.Path())
	if err != nil {
		return err
	}
	defer file.Close()
	fileStat, err := file.Stat()
	if err != nil {
		return err
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
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  tarStream,
		Stdout: writer,
		Stderr: writer,
		Tty:    false,
	})
	if err != nil {
		writer.Close()
		bufMu.Lock()
		defer bufMu.Unlock()
		return fmt.Errorf("failed to stream binary: %s: %s", err.Error(), string(buf))
	}
	writer.Close()

	return nil
}

func (r *Interceptor) WaitForReady(ctx context.Context) error {
	return r.pod.WaitForReady(ctx)
}

func (r *Interceptor) Enable(ctx context.Context) error {
	_ = r.Disable()

	_, err := r.pod.ClientSet().AdmissionregistrationV1().MutatingWebhookConfigurations().Create(ctx, &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-webhook-%s", r.pod.Name(), r.pod.Namespace()),
			Labels: map[string]string{
				"testkube.io/devbox-name": r.pod.Namespace(),
			},
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			{
				Name: "devbox.kb.io",
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Name:      r.pod.Name(),
						Namespace: r.pod.Namespace(),
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
							Values:   []string{r.pod.Namespace()},
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
	return err
}

func (r *Interceptor) Disable() (err error) {
	return r.pod.ClientSet().AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(
		context.Background(),
		fmt.Sprintf("%s-webhook-%s", r.pod.Name(), r.pod.Namespace()),
		metav1.DeleteOptions{})
}
