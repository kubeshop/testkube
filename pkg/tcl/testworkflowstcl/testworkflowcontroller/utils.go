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
	"fmt"
	"reflect"
	"regexp"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor/constants"
)

const (
	KubernetesLogTimeFormat = "2006-01-02T15:04:05.000000000Z"
)

func IsPodDone(pod *corev1.Pod) bool {
	return (pod.Status.Phase != corev1.PodPending && pod.Status.Phase != corev1.PodRunning) || pod.ObjectMeta.DeletionTimestamp != nil
}

func IsJobDone(job *batchv1.Job) bool {
	return (job.Status.Active == 0 && (job.Status.Succeeded > 0 || job.Status.Failed > 0)) || job.ObjectMeta.DeletionTimestamp != nil
}

func WatchJob(ctx context.Context, clientSet kubernetes.Interface, namespace, name string, cacheSize int) Watcher[*batchv1.Job] {
	w := newWatcher[*batchv1.Job](ctx, cacheSize)

	go func() {
		defer w.Close()
		selector := "metadata.name=" + name

		// Get initial pods
		list, err := clientSet.BatchV1().Jobs(namespace).List(w.ctx, metav1.ListOptions{
			FieldSelector: selector,
		})

		// Expose the initial value
		if err != nil {
			w.SendError(err)
			return
		}
		if len(list.Items) == 1 {
			job := list.Items[0]
			w.SendValue(&job)
			if IsJobDone(&job) {
				return
			}
		}

		// Start watching for changes
		jobs, err := clientSet.BatchV1().Jobs(namespace).Watch(w.ctx, metav1.ListOptions{
			ResourceVersion: list.ResourceVersion,
			FieldSelector:   selector,
		})
		if err != nil {
			w.SendError(err)
			return
		}
		defer jobs.Stop()
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
			case event, ok := <-jobs.ResultChan():
				if !ok {
					return
				}
				switch event.Type {
				case watch.Added, watch.Modified:
					job := event.Object.(*batchv1.Job)
					if job == nil {
						continue
					}
					w.SendValue(job)
					if IsJobDone(job) {
						return
					}
				case watch.Deleted:
					return
				}
			}
		}
	}()

	return w
}

func WatchMainPod(ctx context.Context, clientSet kubernetes.Interface, namespace, name string, cacheSize int) Watcher[*corev1.Pod] {
	return watchPod(ctx, clientSet, namespace, ListOptions{
		LabelSelector: constants.ExecutionIdMainPodLabelName + "=" + name,
		CacheSize:     cacheSize,
	})
}

func WatchPodByName(ctx context.Context, clientSet kubernetes.Interface, namespace, name string, cacheSize int) Watcher[*corev1.Pod] {
	return watchPod(ctx, clientSet, namespace, ListOptions{
		FieldSelector: "metadata.name=" + name,
		CacheSize:     cacheSize,
	})
}

func watchPod(ctx context.Context, clientSet kubernetes.Interface, namespace string, options ListOptions) Watcher[*corev1.Pod] {
	w := newWatcher[*corev1.Pod](ctx, options.CacheSize)

	go func() {
		defer w.Close()

		// Get initial pods
		list, err := clientSet.CoreV1().Pods(namespace).List(w.ctx, metav1.ListOptions{
			FieldSelector: options.FieldSelector,
			LabelSelector: options.LabelSelector,
		})

		// Expose the initial value
		if err != nil {
			w.SendError(err)
			return
		}
		if len(list.Items) == 1 {
			pod := list.Items[0]
			w.SendValue(&pod)
			if IsPodDone(&pod) {
				return
			}
		}

		// Start watching for changes
		pods, err := clientSet.CoreV1().Pods(namespace).Watch(w.ctx, metav1.ListOptions{
			ResourceVersion: list.ResourceVersion,
			FieldSelector:   options.FieldSelector,
			LabelSelector:   options.LabelSelector,
		})
		if err != nil {
			w.SendError(err)
			return
		}
		defer pods.Stop()
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
			case event, ok := <-pods.ResultChan():
				if !ok {
					return
				}
				switch event.Type {
				case watch.Added, watch.Modified:
					pod := event.Object.(*corev1.Pod)
					if pod == nil {
						continue
					}
					w.SendValue(pod)
					if IsPodDone(pod) {
						return
					}
				case watch.Deleted:
					return
				}
			}
		}
	}()

	return w
}

