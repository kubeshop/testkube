package core

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDataPoints(t *testing.T) {
	f, err := os.Open("testdata/metrics_valid_metadata.influx")
	require.NoError(t, err)

	samples, _, invalidLines, err := ParseMetrics(context.Background(), f, "testdata/metrics_valid_metadata.influx")
	require.NoError(t, err)
	assert.Empty(t, invalidLines)

	dataPoints := GroupMetrics(samples)

	assert.Len(t, dataPoints.Data, 6)
}

func TestGroupMetrics_WithNilTimestamp(t *testing.T) {
	f, err := os.Open("testdata/metrics_no_timestamp.influx")
	require.NoError(t, err)
	defer f.Close()

	samples, _, invalidLines, err := ParseMetrics(context.Background(), f, "testdata/metrics_no_timestamp.influx")
	require.NoError(t, err)
	assert.Empty(t, invalidLines)

	// Should not panic and should skip metrics without timestamps
	dataPoints := GroupMetrics(samples)

	// Only 2 metrics should be included (the ones with timestamps)
	assert.Len(t, dataPoints.Data, 2)
}
