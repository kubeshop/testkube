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
	"sync"
	"time"

	errors2 "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/kubeshop/testkube/internal/common"
)

var (
	ErrPodNotFound = errors2.New("pod not found")
)

type PodObject struct {
	name       string
	kind       string
	namespace  string
	pod        *corev1.Pod
	service    *corev1.Service
	clientSet  *kubernetes.Clientset
	restConfig *rest.Config

	mu *sync.Mutex
}

func NewPod(kubeClient *kubernetes.Clientset, kubeRestConfig *rest.Config, namespace, name string) *PodObject {
	return &PodObject{
		name:       name,
		kind:       name,
		namespace:  namespace,
		clientSet:  kubeClient,
		restConfig: kubeRestConfig,
		mu:         &sync.Mutex{},
	}
}

func (p *PodObject) SetKind(kind string) *PodObject {
	if kind == "" {
		p.kind = p.name
	} else {
		p.kind = kind
	}
	return p
}

func (p *PodObject) Name() string {
	return p.name
}

func (p *PodObject) Namespace() string {
	return p.namespace
}

func (p *PodObject) Selector() metav1.LabelSelector {
	return metav1.LabelSelector{
		MatchLabels: map[string]string{
			"testkube.io/devbox": p.name,
		},
	}
}

func (p *PodObject) ClientSet() *kubernetes.Clientset {
	return p.clientSet
}

func (p *PodObject) RESTConfig() *rest.Config {
	return p.restConfig
}

func (p *PodObject) Create(ctx context.Context, request *corev1.Pod) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.pod != nil {
		return nil
	}
	return p.create(ctx, request)
}

func (p *PodObject) create(ctx context.Context, request *corev1.Pod) error {
	request = request.DeepCopy()
	request.Name = p.name
	request.Namespace = p.namespace
	request.ResourceVersion = ""
	if len(request.Labels) == 0 {
		request.Labels = make(map[string]string)
	}
	request.Labels["testkube.io/devbox"] = p.name
	request.Labels["testkube.io/devbox-type"] = p.kind

	pod, err := p.clientSet.CoreV1().Pods(p.namespace).Create(ctx, request, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		err = p.clientSet.CoreV1().Pods(p.namespace).Delete(ctx, request.Name, metav1.DeleteOptions{
			GracePeriodSeconds: common.Ptr(int64(0)),
			PropagationPolicy:  common.Ptr(metav1.DeletePropagationForeground),
		})
		if err != nil && !errors.IsNotFound(err) {
			return errors2.Wrap(err, "failed to delete existing pod")
		}
		pod, err = p.clientSet.CoreV1().Pods(p.namespace).Create(context.Background(), request, metav1.CreateOptions{})
	}
	if err != nil {
		return errors2.Wrap(err, "failed to create pod")
	}
	p.pod = pod
	return nil
}

func (p *PodObject) Pod() *corev1.Pod {
	return p.pod
}

func (p *PodObject) RefreshData(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	pods, err := p.clientSet.CoreV1().Pods(p.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("testkube.io/devbox=%s", p.name),
	})
	if err != nil {
		return err
	}
	if len(pods.Items) == 0 {
		p.pod = nil
		return ErrPodNotFound
	}
	p.pod = &pods.Items[0]
	return nil
}

func (p *PodObject) Watch(ctx context.Context) error {
	panic("not implemented")
}

func (p *PodObject) Restart(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	pod := p.pod
	if pod == nil {
		return ErrPodNotFound
	}
	p.pod = nil
	_ = p.clientSet.CoreV1().Pods(p.namespace).Delete(context.Background(), p.name, metav1.DeleteOptions{
		GracePeriodSeconds: common.Ptr(int64(0)),
		PropagationPolicy:  common.Ptr(metav1.DeletePropagationForeground),
	})
	return p.create(context.Background(), pod)
}

func (p *PodObject) WaitForReady(ctx context.Context) error {
	for {
		if p.pod != nil && len(p.pod.Status.ContainerStatuses) > 0 && p.pod.Status.ContainerStatuses[0].Ready {
			return nil
		}
		time.Sleep(300 * time.Millisecond)
		err := p.RefreshData(ctx)
		if err != nil {
			return err
		}
	}
}

func (p *PodObject) WaitForContainerStarted(ctx context.Context) (err error) {
	for {
		if p.pod != nil && len(p.pod.Status.ContainerStatuses) > 0 && p.pod.Status.ContainerStatuses[0].Started != nil && *p.pod.Status.ContainerStatuses[0].Started {
			return nil
		}
		time.Sleep(300 * time.Millisecond)
		err := p.RefreshData(ctx)
		if err != nil {
			return err
		}
	}
}

func (p *PodObject) ClusterIP() string {
	if p.pod == nil {
		return ""
	}
	return p.pod.Status.PodIP
}

func (p *PodObject) ClusterAddress() string {
	if p.service == nil {
		return p.ClusterIP()
	}
	return fmt.Sprintf("%s.%s.svc", p.service.Name, p.service.Namespace)
}

func (p *PodObject) CreateNamedService(ctx context.Context, name string, ports ...corev1.ServicePort) error {
	request := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"testkube.io/devbox": p.name,
			},
			Ports: ports,
		},
	}

	svc, err := p.clientSet.CoreV1().Services(p.namespace).Create(ctx, request, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		err = p.clientSet.CoreV1().Services(p.namespace).Delete(ctx, request.Name, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return errors2.Wrap(err, "failed to delete existing service")
		}
		svc, err = p.clientSet.CoreV1().Services(p.namespace).Create(ctx, request, metav1.CreateOptions{})
	}
	if err != nil {
		return err
	}
	p.service = svc
	return nil
}

func (p *PodObject) CreateService(ctx context.Context, ports ...corev1.ServicePort) error {
	return p.CreateNamedService(ctx, p.name, ports...)
}

func (p *PodObject) Forward(_ context.Context, clusterPort, localPort int, ping bool) error {
	if p.pod == nil {
		return ErrPodNotFound
	}
	return ForwardPod(p.restConfig, p.pod.Namespace, p.pod.Name, clusterPort, localPort, ping)
}
