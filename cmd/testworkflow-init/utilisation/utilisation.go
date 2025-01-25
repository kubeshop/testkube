package utilisation

import (
	"context"
	"fmt"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/output"
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
	"os"
	"time"
)

const defaultSamplingInterval = 1 * time.Second

type MetricRecorder struct {
	writer           Writer
	format           Formatter
	samplingInterval time.Duration
	tags             []KeyValue
}

type Option func(*MetricRecorder)

func WithWriter(writer Writer) Option {
	return func(u *MetricRecorder) {
		u.writer = writer
	}
}

func WithSamplingInterval(interval time.Duration) Option {
	return func(u *MetricRecorder) {
		u.samplingInterval = interval
	}
}

func WithTags(tags []KeyValue) Option {
	return func(u *MetricRecorder) {
		u.tags = tags
	}
}

func NewMetricsRecorder(opts ...Option) *MetricRecorder {
	u := &MetricRecorder{
		format:           NewInfluxDBLineProtocolFormatter(),
		writer:           NewSTDOUTWriter(),
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

func (r *MetricRecorder) buildMemoryFields(metrics *Metrics) []KeyValue {
	return []KeyValue{
		NewKeyValue("rss", fmt.Sprintf("%d", metrics.Memory.RSS)),
		NewKeyValue("vms", fmt.Sprintf("%d", metrics.Memory.VMS)),
		NewKeyValue("swap", fmt.Sprintf("%d", metrics.Memory.Swap)),
		NewKeyValue("percent", fmt.Sprintf("%f", metrics.MemoryPercent)),
	}
}

func (r *MetricRecorder) buildCPUFields(metrics *Metrics) []KeyValue {
	return []KeyValue{
		NewKeyValue("percent", fmt.Sprintf("%f", metrics.CPUPercent)),
	}
}

func (r *MetricRecorder) buildNetworkFields(metrics *Metrics) []KeyValue {
	var kv []KeyValue
	for _, ifaceStat := range metrics.Network {
		kv = append(kv,
			NewKeyValue(fmt.Sprintf("%s_bytes_sent", ifaceStat.Name), fmt.Sprintf("%d", ifaceStat.BytesSent)),
			NewKeyValue(fmt.Sprintf("%s_bytes_recv", ifaceStat.Name), fmt.Sprintf("%d", ifaceStat.BytesRecv)),
		)
	}
	return kv
}

func (r *MetricRecorder) buildDiskFields(metrics *Metrics) []KeyValue {
	var kv []KeyValue
	for device, stats := range metrics.Disk {
		kv = append(kv,
			NewKeyValue(fmt.Sprintf("%s_read_bytes", device), fmt.Sprintf("%d", stats.ReadBytes)),
			NewKeyValue(fmt.Sprintf("%s_write_bytes", device), fmt.Sprintf("%d", stats.WriteBytes)),
			NewKeyValue(fmt.Sprintf("%s_reads", device), fmt.Sprintf("%d", stats.ReadCount)),
			NewKeyValue(fmt.Sprintf("%s_writes", device), fmt.Sprintf("%d", stats.WriteCount)),
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
