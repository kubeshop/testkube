package artifacts

import (
	"io/fs"
	"strings"
	"testing"

	gomock "go.uber.org/mock/gomock"

	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/filesystem"
)

func TestIsJMeterStatisticsReport(t *testing.T) {
	tests := []struct {
		name string
		data string
		want bool
	}{
		{
			name: "jmeter statistics report",
			data: `{"Total":{"transaction":"Total","sampleCount":255,"errorCount":3,"errorPct":1.17,"meanResTime":235.4,"pct1ResTime":337,"pct2ResTime":339.2,"pct3ResTime":353,"throughput":4.23}}`,
			want: true,
		},
		{
			name: "jmeter statistics report with transaction entries",
			data: `{"GET /api":{"transaction":"GET /api","sampleCount":10,"errorCount":0},"Total":{"transaction":"Total","sampleCount":10,"errorCount":0,"throughput":2.1}}`,
			want: true,
		},
		{
			name: "k6 summary is handled by dedicated processor",
			data: `{"metrics":{"http_req_duration":{"values":{"p(95)":123}}}}`,
			want: false,
		},
		{
			name: "artillery report is handled by dedicated processor",
			data: `{"aggregate":{"counters":{"http.requests":10}}}`,
			want: false,
		},
		{
			name: "statistics json without total transaction",
			data: `{"Total":{"sampleCount":255,"errorCount":3}}`,
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
			if got := isJMeterStatisticsReport([]byte(tc.data)); got != tc.want {
				t.Fatalf("expected %t, got %t", tc.want, got)
			}
		})
	}
}

func TestHasJMeterStatisticsShape(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		maxBytes int64
		want     bool
	}{
		{
			name:     "jmeter statistics with total not first",
			data:     `{"GET /api":{"transaction":"GET /api","sampleCount":10},"Total":{"transaction":"Total","sampleCount":10,"errorCount":0,"throughput":2.1}}`,
			maxBytes: maxJMeterStatisticsProbeBytes,
			want:     true,
		},
		{
			name:     "statistics shaped json without numeric metrics",
			data:     `{"Total":{"transaction":"Total","sampleCount":"10","errorCount":"0"}}`,
			maxBytes: maxJMeterStatisticsProbeBytes,
			want:     false,
		},
		{
			name:     "bounded before total",
			data:     `{"metadata":{"message":"` + strings.Repeat("a", 128) + `"},"Total":{"transaction":"Total","sampleCount":10,"errorCount":0}}`,
			maxBytes: 64,
			want:     false,
		},
		{
			name:     "plain json",
			data:     `{"hello":"world"}`,
			maxBytes: maxJMeterStatisticsProbeBytes,
			want:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasJMeterStatisticsShape(strings.NewReader(tc.data), tc.maxBytes); got != tc.want {
				t.Fatalf("expected %t, got %t", tc.want, got)
			}
		})
	}
}

func TestJMeterStatisticsPostProcessorAddUploadsReport(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	report := []byte(`{"Total":{"transaction":"Total","sampleCount":10,"errorCount":0,"throughput":2.1}}`)
	mockFS := filesystem.NewMockFileSystem(mockCtrl)
	mockFS.EXPECT().
		OpenFileRO("/report/statistics.json").
		Times(2).
		DoAndReturn(func(string) (fs.File, error) {
			return filesystem.NewMockFile("statistics.json", report), nil
		})

	mockClient := controlplaneclient.NewMockClient(mockCtrl)
	mockClient.EXPECT().
		AppendExecutionReport(gomock.Any(), "env123", "exec123", "workflow123", "step123", "report/statistics.json", report).
		Return(nil)

	pp := NewJMeterStatisticsPostProcessor(mockFS, mockClient, "env123", "exec123", "workflow123", "step123", "/", "")
	if err := pp.add("report/statistics.json"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestJMeterStatisticsPostProcessorAddSkipsOtherJSONWithoutFullRead(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockFS := filesystem.NewMockFileSystem(mockCtrl)
	mockFS.EXPECT().
		OpenFileRO("/summary.json").
		Return(newGuardedSingleReadFile("summary.json", []byte(`{"Total":{"transaction":"Total","sampleCount":10,"errorCount":0}}`)), nil)

	mockClient := controlplaneclient.NewMockClient(mockCtrl)
	pp := NewJMeterStatisticsPostProcessor(mockFS, mockClient, "env123", "exec123", "workflow123", "step123", "/", "")
	if err := pp.add("summary.json"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestJMeterStatisticsPostProcessorAddSkipsNonReportStatisticsWithoutFullRead(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockFS := filesystem.NewMockFileSystem(mockCtrl)
	mockFS.EXPECT().
		OpenFileRO("/statistics.json").
		Return(newGuardedSingleReadFile("statistics.json", []byte(`{"Total":{"transaction":"Total","sampleCount":"10","errorCount":"0"}}`)), nil)

	mockClient := controlplaneclient.NewMockClient(mockCtrl)
	pp := NewJMeterStatisticsPostProcessor(mockFS, mockClient, "env123", "exec123", "workflow123", "step123", "/", "")
	if err := pp.add("statistics.json"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
