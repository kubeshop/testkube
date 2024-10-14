package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

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
		opts.TimeoutSeconds = common.Ptr(DefaultTimeoutSeconds)
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

// TODO: Delete
func WatchMainPod(ctx context.Context, clientSet kubernetes.Interface, namespace, name string, bufferSize int) Channel[*corev1.Pod] {
	return watchPod(ctx, clientSet, namespace, bufferSize, metav1.ListOptions{
		LabelSelector: constants.ResourceIdLabelName + "=" + name,
	})
}
