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
	"time"

	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor/constants"
)

const (
	DefaultInitTimeout = 1 * time.Second
)

var (
	ErrJobAborted = errors.New("job was aborted")
	ErrJobTimeout = errors.New("timeout retrieving job")
)

type Controller interface {
	Abort(ctx context.Context) error
	Pause(ctx context.Context) error
	Resume(ctx context.Context) error
	Cleanup(ctx context.Context) error
	Watch(ctx context.Context) <-chan ChannelMessage[Notification]
	StopController()
}

func New(parentCtx context.Context, clientSet kubernetes.Interface, namespace, id string, scheduledAt time.Time) (Controller, error) {
	// Get the initialization timeout
	timeout := DefaultInitTimeout

	// Create local context for stopping all the processes
	ctx, ctxCancel := context.WithCancel(parentCtx)

	// Optimistically, start watching all the resources
	job := WatchJob(ctx, clientSet, namespace, id, 0)
	pod := WatchMainPod(ctx, clientSet, namespace, id, 0)
	jobEvents := WatchJobEvents(ctx, clientSet, namespace, id, 0)
	podEvents := WatchPodEventsByPodWatcher(ctx, clientSet, namespace, pod, 0)

	// Ensure the main Job exists in the cluster,
	// and obtain the signature
	var sig []testworkflowprocessor.Signature
	var err error
	select {
	case j, ok := <-job.PeekMessage(ctx):
		if !ok {
			j.Error = context.Canceled
		} else if j.Error == nil && j.Value == nil {
			j.Error = ErrJobAborted
		}
		if j.Error != nil {
			ctxCancel()
			return nil, j.Error
		}
		sig, err = testworkflowprocessor.GetSignatureFromJSON([]byte(j.Value.Annotations[constants.SignatureAnnotationName]))
		if err != nil {
			ctxCancel()
			return nil, errors.Wrap(err, "invalid job signature")
		}
	case <-time.After(timeout):
		select {
		case ev, ok := <-jobEvents.PeekMessage(ctx):
			if !ok {
				err = context.Canceled
			}
			if ev.Value != nil {
				// Job was there, so it was aborted
				err = ErrJobAborted
			}
		case <-time.After(timeout):
			// The job is actually not found
			err = ErrJobTimeout
		}
		ctxCancel()
		return nil, err
	}

	// Build accessible controller
	return &controller{
		id:          id,
		namespace:   namespace,
		scheduledAt: scheduledAt,
		signature:   sig,
		clientSet:   clientSet,
		ctx:         ctx,
		ctxCancel:   ctxCancel,
		job:         job,
		jobEvents:   jobEvents,
		pod:         pod,
		podEvents:   podEvents,
	}, nil
}

type controller struct {
	id          string
	namespace   string
	scheduledAt time.Time
	signature   []testworkflowprocessor.Signature
	clientSet   kubernetes.Interface
	ctx         context.Context
	ctxCancel   context.CancelFunc
	job         Channel[*batchv1.Job]
	jobEvents   Channel[*corev1.Event]
	pod         Channel[*corev1.Pod]
	podEvents   Channel[*corev1.Event]
}

func (c *controller) Abort(ctx context.Context) error {
	return c.Cleanup(ctx)
}

func (c *controller) Cleanup(ctx context.Context) error {
	return Cleanup(ctx, c.clientSet, c.namespace, c.id)
}

func (c *controller) PodIP(ctx context.Context) (string, error) {
	v, ok := <-c.pod.PeekMessage(ctx)
	if v.Error != nil {
		return "", v.Error
	}
	if !ok {
		return "", context.Canceled
	}
	if v.Value.Status.PodIP == "" {
		return "", errors.New("there is no IP assigned to this pod")
	}
	return v.Value.Status.PodIP, nil
}

func (c *controller) Pause(ctx context.Context) error {
	podIP, err := c.PodIP(ctx)
	if err != nil {
		return err
	}
	return Pause(ctx, podIP)
}

func (c *controller) Resume(ctx context.Context) error {
	podIP, err := c.PodIP(ctx)
	if err != nil {
		return err
	}
	return Resume(ctx, podIP)
}

func (c *controller) StopController() {
	c.ctxCancel()
}

func (c *controller) Watch(parentCtx context.Context) <-chan ChannelMessage[Notification] {
	w, err := WatchInstrumentedPod(parentCtx, c.clientSet, c.signature, c.scheduledAt, c.pod, c.podEvents, WatchInstrumentedPodOptions{
		JobEvents: c.jobEvents,
		Job:       c.job,
	})
	if err != nil {
		v := newChannel[Notification](context.Background(), 1)
		v.Error(err)
		v.Close()
		return v.Channel()
	}
	return w.Channel()
}
