package artifacts

import "testing"

func TestIsK6SummaryReport(t *testing.T) {
	tests := []struct {
		name string
		data string
		want bool
	}{
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
			name: "plain json",
			data: `{"hello":"world"}`,
			want: false,
		},
		{
			name: "k6 summary without numeric values",
			data: `{"metrics":{"http_req_duration":{"type":"trend","contains":"time","values":{}}}}`,
			want: false,
		},
		{
			name: "k6 summary null value",
			data: `{"metrics":{"http_req_duration":{"avg":null}}}`,
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isK6SummaryReport([]byte(tc.data)); got != tc.want {
				t.Fatalf("expected %t, got %t", tc.want, got)
			}
		})
	}
}
