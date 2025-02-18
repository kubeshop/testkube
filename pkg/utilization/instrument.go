package utilization

import (
	"context"
	"fmt"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/output"
	"github.com/kubeshop/testkube/pkg/utilization/core"
	"math"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/v4/net"
	gopsutil "github.com/shirou/gopsutil/v4/process"
)

type Metrics struct {
	Memory  *gopsutil.MemoryInfoStat
	CPU     float64
	Disk    *gopsutil.IOCountersStat
	Network net.IOCountersStat
}

func (r *MetricRecorder) iterate(ctx context.Context, process *gopsutil.Process) {
	stdout := output.Std
	stdoutUnsafe := stdout.Direct()

	// Instrument the current process
	metrics, err := instrument(process)
	if err != nil {
		stdoutUnsafe.Error(fmt.Sprintf("failed to instrument current process: %v\n", err))
		return
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

func instrument(process *gopsutil.Process) (*Metrics, error) {
	mem, err := process.MemoryInfo()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get memory info")
	}
	cpu, err := process.CPUPercent()
	io, err := process.IOCounters()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get cpu info")
	}
	net, err := net.IOCounters(false)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get network info")
	}
	return &Metrics{
		Memory:  mem,
		CPU:     cpu,
		Disk:    io,
		Network: net[0],
	}, nil
}

func (r *MetricRecorder) buildMemoryFields(metrics *Metrics) []core.KeyValue {
	return []core.KeyValue{
		core.NewKeyValue("used", fmt.Sprintf("%d", metrics.Memory.RSS)),
	}
}

func (r *MetricRecorder) buildCPUFields(metrics *Metrics) []core.KeyValue {
	return []core.KeyValue{
		core.NewKeyValue("percent", fmt.Sprintf("%f", metrics.CPU)),
		core.NewKeyValue("millicores", fmt.Sprintf("%f", math.Round(metrics.CPU*10))),
	}
}

func (r *MetricRecorder) buildNetworkFields(metrics *Metrics) []core.KeyValue {
	return []core.KeyValue{
		core.NewKeyValue("bytes_sent", fmt.Sprintf("%d", metrics.Network.BytesSent)),
		core.NewKeyValue("bytes_recv", fmt.Sprintf("%d", metrics.Network.BytesRecv)),
	}
}

func (r *MetricRecorder) buildDiskFields(metrics *Metrics) []core.KeyValue {
	return []core.KeyValue{
		core.NewKeyValue("read_bytes", fmt.Sprintf("%d", metrics.Disk.DiskReadBytes)),
		core.NewKeyValue("write_bytes", fmt.Sprintf("%d", metrics.Disk.DiskWriteBytes)),
	}
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
		time.Sleep(1)
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
