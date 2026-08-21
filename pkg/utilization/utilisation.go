package utilization

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/utilization/core"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/output"
)

const (
	slowSamplingInterval    = 15 * time.Second
	fastSamplingInterval    = 1 * time.Second
	defaultSamplingInterval = fastSamplingInterval
	recorderShutdownTimeout = 5 * time.Second
)

type MetricRecorder struct {
	writer           core.Writer
	format           core.Formatter
	samplingInterval time.Duration
	tags             []core.KeyValue
	samples          atomic.Int64
	writeErr         atomic.Pointer[error]
	closeErr         atomic.Pointer[error]
}

type Option func(*MetricRecorder)

func WithFormatter(format core.MetricsFormat) Option {
	return func(u *MetricRecorder) {
		formatter, err := core.NewFormatter(format)
		if err != nil {
			panic(fmt.Sprintf("failed to create formatter: %v", err))
		}
		u.format = formatter
	}
}

func WithWriter(writer core.Writer) Option {
	return func(u *MetricRecorder) {
		u.writer = writer
	}
}

func WithSamplingInterval(interval time.Duration) Option {
	return func(u *MetricRecorder) {
		u.samplingInterval = interval
	}
}

func WithTags(tags []core.KeyValue) Option {
	return func(u *MetricRecorder) {
		u.tags = tags
	}
}

func NewMetricsRecorder(opts ...Option) *MetricRecorder {
	u := &MetricRecorder{
		format:           core.NewInfluxDBLineProtocolFormatter(),
		writer:           core.NewSTDOUTWriter(),
		samplingInterval: defaultSamplingInterval,
	}
	for _, opt := range opts {
		opt(u)
	}
	return u

}

func (r *MetricRecorder) Samples() int64 {
	return r.samples.Load()
}

func (r *MetricRecorder) WriteError() error {
	if err := r.writeErr.Load(); err != nil {
		return *err
	}
	return nil
}

func (r *MetricRecorder) CloseError() error {
	if err := r.closeErr.Load(); err != nil {
		return *err
	}
	return nil
}

// Start starts the metric recorder and writes the metrics to the writer at the specified interval.
// MetricRecorder runs a loop at the specified interval, gathers metrics, formats them using the provided Formatter and writes them using the provided Writer.
// For practical purposes, most often is a FileWriter uses to write the metrics to a file.
func (r *MetricRecorder) Start(ctx context.Context) {
	stdout := output.Std
	stdoutUnsafe := stdout.Direct()

	t := time.NewTicker(r.samplingInterval)
	defer t.Stop()

	previous := &Metrics{}
	for {
		select {
		case <-ctx.Done():
			if err := r.writer.Close(ctx); err != nil {
				r.closeErr.Store(&err)
				stdoutUnsafe.Warnf("failed to close writer: %v\n", err)
			}
			return
		case <-t.C:
			metrics, _ := r.record()
			// write the aggregated metrics
			if err := r.write(ctx, metrics, previous); err != nil {
				r.writeErr.Store(&err)
			} else {
				r.samples.Add(1)
			}
			previous = metrics
		}
	}
}

func (r *MetricRecorder) write(ctx context.Context, metrics, previous *Metrics) error {
	// Build each set of metrics
	memoryMetrics := r.format.Format("memory", r.tags, r.buildMemoryFields(metrics))
	cpuMetrics := r.format.Format("cpu", r.tags, r.buildCPUFields(metrics))
	networkMetrics := r.format.Format("network", r.tags, r.buildNetworkFields(metrics, previous))
	diskMetrics := r.format.Format("disk", r.tags, r.buildDiskFields(metrics, previous))

	// Combine all metrics so we can write them all at once
	data := fmt.Sprintf("%s\n%s\n%s\n%s", memoryMetrics, cpuMetrics, networkMetrics, diskMetrics)
	if err := r.writer.Write(ctx, data); err != nil {
		return errors.Wrap(err, "failed to write combined metrics")
	}

	return nil
}

type Config struct {
	// Dir is the directory where the metrics will be persisted
	Dir string
	// Skip indicated whether to skip the metrics recording.
	// This is used for internal actions like git operations, artifact scraping...
	Skip            bool
	ExecutionConfig ExecutionConfig
	// Format specifies in which format to record the metrics.
	Format core.MetricsFormat
	// Resources specifies the requests and limits of the resources used by the operation.
	ContainerResources core.ContainerResources
}

type ExecutionConfig struct {
	Workflow string
	// Step is a reference to the step in the workflow.
	Step      string
	Execution string
	// Resource is the unique identifier of a container step
	Resource string
}

// WithMetricsRecorder runs the provided function and records the metrics in the specified directory.
// If Config.Skip is set to true, the provided function will be run without recording metrics.
// If there is an error with initiating the metrics recorder, the function will be run without recording metrics.
func WithMetricsRecorder(config Config, fn func(), postProcessFn func() error) {
	var err error
	defer func() {
		if err != nil {
			instructions.PrintOutput(
				config.ExecutionConfig.Step,
				core.ResourceMetricsWarningOutputName,
				instructions.NewExecutionWarning("resource-metrics", "Resource Metrics Issue", err.Error()),
			)
		}
	}()

	stdout := output.Std
	stdoutUnsafe := stdout.Direct()

	// Skip will be set to true for internal operations like git operations, artifact scraping...
	if config.Skip {
		fn()
		return
	}

	cancelCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	metadata := &core.Metadata{
		Workflow:           config.ExecutionConfig.Workflow,
		Step:               core.Step{Ref: config.ExecutionConfig.Step},
		Execution:          config.ExecutionConfig.Execution,
		Format:             config.Format,
		Resource:           config.ExecutionConfig.Resource,
		ContainerResources: config.ContainerResources,
	}
	w, err := core.NewFileWriter(config.Dir, metadata, 4)
	// If we can't create the file writer, log the error, run the function without metrics and exit early.
	if err != nil {
		stdoutUnsafe.Warnf("failed to create file writer: %v\n", err)
		stdoutUnsafe.Warn("running the provided function without metrics recorder\n")
		fn()
		return
	}
	// create the metrics recorder
	r := NewMetricsRecorder(WithWriter(w))
	recorderStopped := make(chan struct{})
	go func() {
		defer close(recorderStopped)
		r.Start(cancelCtx)
	}()
	// run the function
	fn()
	cancel()
	var shutdownErr error
	select {
	case <-recorderStopped:
	case <-time.After(recorderShutdownTimeout):
		shutdownErr = errors.New("timed out waiting for the resource metrics recorder to stop")
		stdoutUnsafe.Warn("timed out waiting for the resource metrics recorder to stop\n")
	}
	if shutdownErr != nil {
		err = shutdownErr
		return
	}
	if err = r.CloseError(); err != nil {
		return
	}
	if err = postProcessFn(); err != nil {
		stdoutUnsafe.Warnf("failed to run post process function: %v\n", err)
		return
	}
	if err = r.WriteError(); err != nil {
		stdoutUnsafe.Warnf("failed to write resource metrics: %v\n", err)
		return
	}
	if r.Samples() > 0 {
		return
	}
	instructions.PrintOutput(
		config.ExecutionConfig.Step,
		core.ResourceMetricsStatusOutputName,
		core.ResourceMetricsStatus{Reason: core.ResourceMetricsReasonNoSamples},
	)
}
