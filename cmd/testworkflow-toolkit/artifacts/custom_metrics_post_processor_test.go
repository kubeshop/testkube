package artifacts

import "testing"

func TestIsGranularMetricsReport(t *testing.T) {
	tests := []struct {
		name string
		data string
		want bool
	}{
		{
			name: "custom metrics",
			data: `{"kind":"testkube.custom-metrics","version":"v1","metrics":[]}`,
			want: true,
		},
		{
			name: "k6 summary with nested values",
			data: `{"metrics":{"http_req_duration":{"values":{"p(95)":123}}}}`,
			want: true,
		},
		{
			name: "k6 summary export",
			data: `{"metrics":{"http_req_duration":{"avg":24.8,"p(95)":29.7},"http_reqs":{"rate":39.5,"count":1187}}}`,
			want: true,
		},
		{
			name: "artillery report",
			data: `{"aggregate":{"counters":{"http.requests":10}}}`,
			want: true,
		},
		{
			name: "playwright json",
			data: `{"suites":[{"specs":[{"tests":[{"results":[{"status":"passed"}]}]}]}]}`,
			want: true,
		},
		{
			name: "cypress json",
			data: `{"stats":{"tests":1},"tests":[{"title":"works","state":"passed"}]}`,
			want: true,
		},
		{
			name: "plain json",
			data: `{"hello":"world"}`,
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isGranularMetricsReport([]byte(tc.data)); got != tc.want {
				t.Fatalf("expected %t, got %t", tc.want, got)
			}
		})
	}
}
