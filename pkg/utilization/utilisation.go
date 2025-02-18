package utilization

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kubeshop/testkube/pkg/utilization/core"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/output"
)

const (
	slowSamplingInterval    = 15 * time.Second
	fastSamplingInterval    = 1 * time.Second
	defaultSamplingInterval = fastSamplingInterval
)

type MetricRecorder struct {
	writer           core.Writer
	format           core.Formatter
	samplingInterval time.Duration
	tags             []core.KeyValue
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

func (r *MetricRecorder) Start(ctx context.Context) {
	stdout := output.Std
	stdoutUnsafe := stdout.Direct()

	stdoutUnsafe.Println("starting metrics recorder")

	t := time.NewTicker(r.samplingInterval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			stdoutUnsafe.Println("stopping metrics recorder")
			if err := r.writer.Close(); err != nil {
				stdoutUnsafe.Error(fmt.Sprintf("failed to close writer: %v\n", err))
			}
			return
		case <-t.C:
			// Instrument the current process
			metrics, err := instrument()
			if err != nil {
				stdoutUnsafe.Error(fmt.Sprintf("failed to instrument current process: %v\n", err))
				continue
			}
			// Build each set of metrics
			memoryMetrics := r.format.Format("memory", r.tags, r.buildMemoryFields(metrics))
			cpuMetrics := r.format.Format("cpu", r.tags, r.buildCPUFields(metrics))
			networkMetrics := r.format.Format("network", r.tags, r.buildNetworkFields(metrics))
			diskMetrics := r.format.Format("disk", r.tags, r.buildDiskFields(metrics))
			// Write each set of metrics
			if err := r.writer.Write(ctx, memoryMetrics); err != nil {
				stdoutUnsafe.Error(fmt.Sprintf("failed to write memory metrics: %v\n", err))
			}
			if err := r.writer.Write(ctx, cpuMetrics); err != nil {
				stdoutUnsafe.Error(fmt.Sprintf("failed to write cpu metrics: %v\n", err))
			}
			if err := r.writer.Write(ctx, networkMetrics); err != nil {
				stdoutUnsafe.Error(fmt.Sprintf("failed to write network metrics: %v\n", err))
			}
			if err := r.writer.Write(ctx, diskMetrics); err != nil {
				stdoutUnsafe.Error(fmt.Sprintf("failed to write disk metrics: %v\n", err))
			}
		}
	}
}

func (r *MetricRecorder) buildMemoryFields(metrics *Metrics) []core.KeyValue {
	return []core.KeyValue{
		core.NewKeyValue("rss", fmt.Sprintf("%d", metrics.Memory.RSS)),
		core.NewKeyValue("vms", fmt.Sprintf("%d", metrics.Memory.VMS)),
		core.NewKeyValue("swap", fmt.Sprintf("%d", metrics.Memory.Swap)),
		core.NewKeyValue("percent", fmt.Sprintf("%f", metrics.MemoryPercent)),
	}
}

func (r *MetricRecorder) buildCPUFields(metrics *Metrics) []core.KeyValue {
	return []core.KeyValue{
		core.NewKeyValue("percent", fmt.Sprintf("%f", metrics.CPUPercent)),
	}
}

func (r *MetricRecorder) buildNetworkFields(metrics *Metrics) []core.KeyValue {
	var kv []core.KeyValue
	for _, ifaceStat := range metrics.Network {
		kv = append(kv,
			core.NewKeyValue(fmt.Sprintf("%s_bytes_sent", ifaceStat.Name), fmt.Sprintf("%d", ifaceStat.BytesSent)),
			core.NewKeyValue(fmt.Sprintf("%s_bytes_recv", ifaceStat.Name), fmt.Sprintf("%d", ifaceStat.BytesRecv)),
		)
	}
	return kv
}

func (r *MetricRecorder) buildDiskFields(metrics *Metrics) []core.KeyValue {
	var kv []core.KeyValue
	for device, stats := range metrics.Disk {
		kv = append(kv,
			core.NewKeyValue(fmt.Sprintf("%s_read_bytes", device), fmt.Sprintf("%d", stats.ReadBytes)),
			core.NewKeyValue(fmt.Sprintf("%s_write_bytes", device), fmt.Sprintf("%d", stats.WriteBytes)),
			core.NewKeyValue(fmt.Sprintf("%s_reads", device), fmt.Sprintf("%d", stats.ReadCount)),
			core.NewKeyValue(fmt.Sprintf("%s_writes", device), fmt.Sprintf("%d", stats.WriteCount)),
		)
	}
	return kv
}

type Metrics struct {
	Memory        *process.MemoryInfoStat
	MemoryPercent float32
	CPUPercent    float64
	Network       []net.IOCountersStat
	Disk          map[string]disk.IOCountersStat
}

func instrument() (*Metrics, error) {
	// Get current process
	p, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get current process")
	}
	// Get process metrics
	mem, err := p.MemoryInfo()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get memory info")
	}
	memPercent, err := p.MemoryPercent()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get memory percent")
	}
	cpuPercent, err := p.CPUPercent()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get CPU info")
	}
	net, err := net.IOCounters(true)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get network info")
	}
	disk, err := disk.IOCounters()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get disk info")
	}
	return &Metrics{
		Memory:        mem,
		MemoryPercent: memPercent,
		CPUPercent:    cpuPercent,
		Network:       net,
		Disk:          disk,
	}, nil
}

type Config struct {
	// Dir is the directory where the metrics will be persisted
	Dir string
	// Skip indicated whether to skip the metrics recording.
	// This is used for internal actions like git operations, artifact scraping...
	Skip      bool
	Workflow  string
	Step      string
	Execution string
	// Format specifies in which format to record the metrics.
	Format core.MetricsFormat
}

// WithMetricsRecorder runs the provided function and records the metrics in the specified directory.
// If Config.Skip is set to true, the provided function will be run without recording metrics.
// If there is an error with initiating the metrics recorder, the function will be run without recording metrics.
func WithMetricsRecorder(config Config, fn func()) {
	stdout := output.Std
	stdoutUnsafe := stdout.Direct()

	// Skip will be set to true for internal operations like git operations, artifact scraping...
	if config.Skip {
		stdoutUnsafe.Println("skipping metrics recording for internal operations")
		fn()
		return
	}

	cancelCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	metadata := &core.Metadata{
		Workflow:  config.Workflow,
		Step:      config.Step,
		Execution: config.Execution,
		Format:    config.Format,
	}
	w, err := core.NewBufferedFileWriter(config.Dir, metadata)
	// If we can't create the file writer, log the error, run the function without metrics and exit early.
	if err != nil {
		stdoutUnsafe.Errorf("failed to create file writer: %v", err)
		stdoutUnsafe.Warn("running the provided function without metrics recorder")
		fn()
		return
	}
	// create the metrics recorder
	r := NewMetricsRecorder(WithWriter(w))
	go func() {
		r.Start(cancelCtx)
	}()
	// run the function
	fn()
	cancel()
}
