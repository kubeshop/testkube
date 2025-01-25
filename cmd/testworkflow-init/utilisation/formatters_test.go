package utilisation

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestInfluxDBLineProtocolFormatter_Format(t *testing.T) {
	t.Parallel()

	mockTime := func() time.Time {
		return time.Unix(0, 1672531200000000000) // Fixed timestamp for all test cases
	}

	formatter := &InfluxDBLineProtocolFormatter{
		now: mockTime,
	}

	tests := []struct {
		name     string
		metric   string
		tags     []KeyValue
		fields   []KeyValue
		expected string
	}{
		{
			name:   "Basic metric with tags and fields",
			metric: "cpu",
			tags: []KeyValue{
				{"host", "server01"},
				{"region", "us-west"},
			},
			fields: []KeyValue{
				{"usage_user", "0.45"},
				{"usage_system", "0.35"},
			},
			expected: "cpu,host=server01,region=us-west usage_user=0.45,usage_system=0.35 1672531200000000000",
		},
		{
			name:     "Metric with no tags or fields",
			metric:   "memory",
			tags:     []KeyValue{},
			fields:   []KeyValue{},
			expected: "memory 1672531200000000000", // No tags or fields
		},
		{
			name:   "Metric with special characters in keys and values",
			metric: "disk",
			tags: []KeyValue{
				{"path", "/var/log"},
				{"mount", "C:\\drive"},
			},
			fields: []KeyValue{
				{"used_percent", "85.2"},
			},
			expected: "disk,path=/var/log,mount=C:\\drive used_percent=85.2 1672531200000000000",
		},
		{
			name:   "Metric with a single tag and field",
			metric: "network",
			tags: []KeyValue{
				{"interface", "eth0"},
			},
			fields: []KeyValue{
				{"tx", "12345"},
			},
			expected: "network,interface=eth0 tx=12345 1672531200000000000",
		},
		{
			name:   "Metric with multiple tags and no fields",
			metric: "process",
			tags: []KeyValue{
				{"pid", "1234"},
				{"status", "running"},
			},
			fields:   []KeyValue{},
			expected: "process,pid=1234,status=running 1672531200000000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.Format(tt.metric, tt.tags, tt.fields)
			assert.Equal(t, tt.expected, result)
		})
	}
}
