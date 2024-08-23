package testworkflowcontroller

import (
	"context"
	"io"
	"time"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	initconstants "github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowcontroller/watchers"
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
	Signature []stage.Signature
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
	NodeName() (string, error)
	PodIP() (string, error)
	StopController()
}

func New(parentCtx context.Context, clientSet kubernetes.Interface, namespace, id string, scheduledAt time.Time, opts ...ControllerOptions) (Controller, error) {
	var signature []stage.Signature
	for _, opt := range opts {
		if opt.Signature != nil {
			signature = opt.Signature
		}
	}

	// Create local context for stopping all the processes
	ctx, ctxCancel := context.WithCancel(parentCtx)

	// Build the execution watcher
	watcher := watchers.NewExecutionWatcher(ctx, clientSet, namespace, id, signature)

	// Wait for the initial data read
	<-watcher.Started()

	// Check if we have any resources that we could base on
	if !watcher.JobExists() && !watcher.PodExists() && !watcher.PodFinished() {
		defer ctxCancel()

		// There was a job or pod for this execution, so we may only assume it is aborted
		if watcher.JobExists() || watcher.PodExists() || watcher.PodFinished() || watcher.JobFinished() {
			return nil, ErrJobAborted
		}

		// We cannot find any resources related to this execution
		return nil, ErrJobTimeout
	}

	// Obtain the signature
	sig, err := watcher.Signature()
	if err != nil {
		ctxCancel()
		return nil, errors.Wrap(err, "invalid job signature")
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

func (c *controller) Abort(ctx context.Context) error {
	return c.Cleanup(ctx)
}

func (c *controller) Cleanup(ctx context.Context) error {
	return Cleanup(ctx, c.clientSet, c.namespace, c.id)
}

func (c *controller) PodIP() (string, error) {
	podIP := c.watcher.PodIP()
	if podIP == "" {
		if c.watcher.PodErr() != nil {
			return "", c.watcher.PodErr()
		}
		return "", ErrNoIPAssigned
	}
	return podIP, nil
}

func (c *controller) NodeName() (string, error) {
	nodeName := c.watcher.PodNodeName()
	if nodeName == "" {
		if c.watcher.PodErr() != nil {
			return "", c.watcher.PodErr()
		}
		return "", ErrNoNodeAssigned
	}
	return nodeName, nil
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

func (c *controller) Watch(parentCtx context.Context) <-chan ChannelMessage[Notification] {
	ch, err := WatchInstrumentedPod(parentCtx, c.clientSet, c.signature, c.scheduledAt, c.watcher, WatchInstrumentedPodOptions{})
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

			nodeName, _ := c.NodeName()
			podIP, _ := c.PodIP()
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
		ch, err := WatchInstrumentedPod(parentCtx, c.clientSet, c.signature, c.scheduledAt, c.watcher, WatchInstrumentedPodOptions{
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
