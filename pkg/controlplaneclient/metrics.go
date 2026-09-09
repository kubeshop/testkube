package controlplaneclient

import (
	"sort"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Live-log streaming metrics, registered on the default registry, which is what
// pkg/server/httpserver.go serves at /metrics.
//
// Counters and the histogram record events at the point where they happen.
// Gauges are read at scrape time from the live session managers through
// liveLogCollector, so they always equal the state the managers hold and no
// code path has to keep them in step.
var (
	liveLogSessionsCreatedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "testkube_live_log_sessions_created_total",
		Help: "Total live-log streaming sessions created",
	}, []string{"kind"})

	liveLogSessionsEvictedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "testkube_live_log_sessions_evicted_total",
		Help: "Total live-log streaming sessions removed from their manager by reason",
	}, []string{"kind", "reason"})

	liveLogResumeTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "testkube_live_log_resume_total",
		Help: "Total live-log resume attempts by result",
	}, []string{"kind", "result"})

	// A source lives as long as the execution it follows, from seconds to hours,
	// so the buckets cover that range instead of the sub-second defaults.
	liveLogSourceDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "testkube_live_log_source_duration_seconds",
		Help:    "Duration of live-log source lifetimes in seconds by result",
		Buckets: []float64{1, 5, 15, 30, 60, 120, 300, 600, 1800, 3600, 7200},
	}, []string{"kind", "result"})

	liveLogSessionsDesc = prometheus.NewDesc(
		"testkube_live_log_sessions",
		"Current number of live-log streaming sessions by state",
		[]string{"kind", "state"}, nil,
	)
	liveLogReplayBytesDesc = prometheus.NewDesc(
		"testkube_live_log_replay_bytes",
		"Approximate bytes held in live-log replay buffers",
		[]string{"kind"}, nil,
	)
	liveLogSubscribersDesc = prometheus.NewDesc(
		"testkube_live_log_subscribers",
		"Current number of live-log stream subscribers",
		[]string{"kind"}, nil,
	)

	// liveLogMetrics is the collector every session manager reports to.
	liveLogMetrics = newLiveLogCollector()
)

func init() {
	prometheus.MustRegister(liveLogMetrics)
}

const (
	liveLogEvictionReasonTTL      = "ttl"
	liveLogEvictionReasonErrored  = "errored"
	liveLogEvictionReasonReplaced = "replaced"

	liveLogResultOK    = "ok"
	liveLogResultError = "error"
)

// liveLogStats is a point-in-time reading of one session manager.
type liveLogStats struct {
	activeSessions int
	doneSessions   int
	subscribers    int
	replayBytes    int
}

func (s liveLogStats) add(other liveLogStats) liveLogStats {
	return liveLogStats{
		activeSessions: s.activeSessions + other.activeSessions,
		doneSessions:   s.doneSessions + other.doneSessions,
		subscribers:    s.subscribers + other.subscribers,
		replayBytes:    s.replayBytes + other.replayBytes,
	}
}

// liveLogStatsSource is a session manager as seen by the collector.
type liveLogStatsSource interface {
	liveLogKind() string
	liveLogStats() liveLogStats
}

// liveLogCollector exposes the gauges of every registered session manager,
// summed by kind. Managers register on creation and leave when their context
// ends.
type liveLogCollector struct {
	mu      sync.Mutex
	sources map[liveLogStatsSource]struct{}
}

func newLiveLogCollector() *liveLogCollector {
	return &liveLogCollector{sources: make(map[liveLogStatsSource]struct{})}
}

func (c *liveLogCollector) add(source liveLogStatsSource) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sources[source] = struct{}{}
}

func (c *liveLogCollector) remove(source liveLogStatsSource) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.sources, source)
}

// totalsByKind reads every source outside the collector lock, because a
// source's stats take its own manager lock.
func (c *liveLogCollector) totalsByKind() map[string]liveLogStats {
	c.mu.Lock()
	sources := make([]liveLogStatsSource, 0, len(c.sources))
	for source := range c.sources {
		sources = append(sources, source)
	}
	c.mu.Unlock()

	totals := make(map[string]liveLogStats)
	for _, source := range sources {
		kind := source.liveLogKind()
		totals[kind] = totals[kind].add(source.liveLogStats())
	}
	return totals
}

func (c *liveLogCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- liveLogSessionsDesc
	ch <- liveLogReplayBytesDesc
	ch <- liveLogSubscribersDesc
}

func (c *liveLogCollector) Collect(ch chan<- prometheus.Metric) {
	totals := c.totalsByKind()
	kinds := make([]string, 0, len(totals))
	for kind := range totals {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	for _, kind := range kinds {
		stats := totals[kind]
		ch <- prometheus.MustNewConstMetric(liveLogSessionsDesc, prometheus.GaugeValue, float64(stats.activeSessions), kind, "active")
		ch <- prometheus.MustNewConstMetric(liveLogSessionsDesc, prometheus.GaugeValue, float64(stats.doneSessions), kind, "done")
		ch <- prometheus.MustNewConstMetric(liveLogReplayBytesDesc, prometheus.GaugeValue, float64(stats.replayBytes), kind)
		ch <- prometheus.MustNewConstMetric(liveLogSubscribersDesc, prometheus.GaugeValue, float64(stats.subscribers), kind)
	}
}
