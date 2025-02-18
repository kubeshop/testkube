package core

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	"github.com/stretchr/testify/assert"
)

func TestParseFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		filepath        string
		wantSampleCount int
		wantFirstSample *Metric
		wantLastSample  *Metric
	}{
		{
			name:            "Valid file with metadata",
			filepath:        "testdata/metrics_valid_metadata.influx",
			wantSampleCount: 50,
			wantFirstSample: &Metric{
				Measurement: "cpu",
				Tags: []KeyValue{
					{Key: "host", Value: "server01"},
				},
				Fields: []KeyValue{
					{Key: "usage_user", Value: "0.10"},
					{Key: "usage_system", Value: "0.20"},
					{Key: "usage_idle", Value: "99.70"},
				},
				Timestamp: ptr.To(time.Unix(0, 1670000000000000000).UTC()),
			},
			wantLastSample: &Metric{
				Measurement: "mem",
				Tags: []KeyValue{
					{Key: "host", Value: "server02"},
				},
				Fields: []KeyValue{
					{Key: "usage_total", Value: "8192"},
					{Key: "usage_used", Value: "4900"},
					{Key: "usage_free", Value: "3292"},
				},
				Timestamp: ptr.To(time.Unix(0, 1670000049000000000).UTC()),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			samples, err := ParseMetricsFile(tc.filepath)
			require.NoError(t, err)
			assert.Len(t, samples, tc.wantSampleCount)
			assert.Equal(t, tc.wantFirstSample, samples[0])
			assert.Equal(t, tc.wantLastSample, samples[len(samples)-1])
		})
	}
}

func TestInfluxDBLineProtocolParser(t *testing.T) {
	t.Parallel()

	parser := NewInfluxDBLineProtocolParser()

	// We'll create a known time value for testing timestamp parsing.
	// For example: 2025-02-14 12:34:56 UTC
	testTime := time.Date(2025, 2, 14, 12, 34, 56, 0, time.UTC)
	testNanos := testTime.UnixNano()

	tests := []struct {
		name          string
		input         string
		wantSample    *Metric
		wantErrSubstr string
	}{
		{
			name:  "Valid with measurement + single field (no tags, no timestamp)",
			input: "cpu usage=50.0",
			wantSample: &Metric{
				Measurement: "cpu",
				Tags:        nil,
				Fields: []KeyValue{
					{Key: "usage", Value: "50.0"},
				},
				Timestamp: nil,
			},
		},
		{
			name:  "Valid with measurement, multiple tags, multiple fields, and timestamp",
			input: "cpu,host=server01,region=uswest usage_idle=55.0,usage_busy=45.0 " + strconv.FormatInt(testNanos, 10),
			wantSample: &Metric{
				Measurement: "cpu",
				Tags: []KeyValue{
					{Key: "host", Value: "server01"},
					{Key: "region", Value: "uswest"},
				},
				Fields: []KeyValue{
					{Key: "usage_idle", Value: "55.0"},
					{Key: "usage_busy", Value: "45.0"},
				},
				Timestamp: &testTime,
			},
		},
		{
			name:          "Invalid line (missing fields part)",
			input:         "cpu",
			wantSample:    nil,
			wantErrSubstr: "invalid line protocol",
		},
		{
			name:          "Invalid tag format",
			input:         "cpu,host=server01=extra, usage=60.0",
			wantSample:    nil,
			wantErrSubstr: "invalid tag",
		},
		{
			name:          "Invalid field format",
			input:         "cpu usage=60.0,brokenfield",
			wantSample:    nil,
			wantErrSubstr: "invalid field",
		},
		{
			name:          "Invalid timestamp",
			input:         "cpu usage=60.0 not_a_timestamp",
			wantSample:    nil,
			wantErrSubstr: "invalid timestamp",
		},
		{
			name:  "Valid line with multiple tags, multiple fields, no timestamp",
			input: "temp,location=roomA,device=sensor1 reading=23.5,errors=2",
			wantSample: &Metric{
				Measurement: "temp",
				Tags: []KeyValue{
					{Key: "location", Value: "roomA"},
					{Key: "device", Value: "sensor1"},
				},
				Fields: []KeyValue{
					{Key: "reading", Value: "23.5"},
					{Key: "errors", Value: "2"},
				},
				Timestamp: nil,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotSample, err := parser.Parse([]byte(tc.input))

			if tc.wantErrSubstr != "" {
				// We expect an error containing wantErrSubstr
				assert.Error(t, err, "Expected an error, but got none.")
				assert.Contains(t, err.Error(), tc.wantErrSubstr, "Error message does not contain expected substring.")
				assert.Nil(t, gotSample)
			} else {
				// We do not expect an error
				assert.NoError(t, err)
				assert.NotNil(t, gotSample)
				assert.Equal(t, tc.wantSample, gotSample)
			}
		})
	}
}
