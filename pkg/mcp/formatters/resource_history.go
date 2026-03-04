package formatters

import (
	"math"
	"strings"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/mcp/utils"
)

// ResourceHistoryResponse is the formatted output for workflow resource history.
type ResourceHistoryResponse struct {
	WorkflowName   string                   `json:"workflowName"`
	ExecutionCount int                      `json:"executionCount"`
	Summary        *ResourceHistorySummary  `json:"summary"`
	Executions     []ResourceHistoryExec    `json:"executions"`
	Outliers       []ResourceHistoryOutlier `json:"outliers,omitempty"`
}

// ResourceHistorySummary contains cross-execution statistics.
type ResourceHistorySummary struct {
	CPU     *MetricSummary `json:"cpu,omitempty"`
	Memory  *MetricSummary `json:"memory,omitempty"`
	Disk    *MetricSummary `json:"disk,omitempty"`
	Network *MetricSummary `json:"network,omitempty"`
}

// MetricSummary holds aggregated stats for a metric across executions.
type MetricSummary struct {
	Mean   float64     `json:"mean"`
	Min    float64     `json:"min"`
	Max    float64     `json:"max"`
	StdDev float64     `json:"stdDev"`
	Trend  utils.Trend `json:"trend"`
}

// ResourceHistoryExec is a compact per-execution resource summary.
type ResourceHistoryExec struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	ScheduledAt time.Time `json:"scheduledAt"`
	Status      string    `json:"status"`
	Duration    string    `json:"duration,omitempty"`
	CPU         float64   `json:"cpuMillicores,omitempty"`
	MemoryMB    float64   `json:"memoryMB,omitempty"`
	DiskMB      float64   `json:"diskMB,omitempty"`
	NetworkKBps float64   `json:"networkKBps,omitempty"`
}

// ResourceHistoryOutlier flags an execution with anomalous resource usage.
type ResourceHistoryOutlier struct {
	ExecutionID   string  `json:"executionId"`
	ExecutionName string  `json:"executionName"`
	Metric        string  `json:"metric"`
	Value         float64 `json:"value"`
	ZScore        float64 `json:"zScore"`
	Reason        string  `json:"reason"`
}

// metricsCollector collects metric values across executions for statistical analysis.
type metricsCollector struct {
	cpu     []float64
	memory  []float64
	disk    []float64
	network []float64
}

// FormatWorkflowResourceHistory parses execution list with resource aggregations
// and returns a formatted response with summary statistics and outlier detection.
func FormatWorkflowResourceHistory(raw string, metricsFilter string) (string, error) {
	result, isEmpty, err := ParseJSON[testkube.TestWorkflowExecutionsResult](raw)
	if err != nil {
		return "", err
	}
	if isEmpty || len(result.Results) == 0 {
		return FormatJSON(ResourceHistoryResponse{
			ExecutionCount: 0,
			Executions:     []ResourceHistoryExec{},
		})
	}

	// Parse metrics filter
	includeMetrics := parseMetricsFilter(metricsFilter)

	// Collect metrics from all executions
	collector := &metricsCollector{
		cpu:     make([]float64, 0, len(result.Results)),
		memory:  make([]float64, 0, len(result.Results)),
		disk:    make([]float64, 0, len(result.Results)),
		network: make([]float64, 0, len(result.Results)),
	}

	executions := make([]ResourceHistoryExec, 0, len(result.Results))

	// Process executions (API returns newest first, we want oldest first for trend)
	for i := len(result.Results) - 1; i >= 0; i-- {
		exec := result.Results[i]

		re := ResourceHistoryExec{
			ID:          exec.Id,
			Name:        exec.Name,
			ScheduledAt: exec.ScheduledAt,
		}

		if exec.Result != nil {
			if exec.Result.Status != nil {
				re.Status = string(*exec.Result.Status)
			}
			re.Duration = exec.Result.Duration
		}

		// Extract resource metrics
		if exec.ResourceAggregations != nil && exec.ResourceAggregations.Global != nil {
			extractResourceMetrics(exec.ResourceAggregations.Global, &re, collector, includeMetrics)
		}

		executions = append(executions, re)
	}

	// Compute summary statistics
	summary := computeSummary(collector, includeMetrics)

	// Detect outliers (using z-score > 2)
	outliers := detectOutliers(executions, collector, includeMetrics)

	// Build response (reverse executions back to newest-first for display)
	displayExecutions := make([]ResourceHistoryExec, len(executions))
	for i, exec := range executions {
		displayExecutions[len(executions)-1-i] = exec
	}

	workflowName := ""
	if len(result.Results) > 0 && result.Results[0].Workflow != nil {
		workflowName = result.Results[0].Workflow.Name
	}

	response := ResourceHistoryResponse{
		WorkflowName:   workflowName,
		ExecutionCount: len(executions),
		Summary:        summary,
		Executions:     displayExecutions,
		Outliers:       outliers,
	}

	return FormatJSON(response)
}

// parseMetricsFilter parses comma-separated metric names into a map.
func parseMetricsFilter(filter string) map[string]bool {
	if filter == "" {
		return map[string]bool{"cpu": true, "memory": true, "disk": true, "network": true}
	}

	result := make(map[string]bool)
	for _, m := range strings.Split(filter, ",") {
		result[strings.TrimSpace(strings.ToLower(m))] = true
	}
	return result
}

