package utilization

import (
	"context"
	errors2 "errors"
	"fmt"
	"math"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/v4/net"
	gopsutil "github.com/shirou/gopsutil/v4/process"

	"github.com/kubeshop/testkube/pkg/utilization/core"
)

type Metrics struct {
	Memory  *gopsutil.MemoryInfoStat
	CPU     float64
	Disk    *gopsutil.IOCountersStat
	Network *net.IOCountersStat
}

// record records the metrics of the provided process.
func (r *MetricRecorder) record(process *gopsutil.Process) (*Metrics, error) {
	// Instrument the current process
	metrics, errs := instrument(process)
	if len(errs) > 0 {
		return nil, errors.Wrapf(errors2.Join(errs...), "failed to gather some metrics for process with pid %q", process.Pid)
	}

	return metrics, nil
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
		var bytesSentRate, bytesRecvRate uint64
		// This safety guard is because if a network interface is removed,
		// the bytes sent and received will be removed from the calculation,
		// and we can end up with lower values than the previous ones.
		// Issue: https://github.com/shirou/gopsutil/issues/511
		if bytesSent > previousBytesSent {
			bytesSentRate = bytesSent - previousBytesSent
		}
		if bytesRecv > previousBytesRecv {
			bytesRecvRate = bytesRecv - previousBytesRecv
		}
		values = append(
			values,
			core.NewKeyValue("bytes_sent_per_s", fmt.Sprintf("%d", bytesSentRate)),
			core.NewKeyValue("bytes_recv_per_s", fmt.Sprintf("%d", bytesRecvRate)),
		)
	}

	return values
}

func (r *MetricRecorder) buildDiskFields(current, previous *Metrics) []core.KeyValue {
	if current.Disk == nil {
		return nil
	}

	diskReadBytes := current.Disk.DiskReadBytes
	diskWriteBytes := current.Disk.DiskWriteBytes
	values := []core.KeyValue{
		core.NewKeyValue("read_bytes_total", fmt.Sprintf("%d", diskReadBytes)),
		core.NewKeyValue("write_bytes_total", fmt.Sprintf("%d", diskWriteBytes)),
	}
	if previous.Disk != nil {
		previousDiskReadBytes := previous.Disk.DiskReadBytes
		previousDiskWriteBytes := previous.Disk.DiskWriteBytes
		var diskReadBytesRate, diskWriteBytesRate uint64
		// This safety guard is because if a disk is unmounted,
		// the bytes sent and received will be removed from the calculation,
		// and we can end up with lower values than the previous ones.
		// Issue: https://github.com/shirou/gopsutil/issues/511
		if diskReadBytes > previousDiskReadBytes {
			diskReadBytesRate = diskReadBytes - previousDiskReadBytes
		}
		if diskWriteBytes > previousDiskWriteBytes {
			diskWriteBytesRate = diskWriteBytes - previousDiskWriteBytes
		}
		values = append(
			values,
			core.NewKeyValue("read_bytes_per_s", fmt.Sprintf("%d", diskReadBytesRate)),
			core.NewKeyValue("write_bytes_per_s", fmt.Sprintf("%d", diskWriteBytesRate)),
		)
	}

	return values
}

// aggregate aggregates the metrics from multiple processes.
// Some test tools might spawn multiple processes to run the tests.
// Example: when executing JMeter, the entry process is a shell script which spawns the actual JMeter Java process.
func aggregate(metrics []*Metrics) *Metrics {
	aggregated := &Metrics{
		Memory:  &gopsutil.MemoryInfoStat{},
		CPU:     0,
		Disk:    &gopsutil.IOCountersStat{},
		Network: &net.IOCountersStat{},
	}
	for _, m := range metrics {
		if m == nil {
			continue
		}
		if m.Memory != nil {
			aggregated.Memory.RSS += m.Memory.RSS
			aggregated.Memory.VMS += m.Memory.VMS
			aggregated.Memory.Swap += m.Memory.Swap
			aggregated.Memory.Data += m.Memory.Data
			aggregated.Memory.Stack += m.Memory.Stack
			aggregated.Memory.Locked += m.Memory.Locked
			aggregated.Memory.Stack += m.Memory.Stack
		}
		aggregated.CPU += m.CPU
		if m.Disk != nil {
			aggregated.Disk.ReadCount += m.Disk.ReadCount
			aggregated.Disk.WriteCount += m.Disk.WriteCount
			aggregated.Disk.ReadBytes += m.Disk.ReadBytes
			aggregated.Disk.WriteBytes += m.Disk.WriteBytes
			aggregated.Disk.DiskReadBytes += m.Disk.DiskReadBytes
			aggregated.Disk.DiskWriteBytes += m.Disk.DiskWriteBytes
		}
		if m.Network != nil {
			aggregated.Network.BytesSent += m.Network.BytesSent
			aggregated.Network.BytesRecv += m.Network.BytesRecv
			aggregated.Network.PacketsSent += m.Network.PacketsSent
			aggregated.Network.PacketsRecv += m.Network.PacketsRecv
			aggregated.Network.Errin += m.Network.Errin
			aggregated.Network.Errout += m.Network.Errout
			aggregated.Network.Dropin += m.Network.Dropin
			aggregated.Network.Dropout += m.Network.Dropout
		}
	}
	return aggregated
}
