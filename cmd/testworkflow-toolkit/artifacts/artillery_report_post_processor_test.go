package artifacts

import "testing"

func TestIsArtilleryReport(t *testing.T) {
	tests := []struct {
		name string
		data string
		want bool
	}{
		{
			name: "k6 summary with nested values is handled by dedicated processor",
			data: `{"metrics":{"http_req_duration":{"values":{"p(95)":123}}}}`,
			want: false,
		},
		{
			name: "k6 summary export is handled by dedicated processor",
			data: `{"metrics":{"http_req_duration":{"avg":24.8,"p(95)":29.7},"http_reqs":{"rate":39.5,"count":1187}}}`,
			want: false,
		},
		{
			name: "artillery report",
			data: `{"aggregate":{"counters":{"http.requests":10}}}`,
			want: true,
		},
		{
			name: "playwright json is not in this processor scope",
			data: `{"suites":[{"specs":[{"tests":[{"results":[{"status":"passed"}]}]}]}]}`,
			want: false,
		},
		{
			name: "cypress json is not in this processor scope",
			data: `{"stats":{"tests":1},"tests":[{"title":"works","state":"passed"}]}`,
			want: false,
		},
		{
			name: "plain json",
			data: `{"hello":"world"}`,
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isArtilleryReport([]byte(tc.data)); got != tc.want {
				t.Fatalf("expected %t, got %t", tc.want, got)
			}
		})
	}
}