// extractResourceMetrics extracts resource data from aggregations.
func extractResourceMetrics(
	global map[string]map[string]testkube.TestWorkflowExecutionResourceAggregations,
	exec *ResourceHistoryExec,
	collector *metricsCollector,
	include map[string]bool,
) {
	// CPU metrics
	if include["cpu"] {
		if cpu, ok := global["cpu"]; ok {
			if millicores, ok := cpu["millicores"]; ok {
				exec.CPU = millicores.Avg
				collector.cpu = append(collector.cpu, millicores.Avg)
			}
		}
	}

	// Memory metrics
	if include["memory"] {
		if memory, ok := global["memory"]; ok {
			if used, ok := memory["used"]; ok {
				// Convert bytes to MB
				exec.MemoryMB = used.Avg / (1024 * 1024)
				collector.memory = append(collector.memory, exec.MemoryMB)
			}
		}
	}

	// Disk metrics
	if include["disk"] {
		if disk, ok := global["disk"]; ok {
			if used, ok := disk["used"]; ok {
				// Convert bytes to MB
				exec.DiskMB = used.Avg / (1024 * 1024)
				collector.disk = append(collector.disk, exec.DiskMB)
			}
		}
	}

	// Network metrics (combined recv + sent)
	if include["network"] {
		if network, ok := global["network"]; ok {
			var totalKBps float64
			if recv, ok := network["bytes_recv_per_s"]; ok {
				totalKBps += recv.Avg / 1024
			}
			if sent, ok := network["bytes_sent_per_s"]; ok {
				totalKBps += sent.Avg / 1024
			}
			if totalKBps > 0 {
				exec.NetworkKBps = totalKBps
				collector.network = append(collector.network, totalKBps)
			}
		}
	}
}

// computeSummary calculates summary statistics for each metric.
func computeSummary(collector *metricsCollector, include map[string]bool) *ResourceHistorySummary {
	summary := &ResourceHistorySummary{}

	if include["cpu"] && len(collector.cpu) > 0 {
		stats := utils.ComputeStats(collector.cpu)
		summary.CPU = &MetricSummary{
			Mean:   roundTo(stats.Mean, 2),
			Min:    roundTo(stats.Min, 2),
			Max:    roundTo(stats.Max, 2),
			StdDev: roundTo(stats.StdDev, 2),
			Trend:  utils.DetectTrend(collector.cpu),
		}
	}

	if include["memory"] && len(collector.memory) > 0 {
		stats := utils.ComputeStats(collector.memory)
		summary.Memory = &MetricSummary{
			Mean:   roundTo(stats.Mean, 2),
			Min:    roundTo(stats.Min, 2),
			Max:    roundTo(stats.Max, 2),
			StdDev: roundTo(stats.StdDev, 2),
			Trend:  utils.DetectTrend(collector.memory),
		}
	}

	if include["disk"] && len(collector.disk) > 0 {
		stats := utils.ComputeStats(collector.disk)
		summary.Disk = &MetricSummary{
			Mean:   roundTo(stats.Mean, 2),
			Min:    roundTo(stats.Min, 2),
			Max:    roundTo(stats.Max, 2),
			StdDev: roundTo(stats.StdDev, 2),
			Trend:  utils.DetectTrend(collector.disk),
		}
	}

	if include["network"] && len(collector.network) > 0 {
		stats := utils.ComputeStats(collector.network)
		summary.Network = &MetricSummary{
			Mean:   roundTo(stats.Mean, 2),
			Min:    roundTo(stats.Min, 2),
			Max:    roundTo(stats.Max, 2),
			StdDev: roundTo(stats.StdDev, 2),
			Trend:  utils.DetectTrend(collector.network),
		}
	}

	return summary
}

// detectOutliers finds executions with metrics > 2 standard deviations from mean.
func detectOutliers(
	executions []ResourceHistoryExec,
	collector *metricsCollector,
	include map[string]bool,
) []ResourceHistoryOutlier {
	const threshold = 2.0
	outliers := make([]ResourceHistoryOutlier, 0)

	checkMetric := func(values []float64, metricName string, getValue func(int) float64) {
		if len(values) < 3 {
			return
		}
		stats := utils.ComputeStats(values)
		if stats.StdDev == 0 {
			return
		}

		for i, exec := range executions {
			value := getValue(i)
			if value == 0 {
				continue
			}
			if utils.IsOutlier(value, stats.Mean, stats.StdDev, threshold) {
				zScore := utils.ZScore(value, stats.Mean, stats.StdDev)
				direction := "above"
				if zScore < 0 {
					direction = "below"
				}
				outliers = append(outliers, ResourceHistoryOutlier{
					ExecutionID:   exec.ID,
					ExecutionName: exec.Name,
					Metric:        metricName,
					Value:         roundTo(value, 2),
					ZScore:        roundTo(zScore, 2),
					Reason:        formatOutlierReason(metricName, direction, math.Abs(zScore)),
				})
			}
		}
	}

	if include["cpu"] && len(collector.cpu) > 0 {
		checkMetric(collector.cpu, "cpu", func(i int) float64 { return executions[i].CPU })
	}
	if include["memory"] && len(collector.memory) > 0 {
		checkMetric(collector.memory, "memory", func(i int) float64 { return executions[i].MemoryMB })
	}
	if include["disk"] && len(collector.disk) > 0 {
		checkMetric(collector.disk, "disk", func(i int) float64 { return executions[i].DiskMB })
	}
	if include["network"] && len(collector.network) > 0 {
		checkMetric(collector.network, "network", func(i int) float64 { return executions[i].NetworkKBps })
	}

	return outliers
}

// formatOutlierReason generates a human-readable explanation.
func formatOutlierReason(metric, direction string, zScore float64) string {
	severity := "significantly"
	if zScore > 3 {
		severity = "extremely"
	}
	return severity + " " + direction + " average " + metric + " usage"
}

// roundTo rounds a float to n decimal places.
func roundTo(value float64, decimals int) float64 {
	pow := math.Pow(10, float64(decimals))
	return math.Round(value*pow) / pow
}
