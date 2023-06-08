package common

import (
	"time"

	"github.com/montanaflynn/stats"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/utils"
)

func CalculateMetrics(executionsMetrics []testkube.ExecutionsMetricsExecutions) (metrics testkube.ExecutionsMetrics) {
	metrics.Executions = executionsMetrics
	var durations []float64

	for j, execution := range metrics.Executions {
		if execution.Status == string(testkube.FAILED_ExecutionStatus) {
			metrics.FailedExecutions++
		}
		metrics.TotalExecutions++

		// ignore empty and invalid durations
		duration, err := time.ParseDuration(execution.Duration)
		if err != nil {
			continue
		}
		durations = append(durations, float64(duration))

		metrics.Executions[j].Duration = utils.RoundDuration(duration).String()
		metrics.Executions[j].DurationMs = int32(duration / time.Millisecond)
	}

	if metrics.TotalExecutions > 0 {
		metrics.PassFailRatio = 100 * float64(metrics.TotalExecutions-metrics.FailedExecutions) / float64(metrics.TotalExecutions)
	}

	durationP50 := time.Duration(calculate(durations, 50))
	durationP90 := time.Duration(calculate(durations, 90))
	durationP95 := time.Duration(calculate(durations, 95))
	durationP99 := time.Duration(calculate(durations, 99))

	metrics.ExecutionDurationP50 = utils.RoundDuration(durationP50).String()
	metrics.ExecutionDurationP90 = utils.RoundDuration(durationP90).String()
	metrics.ExecutionDurationP95 = utils.RoundDuration(durationP95).String()
	metrics.ExecutionDurationP99 = utils.RoundDuration(durationP99).String()

	metrics.ExecutionDurationP50ms = int32(durationP50 / time.Millisecond)
	metrics.ExecutionDurationP90ms = int32(durationP90 / time.Millisecond)
	metrics.ExecutionDurationP95ms = int32(durationP95 / time.Millisecond)
	metrics.ExecutionDurationP99ms = int32(durationP99 / time.Millisecond)

	return
}

func calculate(durations []float64, quantile float64) float64 {
	if len(durations) == 0 {
		return 0
	}
	percentile, err := stats.PercentileNearestRank(durations, quantile)
	if err != nil {
		log.DefaultLogger.Errorw("Unable to calculate percentile", "error", err)
		return 0
	}
	return percentile
}
