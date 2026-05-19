package artifacts

import (
	"io/fs"
	"strings"
	"testing"

	gomock "go.uber.org/mock/gomock"

	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/filesystem"
)

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

func TestHasArtilleryReportShape(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		maxBytes int64
		want     bool
	}{
		{
			name:     "artillery report with metadata before aggregate",
			data:     `{"intermediate":{"latencies":[1,2,3]},"aggregate":{"counters":{"http.requests":10}}}`,
			maxBytes: maxArtilleryReportProbeBytes,
			want:     true,
		},
		{
			name:     "artillery report with summaries",
			data:     `{"aggregate":{"summaries":{"http.response_time":{"p95":123.4}}}}`,
			maxBytes: maxArtilleryReportProbeBytes,
			want:     true,
		},
		{
			name:     "k6 summary export",
			data:     `{"metrics":{"http_req_duration":{"avg":24.8,"p(95)":29.7},"http_reqs":{"rate":39.5,"count":1187}}}`,
			maxBytes: maxArtilleryReportProbeBytes,
			want:     false,
		},
		{
			name:     "bounded before aggregate",
			data:     `{"intermediate":{"message":"` + strings.Repeat("a", 128) + `"},"aggregate":{"counters":{"http.requests":10}}}`,
			maxBytes: 64,
			want:     false,
		},
		{
			name:     "plain json",
			data:     `{"hello":"world"}`,
			maxBytes: maxArtilleryReportProbeBytes,
			want:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasArtilleryReportShape(strings.NewReader(tc.data), tc.maxBytes); got != tc.want {
				t.Fatalf("expected %t, got %t", tc.want, got)
			}
		})
	}
}

func TestArtilleryReportPostProcessorAddUploadsReport(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	report := []byte(`{"aggregate":{"counters":{"http.requests":10}}}`)
	mockFS := filesystem.NewMockFileSystem(mockCtrl)
	mockFS.EXPECT().
		OpenFileRO("/artillery-report.json").
		Times(2).
		DoAndReturn(func(string) (fs.File, error) {
			return filesystem.NewMockFile("artillery-report.json", report), nil
		})

	mockClient := controlplaneclient.NewMockClient(mockCtrl)
	mockClient.EXPECT().
		AppendExecutionReport(gomock.Any(), "env123", "exec123", "workflow123", "step123", "artillery-report.json", report).
		Return(nil)

	pp := NewArtilleryReportPostProcessor(mockFS, mockClient, "env123", "exec123", "workflow123", "step123", "/", "")
	if err := pp.add("artillery-report.json"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestArtilleryReportPostProcessorAddSkipsNonReportJSONWithoutFullRead(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockFS := filesystem.NewMockFileSystem(mockCtrl)
	mockFS.EXPECT().
		OpenFileRO("/browser-trace.json").
		Return(newGuardedSingleReadFile("browser-trace.json", []byte(`{"type":"trace","events":[{"name":"navigation"}]}`)), nil)

	mockClient := controlplaneclient.NewMockClient(mockCtrl)
	pp := NewArtilleryReportPostProcessor(mockFS, mockClient, "env123", "exec123", "workflow123", "step123", "/", "")
	if err := pp.add("browser-trace.json"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
