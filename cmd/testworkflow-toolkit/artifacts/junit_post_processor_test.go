package artifacts

import (
	"io"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/controlplaneclient"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/common/testdata"
	"github.com/kubeshop/testkube/pkg/filesystem"
)

func TestJUnitPostProcessor_Add(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tests := []struct {
		name  string
		setup func(*controlplaneclient.MockClient)
		path  string
		file  fs.File
		want  error
	}{
		{
			name: "is not xml file",
			path: "report/test.log",
			file: filesystem.NewMockFile("test.log", []byte("some random file")),
			want: nil,
		},
		{
			name: "is not junit report",
			path: "report/junit.xml",
			file: filesystem.NewMockFile("junit.xml", []byte(testdata.InvalidJUnit)),
			want: nil,
		},
		{
			name: "valid junit report",
			setup: func(client *controlplaneclient.MockClient) {
				client.EXPECT().
					AppendExecutionReport(gomock.Any(), "env123", "exec123", "workflow123", "step123", "report/junit.xml", []byte(testdata.BasicJUnit)).
					Return(nil)
			},
			path: "report/junit.xml",
			file: filesystem.NewMockFile("basic.xml", []byte(testdata.BasicJUnit)),
			want: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockFS := filesystem.NewMockFileSystem(mockCtrl)
			mockFS.EXPECT().OpenFileRO("/"+tc.path).Return(tc.file, nil)
			mockClient := controlplaneclient.NewMockClient(mockCtrl)
			if tc.setup != nil {
				tc.setup(mockClient)
			}
			pp := NewJUnitPostProcessor(mockFS, mockClient, "env123", "exec123", "workflow123", "step123", "/", "")
			err := pp.Add(tc.path)
			assert.Equal(t, tc.want, err)
		})
	}

}

func TestJUnitPostProcessor_Add_WithPathPrefix(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockFS := filesystem.NewMockFileSystem(mockCtrl)
	mockClient := controlplaneclient.NewMockClient(mockCtrl)

	pathPrefix := "prefixed/junit/report/"
	filePath := "junit.xml"
	junitContent := []byte(testdata.BasicJUnit)

	mockFS.EXPECT().OpenFileRO(gomock.Any()).Return(filesystem.NewMockFile("junit.xml", junitContent), nil)

	pp := NewJUnitPostProcessor(mockFS, mockClient, "env123", "exec123", "workflow123", "step123", "/test_root", pathPrefix)

	mockClient.EXPECT().
		AppendExecutionReport(gomock.Any(), "env123", "exec123", "workflow123", "step123", filepath.Join(pathPrefix, filePath), []byte(junitContent)).
		Return(nil)

	err := pp.Add(filePath)

	assert.NoError(t, err)
}

func TestIsXMLFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		stat *filesystem.MockFileInfo
		want bool
	}{
		{
			name: "is dir",
			stat: &filesystem.MockFileInfo{
				FName:  "some-dir",
				FIsDir: true,
			},
			want: false,
		},
		{
			name: "is random file",
			stat: &filesystem.MockFileInfo{
				FName: "file.test",
				FSize: 0,
			},
			want: false,
		},
		{
			name: "is empty xml file",
			stat: &filesystem.MockFileInfo{
				FName: "file.xml",
				FSize: 0,
			},
			want: false,
		},
		{
			name: "is non-empty xml file",
			stat: &filesystem.MockFileInfo{
				FName: "file.xml",
				FSize: 256,
			},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isXMLFile(tc.stat))
		})
	}
}

func TestIsJUnitReport(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tests := []struct {
		name string
		file fs.File
		want bool
	}{
		{
			name: "basic junit",
			file: filesystem.NewMockFile("basic.xml", []byte(testdata.BasicJUnit)),
			want: true,
		},
		{
			name: "complete junit",
			file: filesystem.NewMockFile("complete.xml", []byte(testdata.CompleteJUnit)),
			want: true,
		},
		{
			name: "invalid junit",
			file: filesystem.NewMockFile("invalid.xml", []byte(testdata.InvalidJUnit)),
			want: false,
		},
		{
			name: "one-line junit",
			file: filesystem.NewMockFile("oneline.xml", []byte(testdata.OneLineJUnit)),
			want: true,
		},
		{
			name: "testsuites only junit",
			file: filesystem.NewMockFile("testsuites.xml", []byte(testdata.TestsuitesOnlyJUnit)),
			want: true,
		},
		{
			name: "testsuite only junit",
			file: filesystem.NewMockFile("testsuite.xml", []byte(testdata.TestsuiteOnlyJUnit)),
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := io.ReadAll(tc.file)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			ok := isJUnitReport(data)
			assert.Equal(t, tc.want, ok)
		})
	}
}
