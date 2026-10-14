package boards

import (
	"regexp"
	"strings"
)

// Aggregates the dashboard offers for a time-series measure, as its
// getAggregateOptions does. The first is the one it picks for a new measure.
var (
	aggregatesCount        = []string{"sum"}
	aggregatesAll          = []string{"sum", "avg", "min", "max"}
	aggregatesMaxFirst     = []string{"max", "min"}
	aggregatesMinFirst     = []string{"min", "max"}
	aggregatesAvg          = []string{"avg"}
	aggregatesMaxPreferred = []string{"max", "avg", "min", "sum"}
)

var (
	minMetricPattern  = regexp.MustCompile(`(^|[_-])min($|[_-])`)
	maxMetricPattern  = regexp.MustCompile(`(^|[_-])max($|[_-])`)
	rateMetricPattern = regexp.MustCompile(`(^|[_-])rate($|[_-])`)
)

// countMetricPatterns and maxMetricPatterns mirror isCountMetric and
// prefersMaxAggregate in the dashboard's metricGroups.ts.
var (
	countMetricPatterns = []string{"_count_", "count_total", "sample_count", "request_count", "requests_completed",
		"scenarios_completed", "scenarios_created", "iterations", "matches", "checks", "http_reqs"}
	maxMetricPatterns = []string{"response_time", "duration", "latency", "waiting", "blocked", "connecting", "receiving",
		"sending", "tls_handshaking", "throughput", "rps", "rate", "pct", "percent", "percentage", "kbytes_per_sec",
		"bytes_per_sec", "data_received", "data_sent"}
)

// AggregateOptions returns the aggregates the dashboard offers for a
// time-series measure, a port of getAggregateOptions. A measure it does not
// recognize, such as a custom metric, is offered every aggregate.
func AggregateOptions(measure string) []string {
	key := strings.ToLower(measure)
	switch {
	case measure == DefaultTimeSeriesMeasure || measure == "case-count":
		return aggregatesCount
	case measure == "execution-duration":
		return aggregatesAll
	case strings.HasSuffix(measure, "-max"):
		return aggregatesMaxFirst
	case strings.HasSuffix(measure, "-min"):
		return aggregatesMinFirst
	case strings.HasSuffix(measure, "-avg"):
		return aggregatesAvg
	case strings.HasSuffix(measure, "-total"):
		return aggregatesCount
	case isMinMetric(key):
		return aggregatesMinFirst
	case isMaxMetric(key):
		return aggregatesMaxFirst
	case isCountMetric(key):
		return aggregatesCount
	case containsAny(key, maxMetricPatterns):
		return aggregatesMaxPreferred
	}
	return aggregatesAll
}

// PreferredAggregate is the aggregate the dashboard picks when a measure is
// chosen: the first of AggregateOptions.
func PreferredAggregate(measure string) string {
	return AggregateOptions(measure)[0]
}

func isMinMetric(key string) bool {
	return minMetricPattern.MatchString(key) || strings.HasSuffix(key, "_min_ms") || strings.HasSuffix(key, "_min")
}

func isMaxMetric(key string) bool {
	return maxMetricPattern.MatchString(key) || strings.HasSuffix(key, "_max_ms") || strings.HasSuffix(key, "_max")
}

func isCountMetric(key string) bool {
	if rateMetricPattern.MatchString(key) {
		return false
	}
	return strings.HasSuffix(key, "_count") || containsAny(key, countMetricPatterns)
}

func containsAny(key string, patterns []string) bool {
	for _, p := range patterns {
		if strings.Contains(key, p) {
			return true
		}
	}
	return false
}
