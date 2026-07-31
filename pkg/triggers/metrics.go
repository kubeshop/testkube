package triggers

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// watcherStartsCount makes relist storms caused by repeated leadership changes visible.
var watcherStartsCount = promauto.NewCounter(prometheus.CounterOpts{
	Name: "testkube_triggers_watcher_starts_total",
	Help: "The total number of times the trigger service started its informer fleet",
})

var leaderGauge = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "testkube_triggers_leader",
	Help: "Whether this instance currently holds the trigger service lease",
})
