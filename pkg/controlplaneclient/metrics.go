package controlplaneclient

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Live-log streaming metrics. Registered on the default registry via promauto,
// which is what pkg/server/httpserver.go serves at /metrics.
var (
	liveLogSessions = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "testkube_live_log_sessions",
		Help: "Current number of live-log streaming sessions by state",
	}, []string{"kind", "state"})

	liveLogReplayBytes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "testkube_live_log_replay_bytes",
		Help: "Approximate bytes held in live-log replay buffers",
	}, []string{"kind"})

	liveLogResumeTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "testkube_live_log_resume_total",
		Help: "Total live-log resume attempts by result",
	}, []string{"kind", "result"})

	liveLogSessionsCreatedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "testkube_live_log_sessions_created_total",
		Help: "Total live-log streaming sessions created",
	}, []string{"kind"})

	liveLogSessionsEvictedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "testkube_live_log_sessions_evicted_total",
		Help: "Total live-log streaming sessions evicted by reason",
	}, []string{"kind", "reason"})

	liveLogSubscribers = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "testkube_live_log_subscribers",
		Help: "Current number of live-log stream subscribers",
	}, []string{"kind"})

	liveLogSourceDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "testkube_live_log_source_duration_seconds",
		Help:    "Duration of live-log source lifetimes in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"kind"})
)
