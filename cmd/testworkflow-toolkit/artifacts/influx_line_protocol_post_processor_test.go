package artifacts

import (
	"io/fs"
	"strings"
	"testing"

	gomock "go.uber.org/mock/gomock"

	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/filesystem"
)

func TestIsInfluxLineProtocolReport(t *testing.T) {
	tests := []struct {
		name string
		data string
		want bool
	}{
		{
			name: "valid line protocol with timestamp",
			data: "cpu,host=server01 usage_user=0.10,usage_system=0.20 1670000000000000000\n",
			want: true,
		},
		{
			name: "valid line protocol without timestamp",
			data: "cpu usage=50.0\n",
			want: true,
		},
		{
			name: "valid line protocol with metadata comment",
			data: "#META workflow=wf step.ref=s1 format=influx\n" +
				"cpu,host=server01 usage_user=0.10\n",
			want: true,
		},
		{
			name: "plain text",
			data: "hello world\n",
			want: false,
		},
		{
			name: "json file",
			data: `{"metrics":{"http_req_duration":{"avg":24.8}}}`,
			want: false,
		},
		{
			name: "comments only",
			data: "# This is a comment\n#META format=influx\n",
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isInfluxLineProtocolReport([]byte(tc.data)); got != tc.want {
				t.Fatalf("expected %t, got %t", tc.want, got)
			}
		})
	}
}

func TestHasInfluxLineProtocolShape(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		maxBytes int64
		want     bool
	}{
		{
			name:     "valid line after metadata",
			data:     "#META workflow=wf format=influx\n" + strings.Repeat("x", 128) + "\ncpu usage=50.0\n",
			maxBytes: maxInfluxLineProtocolProbeBytes,
			want:     true,
		},
		{
			name:     "valid line within probe limit",
			data:     "cpu,host=server01 usage_user=0.10\n",
			maxBytes: maxInfluxLineProtocolProbeBytes,
			want:     true,
		},
		{
			name:     "bounded before valid line",
			data:     strings.Repeat("# comment\n", 20) + "cpu usage=50.0\n",
			maxBytes: 32,
			want:     false,
		},
		{
			name:     "plain text",
			data:     "not line protocol\n",
			maxBytes: maxInfluxLineProtocolProbeBytes,
			want:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasInfluxLineProtocolShape(strings.NewReader(tc.data), tc.maxBytes); got != tc.want {
				t.Fatalf("expected %t, got %t", tc.want, got)
			}
		})
	}
}

func TestInfluxLineProtocolPostProcessorAddUploadsReport(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	report := []byte("cpu,host=server01 usage_user=0.10,usage_system=0.20 1670000000000000000\n")
	mockFS := filesystem.NewMockFileSystem(mockCtrl)
	mockFS.EXPECT().
		OpenFileRO("/metrics.influx").
		Times(2).
		DoAndReturn(func(string) (fs.File, error) {
			return filesystem.NewMockFile("metrics.influx", report), nil
		})

	mockClient := controlplaneclient.NewMockClient(mockCtrl)
	mockClient.EXPECT().
		AppendExecutionReport(gomock.Any(), "env123", "exec123", "workflow123", "step123", "metrics.influx", report).
		Return(nil)

	pp := NewInfluxLineProtocolPostProcessor(mockFS, mockClient, "env123", "exec123", "workflow123", "step123", "/", "")
	if err := pp.add("metrics.influx"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestInfluxLineProtocolPostProcessorAddSkipsNonReportWithoutFullRead(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockFS := filesystem.NewMockFileSystem(mockCtrl)
	mockFS.EXPECT().
		OpenFileRO("/notes.influx").
		Return(newGuardedSingleReadFile("notes.influx", []byte("not valid line protocol\n")), nil)

	mockClient := controlplaneclient.NewMockClient(mockCtrl)
	pp := NewInfluxLineProtocolPostProcessor(mockFS, mockClient, "env123", "exec123", "workflow123", "step123", "/", "")
	if err := pp.add("notes.influx"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestInfluxLineProtocolPostProcessorAddSkipsWrongExtension(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockFS := filesystem.NewMockFileSystem(mockCtrl)
	mockFS.EXPECT().
		OpenFileRO("/metrics.json").
		Return(filesystem.NewMockFile("metrics.json", []byte("cpu usage=50.0\n")), nil)

	mockClient := controlplaneclient.NewMockClient(mockCtrl)
	pp := NewInfluxLineProtocolPostProcessor(mockFS, mockClient, "env123", "exec123", "workflow123", "step123", "/", "")
	if err := pp.add("metrics.json"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
