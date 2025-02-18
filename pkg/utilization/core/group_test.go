package core

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDataPoints(t *testing.T) {
	samples, err := OpenAndParseMetricsFile("testdata/metrics_valid_metadata.influx")
	require.NoError(t, err)

	dataPoints := BuildDataPoints(samples)

	assert.Len(t, dataPoints, 6)

	j, _ := json.Marshal(dataPoints)
	fmt.Println(string(j))
}

func assertDataPoint(t *testing.T, dp *DataPoint, expectedMeasurement, expectedTags string, expectedField string, expectedValuesLength int) {
	t.Helper()

	assert.Equal(t, expectedMeasurement, dp.Measurement)
	assert.Equal(t, expectedField, dp.Field)
	assert.Equal(t, expectedValuesLength, len(dp.Values))
}
