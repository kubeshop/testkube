package utilization

import (
	"fmt"
	"math"

	"github.com/kubeshop/testkube/pkg/utilization/core"
)

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

	diskReadBytes := current.Disk.ReadBytes
	diskWriteBytes := current.Disk.WriteBytes
	values := []core.KeyValue{
		core.NewKeyValue("read_bytes_total", fmt.Sprintf("%d", diskReadBytes)),
		core.NewKeyValue("write_bytes_total", fmt.Sprintf("%d", diskWriteBytes)),
	}
	if previous.Disk != nil {
		previousDiskReadBytes := previous.Disk.ReadBytes
		previousDiskWriteBytes := previous.Disk.WriteBytes
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
