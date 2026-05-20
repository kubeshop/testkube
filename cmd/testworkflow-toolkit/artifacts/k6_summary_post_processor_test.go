package artifacts

import (
	"errors"
	"io/fs"
	"strings"
	"testing"

	gomock "go.uber.org/mock/gomock"

	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/filesystem"
)

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

func TestHasK6SummaryReportShape(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		maxBytes int64
		want     bool
	}{
		{
			name:     "k6 summary with root group before metrics",
			data:     `{"root_group":{"name":"","groups":[]},"metrics":{"http_req_duration":{"type":"trend","contains":"time","values":{"p(95)":123}}}}`,
			maxBytes: maxK6SummaryProbeBytes,
			want:     true,
		},
		{
			name:     "k6 summary export",
			data:     `{"metrics":{"http_req_duration":{"avg":24.8,"p(95)":29.7},"http_reqs":{"rate":39.5,"count":1187}}}`,
			maxBytes: maxK6SummaryProbeBytes,
			want:     true,
		},
		{
			name:     "k6 summary without numeric values",
			data:     `{"metrics":{"http_req_duration":{"type":"trend","contains":"time","values":{},"status":"ok"}}}`,
			maxBytes: maxK6SummaryProbeBytes,
			want:     false,
		},
		{
			name:     "k6 json output stream",
			data:     `{"type":"Point","data":{"time":"2026-01-01T00:00:00Z","value":1,"metric":"http_reqs"}}` + "\n" + `{"type":"Point","data":{"value":2}}`,
			maxBytes: maxK6SummaryProbeBytes,
			want:     false,
		},
		{
			name:     "bounded before metrics",
			data:     `{"root_group":{"name":"` + strings.Repeat("a", 128) + `"},"metrics":{"http_req_duration":{"avg":24.8}}}`,
			maxBytes: 64,
			want:     false,
		},
		{
			name:     "plain json",
			data:     `{"hello":"world"}`,
			maxBytes: maxK6SummaryProbeBytes,
			want:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasK6SummaryReportShape(strings.NewReader(tc.data), tc.maxBytes); got != tc.want {
				t.Fatalf("expected %t, got %t", tc.want, got)
			}
		})
	}
}

func TestK6SummaryPostProcessorAddUploadsSummary(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	summary := []byte(`{"metrics":{"http_req_duration":{"avg":24.8,"p(95)":29.7}}}`)
	mockFS := filesystem.NewMockFileSystem(mockCtrl)
	mockFS.EXPECT().
		OpenFileRO("/summary.json").
		Times(2).
		DoAndReturn(func(string) (fs.File, error) {
			return filesystem.NewMockFile("summary.json", summary), nil
		})

	mockClient := controlplaneclient.NewMockClient(mockCtrl)
	mockClient.EXPECT().
		AppendExecutionReport(gomock.Any(), "env123", "exec123", "workflow123", "step123", "summary.json", summary).
		Return(nil)

	pp := NewK6SummaryPostProcessor(mockFS, mockClient, "env123", "exec123", "workflow123", "step123", "/", "")
	if err := pp.add("summary.json"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestK6SummaryPostProcessorAddSkipsNonSummaryJSONWithoutFullRead(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockFS := filesystem.NewMockFileSystem(mockCtrl)
	mockFS.EXPECT().
		OpenFileRO("/k6-browser-result.json").
		Return(newGuardedSingleReadFile("k6-browser-result.json", []byte(`{"type":"Point","data":{"metric":"http_reqs","value":1}}`)), nil)

	mockClient := controlplaneclient.NewMockClient(mockCtrl)
	pp := NewK6SummaryPostProcessor(mockFS, mockClient, "env123", "exec123", "workflow123", "step123", "/", "")
	if err := pp.add("k6-browser-result.json"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

var errUnexpectedFullRead = errors.New("unexpected full read")

type guardedSingleReadFile struct {
	name string
	data []byte
	read bool
}

func newGuardedSingleReadFile(name string, data []byte) *guardedSingleReadFile {
	return &guardedSingleReadFile{name: name, data: data}
}

func (f *guardedSingleReadFile) Stat() (fs.FileInfo, error) {
	return &filesystem.MockFileInfo{
		FName: f.name,
		FSize: int64(len(f.data)) + 1<<30,
	}, nil
}

func (f *guardedSingleReadFile) Read(p []byte) (int, error) {
	if f.read {
		return 0, errUnexpectedFullRead
	}
	f.read = true
	return copy(p, f.data), nil
}

func (f *guardedSingleReadFile) Close() error {
	return nil
}
