package common

import (
	"testing"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func Test_ShouldCalculatePercentile_AndGiveNearestIndexResult(t *testing.T) {

	durations := []string{"5.34s", "5.74s", "5.93s", "6.16s", "6.56s", "11.56s"}
	var executions []testkube.ExecutionsMetricsExecutions

	for _, duration := range durations {
		executions = append(executions, testkube.ExecutionsMetricsExecutions{Duration: duration})
	}

	result := CalculateMetrics(executions)
	if result.ExecutionDurationP50 != "5.93s" {
		t.Fatalf("Expected 5.93s but got %s", result.ExecutionDurationP50)
	}
	if result.ExecutionDurationP90 != "11.56s" {
		t.Fatalf("Expected 11.56s but got %s", result.ExecutionDurationP90)
	}
	if result.ExecutionDurationP95 != "11.56s" {
		t.Fatalf("Expected 11.56s but got %s", result.ExecutionDurationP95)
	}
	if result.ExecutionDurationP99 != "11.56s" {
		t.Fatalf("Expected 11.56s but got %s", result.ExecutionDurationP99)
	}
}

func Test_WhenNoDurations_ShouldReturn0FromCalculations(t *testing.T) {
	assert := func(t *testing.T, actual string) {
		if actual != "0s" {
			t.Fatalf("Expected 0s but got %s", actual)
		}
	}

	var executions []testkube.ExecutionsMetricsExecutions

	result := CalculateMetrics(executions)
	assert(t, result.ExecutionDurationP50)
	assert(t, result.ExecutionDurationP90)
	assert(t, result.ExecutionDurationP95)
	assert(t, result.ExecutionDurationP99)
}
