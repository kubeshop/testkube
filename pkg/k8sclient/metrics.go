package k8sclient

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"k8s.io/client-go/rest"
)

// requestFailedCode labels responses that never produced a status code, so that transport
// failures stay visible without adding an unbounded label value.
const requestFailedCode = "error"

// The client-go hook for this (k8s.io/client-go/tools/metrics.Register) is deliberately not
// used: it accepts a single set of adapters process wide, guarded by an internal sync.Once,
// and controller-runtime claims it from a package init before any of our code runs. A second
// caller is silently ignored, so adapters registered here would never be invoked in a binary
// that links controller-runtime. Wrapping the transport keeps the metrics ours regardless.
type clientMetrics struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

// defaultClientMetrics reports into the default registerer, which the API server already
// serves on /metrics.
var defaultClientMetrics = newClientMetrics(prometheus.DefaultRegisterer)

// Label values are deliberately kept bounded: the response code and the HTTP method only,
// never a namespace, object name, path or host.
func newClientMetrics(registerer prometheus.Registerer) *clientMetrics {
	factory := promauto.With(registerer)

	return &clientMetrics{
		requests: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "testkube_k8s_client_requests_total",
			Help: "The total number of Kubernetes API requests, partitioned by response code and HTTP method",
		}, []string{"code", "method"}),
		duration: factory.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "testkube_k8s_client_request_duration_seconds",
			Help:    "The duration of Kubernetes API requests until the response headers arrived, partitioned by HTTP method",
			Buckets: prometheus.DefBuckets,
		}, []string{"method"}),
	}
}

// InstrumentConfig makes every client built from config report its Kubernetes API requests.
// Call it on configs obtained elsewhere, for example from controller-runtime, so their traffic
// is counted too.
func InstrumentConfig(config *rest.Config) {
	instrumentConfigWith(config, defaultClientMetrics)
}

func instrumentConfigWith(config *rest.Config, metrics *clientMetrics) {
	config.Wrap(func(next http.RoundTripper) http.RoundTripper {
		return &instrumentedRoundTripper{next: next, metrics: metrics}
	})
}

type instrumentedRoundTripper struct {
	next    http.RoundTripper
	metrics *clientMetrics
}

func (rt *instrumentedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	started := time.Now()
	response, err := rt.next.RoundTrip(req)

	// Watch requests return here once the response headers arrived and stream their body
	// afterwards, so this measures time to first byte rather than the lifetime of a stream.
	rt.metrics.duration.WithLabelValues(req.Method).Observe(time.Since(started).Seconds())

	code := requestFailedCode
	if response != nil {
		code = strconv.Itoa(response.StatusCode)
	}
	rt.metrics.requests.WithLabelValues(code, req.Method).Inc()

	return response, err
}