type ListOptions struct {
	FieldSelector string
	LabelSelector string
	TypeMeta      metav1.TypeMeta
	CacheSize     int
}

func GetEventContainerName(event *corev1.Event) string {
	regex := regexp.MustCompile(`^spec\.(?:initContainers|containers)\{([^]]+)}`)
	path := event.InvolvedObject.FieldPath
	if regex.Match([]byte(path)) {
		name := regex.ReplaceAllString(event.InvolvedObject.FieldPath, "$1")
		return name
	}
	return ""
}

func WatchContainerEvents(ctx context.Context, podEvents Watcher[*corev1.Event], name string, cacheSize int, includePodWarnings bool) Watcher[*corev1.Event] {
	w := newWatcher[*corev1.Event](ctx, cacheSize)
	go func() {
		stream := podEvents.Stream(ctx)
		defer stream.Stop()
		defer w.Close()
		for {
			select {
			case <-w.Done():
				return
			case v, ok := <-stream.Channel():
				if ok {
					if v.Error != nil {
						w.SendError(v.Error)
					} else if GetEventContainerName(v.Value) == name {
						w.SendValue(v.Value)
					} else if includePodWarnings && v.Value.Type == "Warning" {
						w.SendValue(v.Value)
					}
				} else {
					return
				}
			}
		}
	}()
	return w
}

func WatchContainerStatus(ctx context.Context, pod Watcher[*corev1.Pod], containerName string, cacheSize int) Watcher[corev1.ContainerStatus] {
	w := newWatcher[corev1.ContainerStatus](ctx, cacheSize)

	go func() {
		stream := pod.Stream(ctx)
		defer stream.Stop()
		defer w.Close()
		var prev corev1.ContainerStatus
		for {
			select {
			case <-w.Done():
				return
			case p, ok := <-stream.Channel():
				if !ok {
					return
				}
				if p.Error != nil {
					w.SendError(p.Error)
					continue
				}
				if p.Value == nil {
					continue
				}
				for _, s := range append(p.Value.Status.InitContainerStatuses, p.Value.Status.ContainerStatuses...) {
					if s.Name == containerName {
						if !reflect.DeepEqual(s, prev) {
							prev = s
							w.SendValue(s)
						}
						break
					}
				}
				if IsPodDone(p.Value) {
					return
				}
			}
		}
	}()

	return w
}

func WatchPodEventsByName(ctx context.Context, clientSet kubernetes.Interface, namespace, name string, cacheSize int) Watcher[*corev1.Event] {
	return WatchEvents(ctx, clientSet, namespace, ListOptions{
		FieldSelector: "involvedObject.name=" + name,
		TypeMeta:      metav1.TypeMeta{Kind: "Pod"},
		CacheSize:     cacheSize,
	})
}

