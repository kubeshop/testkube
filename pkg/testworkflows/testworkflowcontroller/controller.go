package testworkflowcontroller

import (
	"context"
	"io"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	initconstants "github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowcontroller/watchers"
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
	// Create local context for stopping all the processes
	ctx, ctxCancel := context.WithCancel(parentCtx)

	// Optimistically, start watching all the resources
	jobWatcher := watchers.NewJobWatcher(ctx, clientSet.BatchV1().Jobs(namespace), metav1.ListOptions{
		FieldSelector: "metadata.name=" + id,
	}, 1)
	podWatcher := watchers.NewPodWatcher(ctx, clientSet.CoreV1().Pods(namespace), metav1.ListOptions{
		LabelSelector: constants.ResourceIdLabelName + "=" + id,
	}, 1)
	jobEventsWatcher := watchers.NewEventsWatcher(ctx, clientSet.CoreV1().Events(namespace), metav1.ListOptions{
		FieldSelector: "involvedObject.name=" + id,
		TypeMeta:      metav1.TypeMeta{Kind: "Job"},
	}, 10)
	podListOptionsCh := make(chan metav1.ListOptions)
	go func() {
		select {
		case <-ctx.Done():
		case p, ok := <-podWatcher.Peek(ctx):
			if ok {
				podListOptionsCh <- metav1.ListOptions{
					FieldSelector: "involvedObject.name=" + p.Name,
					TypeMeta:      metav1.TypeMeta{Kind: "Pod"},
				}
			}
		}
		close(podListOptionsCh)
	}()
	podEventsWatcher := watchers.NewAsyncEventsWatcher(ctx, clientSet.CoreV1().Events(namespace), podListOptionsCh, 10)

	// Wait for the job information
	<-jobWatcher.Started()

	// If there is no job found, check if there was any event related to that.
	// We can't do a lot about it anyway, unless finishing the cached TestWorkflowResult.
	if !jobWatcher.Exists() {
		<-jobEventsWatcher.Started()
		defer ctxCancel()

		// TODO: Consider if the Job is required at all
		// The job existed, as there are some events related to that
		if jobEventsWatcher.Count() > 0 {
			return nil, ErrJobAborted
		}

		// There are no leftovers after that job
		return nil, ErrJobTimeout
	}

	// Obtain the signature from the Job
	job := <-jobWatcher.Peek(ctx)
	sig, err := stage.GetSignatureFromJSON([]byte(job.Annotations[constants.SignatureAnnotationName]))
	if err != nil {
		ctxCancel()
		return nil, errors.Wrap(err, "invalid job signature")
	}
	// Wait for the other lists to be started
	<-jobEventsWatcher.Started()
	<-podWatcher.Started()
	select {
	case _, ok := <-podWatcher.Peek(ctx):
		if ok {
			<-podEventsWatcher.Started()
		}
	default:
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
		job:         jobWatcher,
		jobEvents:   jobEventsWatcher,
		pod:         podWatcher,
		podEvents:   podEventsWatcher,
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
	job         watchers.JobWatcher
	jobEvents   watchers.EventsWatcher
	pod         watchers.PodWatcher
	podEvents   watchers.EventsWatcher
}

func (c *controller) Abort(ctx context.Context) error {
	return c.Cleanup(ctx)
}

func (c *controller) Cleanup(ctx context.Context) error {
	return Cleanup(ctx, c.clientSet, c.namespace, c.id)
}

func (c *controller) peekPod(ctx context.Context) (*corev1.Pod, error) {
	v, ok := <-c.pod.Peek(ctx)
	if !ok {
		return v, c.pod.Err()
	}
	if v == nil {
		return nil, errors.New("empty pod information")
	}
	return v, nil
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
	ch, err := WatchInstrumentedPod(parentCtx, c.clientSet, c.signature, c.scheduledAt, c.job, c.pod, c.jobEvents, c.podEvents, WatchInstrumentedPodOptions{})
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
	prevIsFinished := false
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
			isFinished := prevIsFinished
			if v.Value.Result != nil {
				if v.Value.Result.Status != nil {
					status = *v.Value.Result.Status
				} else {
					status = testkube.QUEUED_TestWorkflowStatus
				}
				current = v.Value.Result.Current(sig)
				isFinished = v.Value.Result.IsFinished()
			}

			// TODO: the final status should always have the finishedAt too,
			//       there should be no need for checking isFinished diff
			if nodeName != prevNodeName || isFinished != prevIsFinished || podIP != prevPodIP || prevStatus != status || prevCurrent != current {
				prevNodeName = nodeName
				prevPodIP = podIP
				prevStatus = status
				prevCurrent = current
				prevIsFinished = isFinished
				ch <- LightweightNotification{NodeName: nodeName, PodIP: podIP, Status: status, Current: current, Result: v.Value.Result}
			}
		}
	}()
	return ch
}

// TODO: Avoid WatchInstrumentedPod
func (c *controller) Logs(parentCtx context.Context, follow bool) io.Reader {
	reader, writer := io.Pipe()
	go func() {
		defer writer.Close()
		ref := ""
		ch, err := WatchInstrumentedPod(parentCtx, c.clientSet, c.signature, c.scheduledAt, c.job, c.pod, c.jobEvents, c.podEvents, WatchInstrumentedPodOptions{
			DisableFollow: !follow,
		})
		if err != nil {
			return
		}
		for v := range ch {
			if v.Error == nil && v.Value.Log != "" && !v.Value.Temporary {
				if ref != v.Value.Ref && v.Value.Ref != "" {
					ref = v.Value.Ref
					_, _ = writer.Write([]byte(instructions.SprintHint(ref, initconstants.InstructionStart)))
				}
				_, _ = writer.Write([]byte(v.Value.Log))
			}
		}
	}()
	return reader
}
