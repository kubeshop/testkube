package core

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
)

func TestCalculateAggregations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		metrics []*DataPoint
		want    GroupedAggregations
	}{
		{
			name:    "empty slice of metrics",
			metrics: []*DataPoint{},
			want:    GroupedAggregations{}, // no aggregations
		},
		{
			name: "single data point",
			metrics: []*DataPoint{
				{Measurement: "cpu", Field: "millicores", Values: []timestampValueTuple{{0, 10.5}}},
			},
			want: GroupedAggregations{
				"cpu": {
					"millicores": {
						count: 1,
						TestWorkflowExecutionResourceAggregations: testworkflowsv1.TestWorkflowExecutionResourceAggregations{
							Total:  10.5,
							Min:    10.5,
							Max:    10.5,
							Avg:    10.5,
							StdDev: 0.0, // single data point => StdDev=0
						},
					},
				},
			},
		},
		{
			name: "multiple data points, single field",
			metrics: []*DataPoint{
				{Measurement: "cpu", Field: "millicores", Values: []timestampValueTuple{{0, 10}, {1, 20}, {2, 30}}},
			},
			want: GroupedAggregations{
				"cpu": {
					"millicores": {
						count: 3,
						TestWorkflowExecutionResourceAggregations: testworkflowsv1.TestWorkflowExecutionResourceAggregations{
							Total:  60,
							Min:    10,
							Max:    30,
							Avg:    20,
							StdDev: 10,
						},
					},
				},
			},
		},
		{
			name: "multiple fields",
			metrics: []*DataPoint{
				{Measurement: "cpu", Field: "millicores", Values: []timestampValueTuple{{0, 5}, {1, 15}}},
				{Measurement: "mem", Field: "used", Values: []timestampValueTuple{{0, 100}, {1, 300}}},
			},
			want: GroupedAggregations{
				// CPU data: [5, 15]
				"cpu": {
					"millicores": {
						count: 2,
						TestWorkflowExecutionResourceAggregations: testworkflowsv1.TestWorkflowExecutionResourceAggregations{
							Total:  20,
							Min:    5,
							Max:    15,
							Avg:    10,
							StdDev: 7.071,
						},
					},
				},
				"mem": {
					"used": {
						count: 2,
						TestWorkflowExecutionResourceAggregations: testworkflowsv1.TestWorkflowExecutionResourceAggregations{
							Total:  400,
							Min:    100,
							Max:    300,
							Avg:    200,
							StdDev: 141.421,
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			got := CalculateAggregations(tc.metrics)

			if len(got) != len(tc.want) {
				t.Fatalf("expected %d fields, got %d", len(tc.want), len(got))
			}

			// Check each field's aggregation
			for measurementName, byFields := range tc.want {
				for fieldName, wantAgg := range byFields {
					agg, ok := got[measurementName][fieldName]
					if !ok {
						t.Fatalf("missing aggregation for measurement %q and field %q", measurementName, fieldName)
					}

					// Check unexported field `count`
					assert.Equal(t, wantAgg.count, agg.count, "count not equal")
					assert.Equal(t, wantAgg.Total, agg.Total, "total not equal")
					assert.Equal(t, wantAgg.Min, agg.Min, "min not equal")
					assert.Equal(t, wantAgg.Max, agg.Max, "max not equal")
					assert.Equal(t, wantAgg.Avg, agg.Avg, "avg not equal")

					assertFloatNear(t, wantAgg.StdDev, agg.StdDev, 1e-3)
				}

			}
		})
	}
}

func assertFloatNear(t *testing.T, want, got, tolerance float64) {
	t.Helper()
	delta := math.Abs(got - want)
	if delta > tolerance {
		t.Errorf("expected tolerance to be max %f, got %f", tolerance, delta)
	}
}
