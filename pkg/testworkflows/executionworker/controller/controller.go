package controller

import (
	"context"
	"io"
	"time"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	initconstants "github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/controller/watchers"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

var (
	ErrJobAborted             = errors.New("job was aborted")
	ErrJobTimeout             = errors.New("timeout retrieving job")
	ErrJobDifferentRunner     = errors.New("job is assigned to a different runner")
	ErrNoIPAssigned           = errors.New("there is no IP assigned to this pod")
	ErrNoNodeAssigned         = errors.New("the pod is not assigned to a node yet")
	ErrMissingEstimatedResult = errors.New("could not estimate the result")
)

type ControllerOptions struct {
	Signature []stage.Signature
	RunnerId  string
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
	Watch(ctx context.Context, disableFollow, logAbortedDetails bool) <-chan ChannelMessage[Notification]
	WatchLightweight(ctx context.Context) <-chan LightweightNotification
	Logs(ctx context.Context, follow bool) io.Reader
	NodeName() (string, error)
	PodIP() (string, error)
	ContainersReady() (bool, error)
	InternalConfig() (testworkflowconfig.InternalConfig, error)
	EstimatedResult(parentCtx context.Context) (*testkube.TestWorkflowResult, error)
	Signature() []stage.Signature
	HasPod() bool
	ResourceID() string
	Namespace() string
	StopController()
}

func New(parentCtx context.Context, clientSet kubernetes.Interface, namespace, id string, scheduledAt time.Time, opts ...ControllerOptions) (Controller, error) {
	var signature []stage.Signature
	var expectedRunnerId string
	for _, opt := range opts {
		if opt.Signature != nil {
			signature = opt.Signature
		}
		if opt.RunnerId != "" {
			expectedRunnerId = opt.RunnerId
		}
	}

	// Create local context for stopping all the processes
	ctx, ctxCancel := context.WithCancel(parentCtx)

	// Build the execution watcher
	watcher := watchers.NewExecutionWatcher(ctx, clientSet, namespace, id, signature, scheduledAt)

	// Wait for the initial data read
	<-watcher.Started()

	// Check if we have any resources that we could base on
	if watcher.State().Job() == nil && watcher.State().Pod() == nil && watcher.State().CompletionTimestamp().IsZero() {
		defer func() {
			ctxCancel()
		}()

		// There was a job or pod for this execution, so we may only assume it is aborted
		if !watcher.State().JobEvents().FirstTimestamp().IsZero() || !watcher.State().PodEvents().FirstTimestamp().IsZero() {
			log.DefaultLogger.Errorw("connecting to aborted execution", "executionId", watcher.State().ResourceId(), "debug", watcher.State().Debug())
			return nil, ErrJobAborted
		}

		// We cannot find any resources related to this execution
		return nil, ErrJobTimeout
	}

	// Ensure it's not using the resource that is isolated for a different runner
	if watcher.State().RunnerId() != "" && watcher.State().RunnerId() != expectedRunnerId {
		ctxCancel()
		return nil, ErrJobDifferentRunner
	}

	// Obtain the signature
	sig, err := watcher.State().Signature()
	if err != nil {
		ctxCancel()
		return nil, errors.Wrap(err, "invalid job signature")
	}

	// Obtain the scheduled at timestamp
	scheduledAt = watcher.State().ScheduledAt()

	// Build accessible controller
	return &controller{
		id:          id,
		namespace:   namespace,
		scheduledAt: scheduledAt,
		signature:   sig,
		clientSet:   clientSet,
		ctx:         ctx,
		ctxCancel:   ctxCancel,
		watcher:     watcher,
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
	watcher     watchers.ExecutionWatcher
}

func (c *controller) Signature() []stage.Signature {
	return c.signature
}

func (c *controller) HasPod() bool {
	return c.watcher.State().Pod() != nil
}

func (c *controller) ResourceID() string {
	return c.id
}

func (c *controller) Namespace() string {
	return c.namespace
}

func (c *controller) Abort(ctx context.Context) error {
	return c.Cleanup(ctx)
}

func (c *controller) Cleanup(ctx context.Context) error {
	return Cleanup(ctx, c.clientSet, c.namespace, c.id)
}

func (c *controller) PodIP() (string, error) {
	podIP := c.watcher.State().PodIP()
	if podIP == "" {
		if c.watcher.PodErr() != nil {
			return "", c.watcher.PodErr()
		}
		return "", ErrNoIPAssigned
	}
	return podIP, nil
}

func (c *controller) InternalConfig() (testworkflowconfig.InternalConfig, error) {
	return c.watcher.State().InternalConfig()
}

func (c *controller) NodeName() (string, error) {
	nodeName := c.watcher.State().PodNodeName()
	if nodeName == "" {
		if c.watcher.PodErr() != nil {
			return "", c.watcher.PodErr()
		}
		return "", ErrNoNodeAssigned
	}
	return nodeName, nil
}

func (c *controller) ContainersReady() (bool, error) {
	_, err := c.PodIP()
	if err != nil {
		return false, err
	}
	return c.watcher.State().ContainersReady(), nil
}

func (c *controller) Pause(ctx context.Context) error {
	podIP, err := c.PodIP()
	if err != nil {
		return err
	}
	return Pause(ctx, podIP)
}

func (c *controller) Resume(ctx context.Context) error {
	podIP, err := c.PodIP()
	if err != nil {
		return err
	}
	return Resume(ctx, podIP)
}

func (c *controller) StopController() {
	c.ctxCancel()
}

func (c *controller) EstimatedResult(parentCtx context.Context) (*testkube.TestWorkflowResult, error) {
	notifier := newNotifier(parentCtx, testkube.TestWorkflowResult{}, c.scheduledAt)
	go notifier.Align(c.watcher.State())
	message := <-notifier.ch
	if message.Error != nil {
		return nil, message.Error
	}
	if message.Value.Result != nil {
		return message.Value.Result, nil
	}
	return nil, ErrMissingEstimatedResult
}

func (c *controller) Watch(parentCtx context.Context, disableFollow bool, logAbortedDetails bool) <-chan ChannelMessage[Notification] {
	ch, err := WatchInstrumentedPod(parentCtx, c.clientSet, c.signature, c.scheduledAt, c.watcher, WatchInstrumentedPodOptions{
		DisableFollow:     disableFollow,
		LogAbortedDetails: logAbortedDetails,
	})
	if err != nil {
		v := make(chan ChannelMessage[Notification], 1)
		v <- ChannelMessage[Notification]{Error: err}
		close(v)
		return v
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
		for v := range c.Watch(parentCtx, false, false) {
			if v.Error != nil {
				ch <- LightweightNotification{Error: v.Error}
				continue
			}

			nodeName, _ := c.NodeName()
			podIP, _ := c.PodIP()
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
		ch, err := WatchInstrumentedPod(parentCtx, c.clientSet, c.signature, c.scheduledAt, c.watcher, WatchInstrumentedPodOptions{
			DisableFollow: !follow,
		})
		if err != nil {
			return
		}
		for v := range ch {
			if v.Error == nil && v.Value.Log != "" {
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
