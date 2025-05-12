package dockerworker

import (
	"context"
	errors2 "errors"
	"fmt"
	"io"
	"time"

	container2 "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	dockerclient "github.com/docker/docker/client"
	"github.com/pkg/errors"

	initconstants "github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	testworkflowcontroller "github.com/kubeshop/testkube/pkg/testworkflows/executionworker/controller"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/controller/watchers"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

type ControllerOptions struct {
	Signature []stage.Signature
	RunnerId  string
}

func NewController(parentCtx context.Context, client *dockerclient.Client, id string, scheduledAt time.Time, opts ...ControllerOptions) (*controller, error) {
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
	watcher := NewExecutionWatcher(ctx, client, id, signature, scheduledAt)

	// Wait for the initial data read
	<-watcher.Started()

	// Check if we have any resources that we could base on
	if !watcher.State().Available() && watcher.State().CompletionTimestamp().IsZero() {
		defer func() {
			ctxCancel()
		}()

		// There was a job or pod for this execution, so we may only assume it is aborted
		if !watcher.State().Events().FirstTimestamp().IsZero() {
			log.DefaultLogger.Errorw("connecting to aborted execution", "executionId", watcher.State().ResourceId(), "debug", watcher.State().Debug())
			return nil, testworkflowcontroller.ErrJobAborted
		}

		// We cannot find any resources related to this execution
		return nil, testworkflowcontroller.ErrJobTimeout
	}

	// Ensure it's not using the resource that is isolated for a different runner
	if watcher.State().RunnerId() != "" && watcher.State().RunnerId() != expectedRunnerId {
		ctxCancel()
		return nil, testworkflowcontroller.ErrJobDifferentRunner
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
		client:      client,
		scheduledAt: scheduledAt,
		signature:   sig,
		ctx:         ctx,
		ctxCancel:   ctxCancel,
		watcher:     watcher,
	}, nil
}

type controller struct {
	id          string
	scheduledAt time.Time
	signature   []stage.Signature
	client      *dockerclient.Client
	ctx         context.Context
	ctxCancel   context.CancelFunc
	watcher     watchers.ExecutionWatcher
}

func (c *controller) Signature() []stage.Signature {
	return c.signature
}

func (c *controller) HasPod() bool {
	return true
}

func (c *controller) ResourceID() string {
	return c.id
}

func (c *controller) Namespace() string {
	return ""
}

func (c *controller) Abort(ctx context.Context) error {
	return c.Cleanup(ctx)
}

func (c *controller) Cleanup(ctx context.Context) error {
	return Cleanup(ctx, c.client, c.id)
}

func (c *controller) PodIP() (string, error) {
	// TODO?
	return "", nil
}

func (c *controller) InternalConfig() (testworkflowconfig.InternalConfig, error) {
	return c.watcher.State().InternalConfig()
}

func (c *controller) NodeName() (string, error) {
	return "", nil
}

func (c *controller) ContainersReady() (bool, error) {
	_, err := c.PodIP()
	if err != nil {
		return false, err
	}
	return c.watcher.State().ContainersReady(), nil
}

func (c *controller) Pause(ctx context.Context) error {
	return errors.New("not implemented")
}

func (c *controller) Resume(ctx context.Context) error {
	return errors.New("not implemented")
}

func (c *controller) StopController() {
	c.ctxCancel()
}

func (c *controller) EstimatedResult(parentCtx context.Context) (*testkube.TestWorkflowResult, error) {
	notifier := testworkflowcontroller.NewNotifier(parentCtx, testkube.TestWorkflowResult{}, c.scheduledAt)
	go notifier.Align(c.watcher.State())
	message := <-notifier.Channel()
	if message.Error != nil {
		return nil, message.Error
	}
	if message.Value.Result != nil {
		return message.Value.Result, nil
	}
	return nil, testworkflowcontroller.ErrMissingEstimatedResult
}

func (c *controller) watch(parentCtx context.Context, disableFollow bool, logAbortedDetails bool) (<-chan testworkflowcontroller.ChannelMessage[testworkflowcontroller.Notification], error) {
	return testworkflowcontroller.WatchInstrumented(parentCtx, c.signature, c.scheduledAt, c.watcher, testworkflowcontroller.WatchInstrumentedPodOptions{
		DisableFollow:     disableFollow,
		LogAbortedDetails: logAbortedDetails,
	}, func(ctx context.Context, container string, isDone func() bool, isLastHint func(instruction *instructions.Instruction) bool) <-chan testworkflowcontroller.ChannelMessage[testworkflowcontroller.ContainerLog] {
		return testworkflowcontroller.WatchContainerLogsBare(ctx, 10, 8, isLastHint, func(ctx context.Context, since *time.Time) (io.Reader, error) {
			opts := container2.LogsOptions{
				ShowStdout: true,
				ShowStderr: true,
				Timestamps: true,
				Follow:     true,
				Details:    false,
			}
			if since != nil {
				opts.Since = since.Format(time.RFC3339Nano)
			}
			return c.client.ContainerLogs(ctx, fmt.Sprintf("/%s-%s", c.ResourceID(), container), opts)
		})
	})
}

func (c *controller) Watch(parentCtx context.Context, disableFollow bool, logAbortedDetails bool) <-chan testworkflowcontroller.ChannelMessage[testworkflowcontroller.Notification] {
	ch, err := c.watch(parentCtx, disableFollow, logAbortedDetails)
	if err != nil {
		v := make(chan testworkflowcontroller.ChannelMessage[testworkflowcontroller.Notification], 1)
		v <- testworkflowcontroller.ChannelMessage[testworkflowcontroller.Notification]{Error: err}
		close(v)
		return v
	}
	return ch
}

func (c *controller) Logs(parentCtx context.Context, follow bool) io.Reader {
	reader, writer := io.Pipe()
	go func() {
		defer writer.Close()
		ref := ""
		ch, err := c.watch(parentCtx, !follow, false)
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

func Cleanup(ctx context.Context, client *dockerclient.Client, id string) error {
	_, err := client.VolumesPrune(ctx, filters.NewArgs(filters.KeyValuePair{Key: "label", Value: fmt.Sprintf("%s=%s", constants.RootResourceIdLabelName, id)}))
	_, err2 := client.ContainersPrune(ctx, filters.NewArgs(filters.KeyValuePair{Key: "label", Value: fmt.Sprintf("%s=%s", constants.ResourceIdLabelName, id)}))
	_, err3 := client.ContainersPrune(ctx, filters.NewArgs(filters.KeyValuePair{Key: "label", Value: fmt.Sprintf("%s=%s", constants.RootResourceIdLabelName, id)}))
	return errors2.Join(err, err2, err3)
}
