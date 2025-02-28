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
