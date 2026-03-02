package utilization

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/shirou/gopsutil/v4/disk"
)

func TestFilterAndAggregateDiskStats(t *testing.T) {

	tests := []struct {
		name  string
		stats map[string]disk.IOCountersStat
		want  disk.IOCountersStat
	}{
		{
			name: "Include base sda, skip sda1",
			stats: map[string]disk.IOCountersStat{
				"sda":  {ReadBytes: 1000, WriteBytes: 2000},
				"sda1": {ReadBytes: 3000, WriteBytes: 4000},
			},
			// Only sda is included.
			want: disk.IOCountersStat{ReadBytes: 1000, WriteBytes: 2000},
		},
		{
			name: "Include base nvme0n1, skip nvme0n1p1",
			stats: map[string]disk.IOCountersStat{
				"nvme0n1":   {ReadBytes: 5000, WriteBytes: 6000},
				"nvme0n1p1": {ReadBytes: 7000, WriteBytes: 8000},
			},
			// Only nvme0n1 is included.
			want: disk.IOCountersStat{ReadBytes: 5000, WriteBytes: 6000},
		},
		{
			name: "Include multiple base devices (vda, vdb, sdz, nvme1n1) but skip their partitions",
			stats: map[string]disk.IOCountersStat{
				"vda":       {ReadBytes: 100, WriteBytes: 200},
				"vdb":       {ReadBytes: 500, WriteBytes: 700},
				"vda1":      {ReadBytes: 10, WriteBytes: 20},
				"sdz":       {ReadBytes: 900, WriteBytes: 1100},
				"sdz1":      {ReadBytes: 2000, WriteBytes: 3000},
				"nvme1n1":   {ReadBytes: 4000, WriteBytes: 5000},
				"nvme1n1p2": {ReadBytes: 100, WriteBytes: 100},
				"xvda":      {ReadBytes: 4000, WriteBytes: 5000},
				"xvda1":     {ReadBytes: 100, WriteBytes: 100},
			},
			// Summation of base devices only: vda + vdb + sdz + nvme1n1
			want: disk.IOCountersStat{
				ReadBytes:  100 + 500 + 900 + 4000 + 4000,
				WriteBytes: 200 + 700 + 1100 + 5000 + 5000,
			},
		},
		{
			name: "No matching devices",
			stats: map[string]disk.IOCountersStat{
				"loop0": {ReadBytes: 1000, WriteBytes: 2000},
				"dm-0":  {ReadBytes: 300, WriteBytes: 400},
			},
			// None match the regex, so the result should be zeroed.
			want: disk.IOCountersStat{ReadBytes: 0, WriteBytes: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterAndAggregateDiskStats(tt.stats)
			assert.Equal(t, tt.want.ReadBytes, got.ReadBytes)
			assert.Equal(t, tt.want.WriteBytes, got.WriteBytes)
		})
	}
}
