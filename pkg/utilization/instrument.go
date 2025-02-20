package utilization

import (
	"context"
	"fmt"
	"math"
	"os"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/v4/net"
	gopsutil "github.com/shirou/gopsutil/v4/process"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/output"
	"github.com/kubeshop/testkube/pkg/utilization/core"
)

type Metrics struct {
	Memory  *gopsutil.MemoryInfoStat
	CPU     float64
	Disk    *gopsutil.IOCountersStat
	Network *net.IOCountersStat
}

func (r *MetricRecorder) iterate(ctx context.Context, process *gopsutil.Process, previous *Metrics) *Metrics {
	stdout := output.Std
	stdoutUnsafe := stdout.Direct()

	// Instrument the current process
	metrics, err := instrument(process)
	if err != nil {
		stdoutUnsafe.Error(fmt.Sprintf("failed to gather some metrics: %v\n", err))
	}

	// Build each set of metrics
	memoryMetrics := r.format.Format("memory", r.tags, r.buildMemoryFields(metrics))
	cpuMetrics := r.format.Format("cpu", r.tags, r.buildCPUFields(metrics))
	networkMetrics := r.format.Format("network", r.tags, r.buildNetworkFields(metrics, previous))
	diskMetrics := r.format.Format("disk", r.tags, r.buildDiskFields(metrics, previous))

	// Combine all metrics so we can write them all at once
	data := fmt.Sprintf("%s\n%s\n%s\n%s", memoryMetrics, cpuMetrics, networkMetrics, diskMetrics)
	if err := r.writer.Write(ctx, data); err != nil {
		stdoutUnsafe.Error(fmt.Sprintf("failed to write memory metrics: %v\n", err))
	}

	return metrics
}

func instrument(process *gopsutil.Process) (*Metrics, []error) {
	var errs []error
	mem, err := process.MemoryInfo()
	if err != nil {
		errs = append(errs, errors.Wrapf(err, "failed to get memory info"))
	}
	cpu, err := process.CPUPercent()
	if err != nil {
		errs = append(errs, errors.Wrapf(err, "failed to get cpu info"))
	}
	io, err := process.IOCounters()
	if err != nil {
		errs = append(errs, errors.Wrapf(err, "failed to get cpu info"))
	}
	net, err := net.IOCounters(false)
	if err != nil {
		errs = append(errs, errors.Wrapf(err, "failed to get network info"))
	}
	m := &Metrics{
		Memory: mem,
		CPU:    cpu,
		Disk:   io,
	}
	if len(net) > 0 {
		m.Network = &net[0]
	}
	return m, errs
}

func (r *MetricRecorder) buildMemoryFields(metrics *Metrics) []core.KeyValue {
	if metrics.Memory == nil {
		return nil
	}
	return []core.KeyValue{
		core.NewKeyValue("used", fmt.Sprintf("%d", metrics.Memory.RSS)),
	}
}

func (r *MetricRecorder) buildCPUFields(metrics *Metrics) []core.KeyValue {
	return []core.KeyValue{
		core.NewKeyValue("percent", fmt.Sprintf("%.2f", metrics.CPU)),
		core.NewKeyValue("millicores", fmt.Sprintf("%d", int64(math.Round(metrics.CPU*10)))),
	}
}

func (r *MetricRecorder) buildNetworkFields(current, previous *Metrics) []core.KeyValue {
	if current.Network == nil {
		return nil
	}
	bytesSent := current.Network.BytesSent
	bytesRecv := current.Network.BytesRecv
	values := []core.KeyValue{
		core.NewKeyValue("bytes_sent_total", fmt.Sprintf("%d", bytesSent)),
		core.NewKeyValue("bytes_recv_total", fmt.Sprintf("%d", bytesRecv)),
	}
	if previous.Network != nil {
		previousBytesSent := previous.Network.BytesSent
		previousBytesRecv := previous.Network.BytesRecv
		values = append(
			values,
			core.NewKeyValue("bytes_sent_per_s", fmt.Sprintf("%d", bytesSent-previousBytesSent)),
			core.NewKeyValue("bytes_recv_per_s", fmt.Sprintf("%d", bytesRecv-previousBytesRecv)),
		)
	}

	return values
}

func (r *MetricRecorder) buildDiskFields(current, previous *Metrics) []core.KeyValue {
	if current.Disk == nil {
		return nil
	}

	readBytes := current.Disk.DiskReadBytes
	writeBytes := current.Disk.DiskWriteBytes
	values := []core.KeyValue{
		core.NewKeyValue("read_bytes_total", fmt.Sprintf("%d", readBytes)),
		core.NewKeyValue("write_bytes_total", fmt.Sprintf("%d", writeBytes)),
	}
	if previous.Disk != nil {
		previousDiskReadBytes := previous.Disk.DiskReadBytes
		previousDiskWriteBytes := previous.Disk.DiskWriteBytes
		values = append(
			values,
			core.NewKeyValue("read_bytes_per_s", fmt.Sprintf("%d", readBytes-previousDiskReadBytes)),
			core.NewKeyValue("write_bytes_per_s", fmt.Sprintf("%d", writeBytes-previousDiskWriteBytes)),
		)
	}

	return values
}

// getChildProcess tries to find the child process of the current process.
// The child process is the process which is running the underlying test.
func getChildProcess() (*gopsutil.Process, error) {
	var processes []*gopsutil.Process
	var err error
	// We need to retry a few times to get the process because a race condition might occur where the child process is not yet created.
	for i := 0; i < 5; i++ {
		processes, err = gopsutil.Processes()
		if err != nil {
			return nil, errors.Wrap(err, "failed to list running processes")
		}
		if len(processes) > 1 {
			break
		}
	}

	// Print debug info
	fmt.Printf("processes: %v\n", processes)
	fmt.Printf("num processes: %d\n", len(processes))

	// Find the pid of the process which is running the underlying binary.
	pid := int32(os.Getpid())
	var process *gopsutil.Process
	for _, p := range processes {
		ppid, err := p.Ppid()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get parent process id for process pid: %d", p.Pid)
		}
		if p.Pid != pid && ppid == pid {
			process = p
			break
		}
	}
	// If the process is not found, return an error.
	if process == nil {
		return nil, errors.New("failed to find process")
	}

	// Print debug info
	fmt.Printf("child process: %d\n", process.Pid)
	name, _ := process.Name()
	fmt.Printf("child process name: %s\n", name)

	return process, nil
}
