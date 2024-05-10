// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowcontroller

import (
	"context"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor/constants"
)

const (
	TimeoutSeconds = int64(365 * 24 * 3600)
)

type kubernetesClient[T any, U any] interface {
	List(ctx context.Context, options metav1.ListOptions) (*T, error)
	Watch(ctx context.Context, options metav1.ListOptions) (watch.Interface, error)
}

func readKubernetesChannel[T any](w *channel[*T], input <-chan watch.Event, stopFn func(*T) bool) {
	for {
		// Prioritize checking for done
		select {
		case <-w.Done():
			return
		default:
		}
		// Wait for results
		select {
		case <-w.Done():
			return
		case event, ok := <-input:
			if !ok {
				return
			}
			value, ok := event.Object.(any).(*T)
			if !ok || value == nil {
				continue
			}
			w.Send(value)
			if stopFn(value) {
				return
			}
		}
	}
}

// TODO: Allow rebuilding watcher after the connection is down
func watchKubernetes[T any, U any](ctx context.Context, w *channel[*U], client kubernetesClient[T, U], accessor func(*T) ([]U, string), stopFn func(*U) bool, opts metav1.ListOptions) {
	if opts.TimeoutSeconds == nil {
		opts.TimeoutSeconds = common.Ptr(TimeoutSeconds)
	}

	// Read initial data
	list, err := client.List(ctx, opts)
	if err != nil {
		w.Error(err)
		return
	}
	items, resourceVersion := accessor(list)
	list = nil
	for i := range items {
		v := items[i]
		w.Send(&v)
		if stopFn(&v) {
			return
		}
	}
	items = nil

	// Watch for changes
	opts.ResourceVersion = resourceVersion
	events, err := client.Watch(ctx, opts)
	if err != nil {
		w.Error(err)
		return
	}
	defer events.Stop()
	readKubernetesChannel(w, events.ResultChan(), stopFn)
}

func watchPod(ctx context.Context, clientSet kubernetes.Interface, namespace string, bufferSize int, options metav1.ListOptions) Channel[*corev1.Pod] {
	w := newChannel[*corev1.Pod](ctx, bufferSize)

	go func() {
		defer w.Close()

		getItems := func(list *corev1.PodList) ([]corev1.Pod, string) {
			return list.Items, list.ResourceVersion
		}
		watchKubernetes(w.ctx, w, clientSet.CoreV1().Pods(namespace), getItems, IsPodDone, options)
	}()

	return w
}

func watchEvents(clientSet kubernetes.Interface, namespace string, options metav1.ListOptions, w *channel[*corev1.Event]) Channel[*corev1.Event] {
	go func() {
		defer w.Close()

		getItems := func(t *corev1.EventList) ([]corev1.Event, string) {
			return t.Items, t.ResourceVersion
		}
		watchKubernetes(w.ctx, w, clientSet.CoreV1().Events(namespace), getItems, common.Never[*corev1.Event], options)
	}()

	return w
}

func WatchJob(ctx context.Context, clientSet kubernetes.Interface, namespace, name string, bufferSize int) Channel[*batchv1.Job] {
	w := newChannel[*batchv1.Job](ctx, bufferSize)

	go func() {
		defer w.Close()

		getItems := func(list *batchv1.JobList) ([]batchv1.Job, string) {
			return list.Items, list.ResourceVersion
		}
		watchKubernetes(w.ctx, w, clientSet.BatchV1().Jobs(namespace), getItems, IsJobDone, metav1.ListOptions{
			FieldSelector:  "metadata.name=" + name,
			TimeoutSeconds: common.Ptr(TimeoutSeconds),
		})
	}()

	return w
}

func WatchMainPod(ctx context.Context, clientSet kubernetes.Interface, namespace, name string, bufferSize int) Channel[*corev1.Pod] {
	return watchPod(ctx, clientSet, namespace, bufferSize, metav1.ListOptions{
		LabelSelector: constants.ExecutionIdMainPodLabelName + "=" + name,
	})
}

func WatchPodEventsByPodWatcher(ctx context.Context, clientSet kubernetes.Interface, namespace string, pod Peekable[*corev1.Pod], bufferSize int) Channel[*corev1.Event] {
	w := newChannel[*corev1.Event](ctx, bufferSize)

	go func() {
		defer w.Close()

		v, ok := <-pod.PeekMessage(ctx)
		if !ok {
			return
		}
		if v.Error != nil {
			w.Error(v.Error)
			return
		}

		// Combine all streams together
		watchEvents(clientSet, namespace, metav1.ListOptions{
			FieldSelector: "involvedObject.name=" + v.Value.Name,
			TypeMeta:      metav1.TypeMeta{Kind: "Pod"},
		}, w)
	}()

	return w
}

func WatchJobEvents(ctx context.Context, clientSet kubernetes.Interface, namespace, name string, bufferSize int) Channel[*corev1.Event] {
	return WatchEvents(ctx, clientSet, namespace, bufferSize, metav1.ListOptions{
		FieldSelector: "involvedObject.name=" + name,
		TypeMeta:      metav1.TypeMeta{Kind: "Job"},
	})
}

func WatchEvents(ctx context.Context, clientSet kubernetes.Interface, namespace string, bufferSize int, options metav1.ListOptions) Channel[*corev1.Event] {
	return watchEvents(clientSet, namespace, options, newChannel[*corev1.Event](ctx, bufferSize))
}
