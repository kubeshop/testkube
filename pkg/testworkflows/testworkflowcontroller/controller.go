package testworkflowcontroller

import (
	"context"
	"io"
	"time"

	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	initconstants "github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

const (
	DefaultInitTimeout = 5 * time.Second
)

var (
	ErrJobAborted     = errors.New("job was aborted")
	ErrJobTimeout     = errors.New("timeout retrieving job")
	ErrNoIPAssigned   = errors.New("there is no IP assigned to this pod")
	ErrNoNodeAssigned = errors.New("the pod is not assigned to a node yet")
)

type ControllerOptions struct {
	Timeout time.Duration
}

type LightweightNotification struct {
	Error    error
	NodeName string
	PodIP    string
	Current  string
	Status   testkube.TestWorkflowStatus
	Result   *testkube.TestWorkflowResult
}

type Controller interface {
	Abort(ctx context.Context) error
	Pause(ctx context.Context) error
	Resume(ctx context.Context) error
	Cleanup(ctx context.Context) error
	Watch(ctx context.Context) <-chan ChannelMessage[Notification]
	WatchLightweight(ctx context.Context) <-chan LightweightNotification
	Logs(ctx context.Context, follow bool) io.Reader
	NodeName(ctx context.Context) (string, error)
	PodIP(ctx context.Context) (string, error)
	StopController()
}

func New(parentCtx context.Context, clientSet kubernetes.Interface, namespace, id string, scheduledAt time.Time, opts ...ControllerOptions) (Controller, error) {
	// Get the initialization timeout
	timeout := DefaultInitTimeout
	for _, opt := range opts {
		if opt.Timeout != 0 {
			timeout = opt.Timeout
		}
	}

	// Create local context for stopping all the processes
	ctx, ctxCancel := context.WithCancel(parentCtx)

	// Optimistically, start watching all the resources
	job := WatchJob(ctx, clientSet, namespace, id, 0)
	pod := WatchMainPod(ctx, clientSet, namespace, id, 0)
	jobEvents := WatchJobEvents(ctx, clientSet, namespace, id, 0)
	podEvents := WatchPodEventsByPodWatcher(ctx, clientSet, namespace, pod, 0)

	// Ensure the main Job exists in the cluster,
	// and obtain the signature
	var sig []stage.Signature
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
		sig, err = stage.GetSignatureFromJSON([]byte(j.Value.Annotations[constants.SignatureAnnotationName]))
		if err != nil {
			ctxCancel()
			return nil, errors.Wrap(err, "invalid job signature")
		}
	case <-time.After(timeout):
		select {
		case ev, ok := <-jobEvents.PeekMessage(ctx):
			if !ok {
				err = context.Canceled
			} else if ev.Value != nil {
				// Job was there, so it was aborted
				err = ErrJobAborted
			} else {
				// There was an internal error while loading the job event
				err = ev.Error
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
	signature   []stage.Signature
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

func (c *controller) peekPod(ctx context.Context) (*corev1.Pod, error) {
	v, ok := <-c.pod.PeekMessage(ctx)
	if v.Error != nil {
		return nil, v.Error
	}
	if !ok {
		return nil, context.Canceled
	}
	if v.Value == nil {
		return nil, errors.New("empty pod information")
	}
	return v.Value, nil
}

func (c *controller) PodIP(ctx context.Context) (string, error) {
	pod, err := c.peekPod(ctx)
	if err != nil {
		return "", err
	}
	if pod.Status.PodIP == "" {
		return "", ErrNoIPAssigned
	}
	return pod.Status.PodIP, nil
}

func (c *controller) NodeName(ctx context.Context) (string, error) {
	pod, err := c.peekPod(ctx)
	if err != nil {
		return "", err
	}
	nodeName := pod.Status.NominatedNodeName
	if nodeName == "" {
		nodeName = pod.Spec.NodeName
	}
	if nodeName == "" {
		return "", ErrNoNodeAssigned
	}
	return nodeName, nil
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
	ch, err := WatchInstrumentedPod(parentCtx, c.clientSet, c.signature, c.scheduledAt, c.pod, c.podEvents, WatchInstrumentedPodOptions{
		JobEvents: c.jobEvents,
		Job:       c.job,
	})
	if err != nil {
		v := newChannel[Notification](context.Background(), 1)
		v.Error(err)
		v.Close()
		return v.Channel()
	}
	return ch
}

// TODO: Make it actually light
func (c *controller) WatchLightweight(parentCtx context.Context) <-chan LightweightNotification {
	prevCurrent := ""
	prevNodeName := ""
	prevPodIP := ""
	prevStatus := testkube.QUEUED_TestWorkflowStatus
	sig := stage.MapSignatureListToInternal(c.signature)
	ch := make(chan LightweightNotification)
	go func() {
		defer close(ch)
		for v := range c.Watch(parentCtx) {
			if v.Error != nil {
				ch <- LightweightNotification{Error: v.Error}
				continue
			}

			nodeName, _ := c.NodeName(parentCtx)
			podIP, _ := c.PodIP(parentCtx)
			current := prevCurrent
			status := prevStatus
			if v.Value.Result != nil {
				if v.Value.Result.Status != nil {
					status = *v.Value.Result.Status
				} else {
					status = testkube.QUEUED_TestWorkflowStatus
				}
				current = v.Value.Result.Current(sig)
			}

			if nodeName != prevNodeName || podIP != prevPodIP || prevStatus != status || prevCurrent != current {
				prevNodeName = nodeName
				prevPodIP = podIP
				prevStatus = status
				prevCurrent = current
				ch <- LightweightNotification{NodeName: nodeName, PodIP: podIP, Status: status, Current: current, Result: v.Value.Result}
			}
		}
	}()
	return ch
}

func (c *controller) Logs(parentCtx context.Context, follow bool) io.Reader {
	reader, writer := io.Pipe()
	go func() {
		defer writer.Close()
		ref := ""
		// Wait until there will be events fetched first
		alignTimeoutCh := time.After(alignmentTimeout)
		select {
		case <-c.jobEvents.Peek(parentCtx):
		case <-alignTimeoutCh:
		}
		select {
		case <-c.podEvents.Peek(parentCtx):
		case <-alignTimeoutCh:
		}
		ch, err := WatchInstrumentedPod(parentCtx, c.clientSet, c.signature, c.scheduledAt, c.pod, c.podEvents, WatchInstrumentedPodOptions{
			JobEvents: c.jobEvents,
			Job:       c.job,
			Follow:    common.Ptr(follow),
		})
		if err != nil {
			return
		}
		for v := range ch {
			if v.Error == nil && v.Value.Log != "" && !v.Value.Temporary {
				if ref != v.Value.Ref && v.Value.Ref != "" {
					ref = v.Value.Ref
					_, _ = writer.Write([]byte(data.SprintHint(ref, initconstants.InstructionStart)))
				}
				_, _ = writer.Write([]byte(v.Value.Log))
			}
		}
	}()
	return reader
}
