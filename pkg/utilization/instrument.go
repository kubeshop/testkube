package utilization

import (
	errors2 "errors"
	"regexp"
	"sync"

	"github.com/shirou/gopsutil/v4/disk"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

// This regex matches:
//   - sd + exactly one letter (e.g. sda, sdb)
//   - vd + exactly one letter (e.g. vda, vdb)
//   - nvme + digits + n + digits (e.g. nvme0n1, nvme1n1, not partitions)
var reBaseDevice = regexp.MustCompile(`^(sd[a-z]|vd[a-z]|xvd[a-z]|nvme\d+n\d+)$`)

type Metrics struct {
	Memory  *process.MemoryInfoStat
	CPU     float64
	Disk    *disk.IOCountersStat
	Network *net.IOCountersStat
}

// record captures a single metrics data point for all processes .
func (r *MetricRecorder) record() (*Metrics, error) {
	processes, err := process.Processes()
	if err != nil {
		return nil, err
	}

	metrics := make([]*Metrics, len(processes))
	wg := sync.WaitGroup{}
	wg.Add(len(processes))
	for i := range processes {
		go func(i int) {
			defer wg.Done()
			m, err := instrument(processes[i])
			if err != nil {
				return
			}
			metrics[i] = m
		}(i)
	}
	wg.Wait()
	// aggregate CPU and Memory metrics as they are fetched per process
	aggregated := aggregate(metrics)
	// fetch Disk and Network metrics and add them to the aggregated metrics
	r.recordSystemWideMetrics(aggregated)

	return aggregated, nil
}

// instrument captures the metrics of the provided process.
func instrument(process *process.Process) (*Metrics, error) {
	var errs []error
	cpu, err := process.CPUPercent()
	if err != nil {
		errs = append(errs, errors.Wrapf(err, "failed to get cpu info"))
	}
	mem, err := process.MemoryInfo()
	if err != nil {
		errs = append(errs, errors.Wrapf(err, "failed to get memory info"))
	}
	m := &Metrics{
		CPU:    cpu,
		Memory: mem,
	}
	return m, errors2.Join(errs...)
}

// aggregate aggregates the metrics from multiple processes.
// Some test tools might spawn multiple processes to run the tests.
// Example: when executing JMeter, the entry process is a shell script which spawns the actual JMeter Java process.
func aggregate(metrics []*Metrics) *Metrics {
	aggregated := &Metrics{
		Memory: &process.MemoryInfoStat{},
		CPU:    0,
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
	}
	return aggregated
}

// recordSystemWideMetrics captures the network and disk system-wide metrics by using the global gopsutil packages instead of the process one.
func (r *MetricRecorder) recordSystemWideMetrics(aggregated *Metrics) {
	io, _ := disk.IOCounters()
	if len(io) > 0 {
		aggregated.Disk = filterAndAggregateDiskStats(io)
	}
	n, _ := net.IOCounters(false)
	if len(n) > 0 {
		n := n[0]
		aggregated.Network = &n
	}
}

// filterAndAggregateDiskStats filters stats for disk devices (but not partitions).
// It matches:
//   - sda, sdb, sdc, etc. (SCSI/SATA/SAS)
//   - vda, vdb, etc. (VirtIO)
//   - xvda, xvdb, etc. (AWS Xen)
//   - nvme0n1, nvme1n1, etc. (NVMe)
//
// It does NOT match sda1, sdb2, nvme0n1p1, etc.
func filterAndAggregateDiskStats(stats map[string]disk.IOCountersStat) *disk.IOCountersStat {
	aggregated := &disk.IOCountersStat{}

	for diskName, stat := range stats {
		if reBaseDevice.MatchString(diskName) {
			aggregateDiskStats(aggregated, &stat)
		}
	}

	return aggregated
}

func aggregateDiskStats(aggregate, stats *disk.IOCountersStat) {
	aggregate.ReadCount += stats.ReadCount
	aggregate.WriteCount += stats.WriteCount
	aggregate.ReadBytes += stats.ReadBytes
	aggregate.WriteBytes += stats.WriteBytes
	aggregate.ReadTime += stats.ReadTime
	aggregate.WriteTime += stats.WriteTime
	aggregate.IopsInProgress += stats.IopsInProgress
	aggregate.IoTime += stats.IoTime
	aggregate.WeightedIO += stats.WeightedIO
	aggregate.MergedReadCount += stats.MergedReadCount
	aggregate.MergedWriteCount += stats.MergedWriteCount
}