func WatchPodEventsByPodWatcher(ctx context.Context, clientSet kubernetes.Interface, namespace string, pod Watcher[*corev1.Pod], cacheSize int) Watcher[*corev1.Event] {
	w := newWatcher[*corev1.Event](ctx, cacheSize)

	go func() {
		defer w.Close()

		v, ok := <-pod.Any(ctx)
		if v.Error != nil {
			w.SendError(v.Error)
			return
		}
		if !ok || v.Value == nil {
			return
		}
		_, wch := watchEvents(clientSet, namespace, ListOptions{
			FieldSelector: "involvedObject.name=" + v.Value.Name,
			TypeMeta:      metav1.TypeMeta{Kind: "Pod"},
		}, w)

		// Wait for all immediate events
		<-wch

		// Adds missing "Started" events.
		// It may have duplicated "Started", but better than no events.
		// @see {@link https://github.com/kubernetes/kubernetes/issues/122904#issuecomment-1944387021}
		started := map[string]bool{}
		for p := range pod.Stream(ctx).Channel() {
			for i, s := range append(p.Value.Status.InitContainerStatuses, p.Value.Status.ContainerStatuses...) {
				if !started[s.Name] && (s.State.Running != nil || s.State.Terminated != nil) {
					ts := metav1.Time{Time: time.Now()}
					if s.State.Running != nil {
						ts = s.State.Running.StartedAt
					} else if s.State.Terminated != nil {
						ts = s.State.Terminated.StartedAt
					}
					started[s.Name] = true
					fieldPath := fmt.Sprintf("spec.containers{%s}", s.Name)
					if i >= len(p.Value.Status.InitContainerStatuses) {
						fieldPath = fmt.Sprintf("spec.initContainers{%s}", s.Name)
					}
					w.SendValue(&corev1.Event{
						ObjectMeta:     metav1.ObjectMeta{CreationTimestamp: ts},
						FirstTimestamp: ts,
						Type:           "Normal",
						Reason:         "Started",
						Message:        fmt.Sprintf("Started container %s", s.Name),
						InvolvedObject: corev1.ObjectReference{FieldPath: fieldPath},
					})
				}
			}
		}
	}()

	return w
}

func WatchJobEvents(ctx context.Context, clientSet kubernetes.Interface, namespace, name string, cacheSize int) Watcher[*corev1.Event] {
	return WatchEvents(ctx, clientSet, namespace, ListOptions{
		FieldSelector: "involvedObject.name=" + name,
		TypeMeta:      metav1.TypeMeta{Kind: "Job"},
		CacheSize:     cacheSize,
	})
}

func WatchJobPreEvents(ctx context.Context, jobEvents Watcher[*corev1.Event], cacheSize int) Watcher[*corev1.Event] {
	w := newWatcher[*corev1.Event](ctx, cacheSize)
	go func() {
		defer w.Close()
		stream := jobEvents.Stream(ctx)
		defer stream.Stop()

		for {
			select {
			case <-w.Done():
				return
			case v := <-stream.Channel():
				if v.Error != nil {
					w.SendError(v.Error)
				} else {
					w.SendValue(v.Value)
					if v.Value.Reason == "SuccessfulCreate" {
						return
					}
				}
			}
		}
	}()
	return w
}

func WatchEvents(ctx context.Context, clientSet kubernetes.Interface, namespace string, options ListOptions) Watcher[*corev1.Event] {
	w, _ := watchEvents(clientSet, namespace, options, newWatcher[*corev1.Event](ctx, options.CacheSize))
	return w
}

func watchEvents(clientSet kubernetes.Interface, namespace string, options ListOptions, w *watcher[*corev1.Event]) (Watcher[*corev1.Event], chan struct{}) {
	initCh := make(chan struct{})
	go func() {
		defer w.Close()

		// Get initial events
		list, err := clientSet.CoreV1().Events(namespace).List(w.ctx, metav1.ListOptions{
			FieldSelector: options.FieldSelector,
			LabelSelector: options.LabelSelector,
			TypeMeta:      options.TypeMeta,
		})

		// Expose the initial value
		if err != nil {
			w.SendError(err)
			close(initCh)
			return
		}
		for _, event := range list.Items {
			w.SendValue(event.DeepCopy())
		}
		close(initCh)

		// Start watching for changes
		events, err := clientSet.CoreV1().Events(namespace).Watch(w.ctx, metav1.ListOptions{
			ResourceVersion: list.ResourceVersion,
			FieldSelector:   options.FieldSelector,
			LabelSelector:   options.LabelSelector,
			TypeMeta:        options.TypeMeta,
		})
		if err != nil {
			w.SendError(err)
			return
		}
		defer events.Stop()
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
			case event, ok := <-events.ResultChan():
				if !ok {
					return
				}
				if event.Object == nil {
					continue
				}
				switch event.Type {
				case watch.Added, watch.Modified:
					v := event.Object.(*corev1.Event)
					if v != nil {
						w.SendValue(v)
					}
				}
			}
		}
	}()

	return w, initCh
}
