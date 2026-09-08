package artifacts

import (
	"context"
	"io"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/kubeshop/testkube/pkg/controlplaneclient"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/common/testdata"
	"github.com/kubeshop/testkube/pkg/filesystem"
	"github.com/kubeshop/testkube/pkg/testresults"
)

func TestJUnitPostProcessor_Add(t *testing.T) {
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
					AppendExecutionReport(gomock.Any(), "env123", "exec123", "workflow123", "step123", "report/junit.xml", []byte(testdata.BasicJUnit), gomock.Not(gomock.Nil())).
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
		AppendExecutionReport(gomock.Any(), "env123", "exec123", "workflow123", "step123", filepath.Join(pathPrefix, filePath), []byte(junitContent), gomock.Not(gomock.Nil())).
		Return(nil)

	err := pp.Add(filePath)

	assert.NoError(t, err)
}

func TestIsXMLFile(t *testing.T) {
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
			ok := testresults.Sniff(data)
			assert.Equal(t, tc.want, ok)
		})
	}
}

// The whole point of the handoff: the uploaded report has to say which failures
// the step declared acceptable, and only the container that decided the verdict
// knows. Joined on the report file, because the artifacts stage carries a step
// reference of its own that would never match the stage that ran the tests.
func TestJUnitPostProcessor_AppliesTheVerdictRecordedForTheReport(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	t.Setenv("TESTKUBE_TW_INTERNAL_PATH", t.TempDir())

	root := t.TempDir()
	reportPath := filepath.Join(root, "junit.xml")
	junit := []byte(`<testsuite name="s" tests="2">
  <testcase name="known" classname="c"><failure message="expected"/></testcase>
  <testcase name="real" classname="c"><failure message="not expected"/></testcase>
</testsuite>`)

	require.NoError(t, testresults.WriteVerdict("rrun1", testresults.ReportVerdict{
		Reports:   []string{reportPath},
		Muted:     []string{"s/c/known"},
		Tolerated: true,
	}))

	mockFS := filesystem.NewMockFileSystem(mockCtrl)
	mockFS.EXPECT().OpenFileRO(reportPath).Return(filesystem.NewMockFile("junit.xml", junit), nil)
	mockClient := controlplaneclient.NewMockClient(mockCtrl)

	var got *testresults.Digest
	mockClient.EXPECT().
		AppendExecutionReport(gomock.Any(), "env123", "exec123", "workflow123", "artifacts-step", "junit.xml", junit, gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _, _, _, _ string, _ []byte, digest *testresults.Digest) error {
			got = digest
			return nil
		})

	// The step reference here is the artifacts stage's, deliberately not the one
	// the verdict was written under.
	pp := NewJUnitPostProcessor(mockFS, mockClient, "env123", "exec123", "workflow123", "artifacts-step", root, "")
	require.NoError(t, pp.Start())
	require.NoError(t, pp.Add("junit.xml"))

	require.NotNil(t, got)
	assert.True(t, got.VerdictApplied)
	assert.Equal(t, int32(1), got.Muted)
	assert.Equal(t, int32(1), got.Unexpected)
	assert.True(t, got.Tolerated)

	byID := map[string]bool{}
	for _, failure := range got.Failures {
		byID[failure.Id] = failure.Muted
	}
	assert.True(t, byID["s/c/known"])
	assert.False(t, byID["s/c/real"])
}

// A step with no testCases policy reaches no verdict, which is almost every step
// there is. Its report still uploads, and says nothing it cannot know.
func TestJUnitPostProcessor_LeavesTheVerdictUnsetWhenNoneWasRecorded(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	t.Setenv("TESTKUBE_TW_INTERNAL_PATH", t.TempDir())

	root := t.TempDir()
	junit := []byte(`<testsuite name="s" tests="1">
  <testcase name="real" classname="c"><failure message="not expected"/></testcase>
</testsuite>`)

	mockFS := filesystem.NewMockFileSystem(mockCtrl)
	mockFS.EXPECT().OpenFileRO(filepath.Join(root, "junit.xml")).Return(filesystem.NewMockFile("junit.xml", junit), nil)
	mockClient := controlplaneclient.NewMockClient(mockCtrl)

	var got *testresults.Digest
	mockClient.EXPECT().
		AppendExecutionReport(gomock.Any(), "env123", "exec123", "workflow123", "step123", "junit.xml", junit, gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _, _, _, _ string, _ []byte, digest *testresults.Digest) error {
			got = digest
			return nil
		})

	pp := NewJUnitPostProcessor(mockFS, mockClient, "env123", "exec123", "workflow123", "step123", root, "")
	require.NoError(t, pp.Start())
	require.NoError(t, pp.Add("junit.xml"))

	require.NotNil(t, got)
	assert.False(t, got.VerdictApplied, "no verdict was recorded, so none may be claimed")
	assert.Zero(t, got.Muted)
	require.Len(t, got.Failures, 1)
	assert.False(t, got.Failures[0].Muted)
}
