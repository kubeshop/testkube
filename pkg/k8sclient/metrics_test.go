package k8sclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Driving a real clientset asserts the metrics through the path the agent uses, which calling
// the collectors directly would not. Reading an exact label combination back also guards the
// bounded label set, since an extra label would make WithLabelValues panic here.
func TestInstrumentConfig_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		// stopServer makes the request fail at the transport, before any status code exists.
		stopServer bool
		statusCode int
		wantCode   string
	}{
		{
			name:       "records a served response by code and method",
			statusCode: http.StatusOK,
			wantCode:   "200",
		},
		{
			name:       "records a throttled response",
			statusCode: http.StatusTooManyRequests,
			wantCode:   "429",
		},
		{
			name:       "records a transport failure under a bounded code",
			stopServer: true,
			wantCode:   requestFailedCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(`{"kind":"PodList","apiVersion":"v1","items":[]}`))
			}))
			if tt.stopServer {
				server.Close()
			} else {
				defer server.Close()
			}

			metrics := newClientMetrics(prometheus.NewRegistry())
			config := &rest.Config{Host: server.URL}
			instrumentConfigWith(config, metrics)

			clientset, err := kubernetes.NewForConfig(config)
			require.NoError(t, err)

			_, err = clientset.CoreV1().Pods("testkube").List(context.Background(), metav1.ListOptions{})
			if tt.wantCode == "200" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}

			var requests dto.Metric
			require.NoError(t, metrics.requests.WithLabelValues(tt.wantCode, http.MethodGet).Write(&requests))
			assert.Equal(t, float64(1), requests.GetCounter().GetValue())

			var duration dto.Metric
			histogram := metrics.duration.WithLabelValues(http.MethodGet).(prometheus.Histogram)
			require.NoError(t, histogram.Write(&duration))
			assert.Equal(t, uint64(1), duration.GetHistogram().GetSampleCount())
		})
	}
}
