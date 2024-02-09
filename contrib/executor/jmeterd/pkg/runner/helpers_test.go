package runner

import (
	"errors"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/filesystem"
)

func TestGetTestPathAndWorkingDir(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	type args struct {
		fs        *filesystem.MockFileSystem
		execution testkube.Execution
		dataDir   string
	}
	tests := []struct {
		name           string
		args           args
		wantTestPath   string
		wantWorkingDir string
		wantTestFile   string
		wantErr        bool
		setup          func(fs *filesystem.MockFileSystem)
	}{
		{
			name: "Get test path and working dir for file URI",
			args: args{
				fs: filesystem.NewMockFileSystem(mockCtrl),
				execution: testkube.Execution{
					Content: &testkube.TestContent{Type_: string(testkube.TestContentTypeFileURI)},
					Args:    []string{"arg1"},
				},
				dataDir: "/tmp/data",
			},
			wantTestPath:   "/tmp/data/test-content",
			wantWorkingDir: "/tmp/data",
			wantTestFile:   "",
			wantErr:        false,
			setup: func(fs *filesystem.MockFileSystem) {
				fs.EXPECT().Stat("/tmp/data/test-content").Return(&filesystem.MockFileInfo{FIsDir: false}, nil)
			},
		},
		{
			name: "Get test path and working dir for git dir (no repository)",
			args: args{
				fs: filesystem.NewMockFileSystem(mockCtrl),
				execution: testkube.Execution{
					Content: &testkube.TestContent{Type_: string(testkube.TestContentTypeGitFile)},
					Args:    []string{"arg1", "test.jmx"},
				},
				dataDir: "/tmp/data",
			},
			wantTestPath:   "/tmp/data/repo/test.jmx",
			wantWorkingDir: "/tmp/data",
			wantTestFile:   "test.jmx",
			wantErr:        false,
			setup: func(fs *filesystem.MockFileSystem) {
				gomock.InOrder(
					fs.EXPECT().Stat("/tmp/data/repo").Return(&filesystem.MockFileInfo{FIsDir: true}, nil),
					fs.EXPECT().Stat("/tmp/data/repo/test.jmx").Return(&filesystem.MockFileInfo{FIsDir: false}, nil),
				)

			},
		},
		{
			name: "Get test path and working dir for git dir (repository)",
			args: args{
				fs: filesystem.NewMockFileSystem(mockCtrl),
				execution: testkube.Execution{
					Content: &testkube.TestContent{
						Type_: string(testkube.TestContentTypeGitFile),
						Repository: &testkube.Repository{
							WorkingDir: "tests",
							Path:       "tests/test1",
						},
					},
					Args: []string{"arg1", "test.jmx"},
				},
				dataDir: "/tmp/data",
			},
			wantTestPath:   "/tmp/data/repo/tests/test1/test.jmx",
			wantWorkingDir: "/tmp/data/repo/tests",
			wantTestFile:   "test.jmx",
			wantErr:        false,
			setup: func(fs *filesystem.MockFileSystem) {
				gomock.InOrder(
					fs.EXPECT().Stat("/tmp/data/repo/tests/test1").Return(&filesystem.MockFileInfo{FIsDir: true}, nil),
					fs.EXPECT().Stat("/tmp/data/repo/tests/test1/test.jmx").Return(&filesystem.MockFileInfo{FIsDir: false}, nil),
				)
			},
		},
		{
			name: "Error on fs.Stat",
			args: args{
				fs:        filesystem.NewMockFileSystem(mockCtrl),
				execution: testkube.Execution{
					// Set appropriate fields
				},
				dataDir: "data/dir",
			},
			wantTestPath:   "",
			wantWorkingDir: "",
			wantTestFile:   "",
			wantErr:        true,
			setup: func(fs *filesystem.MockFileSystem) {
				fs.EXPECT().Stat(gomock.Any()).Return(nil, errors.New("stat error"))
			},
		},
		{
			name: "Get test path and working dir for -t absolute with working dir",
			args: args{
				fs: filesystem.NewMockFileSystem(mockCtrl),
				execution: testkube.Execution{
					Content: &testkube.TestContent{
						Type_: string(testkube.TestContentTypeGitFile),
						Repository: &testkube.Repository{
							WorkingDir: "tests",
							Path:       "tests/test1",
						},
					},
					Args: []string{"-t", "/tmp/data/repo/tests/test1/test.jmx"},
				},
				dataDir: "/tmp/data",
			},
			wantTestPath:   "/tmp/data/repo/tests/test1/test.jmx",
			wantWorkingDir: "/tmp/data/repo/tests",
			wantTestFile:   "test.jmx",
			wantErr:        false,
			setup: func(fs *filesystem.MockFileSystem) {
				gomock.InOrder(
					fs.EXPECT().Stat("/tmp/data/repo/tests/test1").Return(&filesystem.MockFileInfo{FIsDir: true}, nil),
					fs.EXPECT().Stat("/tmp/data/repo/tests/test1/test.jmx").Return(&filesystem.MockFileInfo{FIsDir: false}, nil),
				)
			},
		},
		{
			name: "Get test path and working dir for -t absolute without working dir",
			args: args{
				fs: filesystem.NewMockFileSystem(mockCtrl),
				execution: testkube.Execution{
					Content: &testkube.TestContent{
						Type_: string(testkube.TestContentTypeGitFile),
						Repository: &testkube.Repository{
							Path: "tests/test1",
						},
					},
					Args: []string{"-t", "/tmp/data/repo/tests/test1/test.jmx"},
				},
				dataDir: "/tmp/data",
			},
			wantTestPath:   "/tmp/data/repo/tests/test1/test.jmx",
			wantWorkingDir: "/tmp/data",
			wantTestFile:   "test.jmx",
			wantErr:        false,
			setup: func(fs *filesystem.MockFileSystem) {
				gomock.InOrder(
					fs.EXPECT().Stat("/tmp/data/repo/tests/test1").Return(&filesystem.MockFileInfo{FIsDir: true}, nil),
					fs.EXPECT().Stat("/tmp/data/repo/tests/test1/test.jmx").Return(&filesystem.MockFileInfo{FIsDir: false}, nil),
				)
			},
		},
		{
			name: "Get test path and working dir for -t relative with working dir",
			args: args{
				fs: filesystem.NewMockFileSystem(mockCtrl),
				execution: testkube.Execution{
					Content: &testkube.TestContent{
						Type_: string(testkube.TestContentTypeGitFile),
						Repository: &testkube.Repository{
							WorkingDir: "tests",
							Path:       "tests/test1",
						},
					},
					Args: []string{"-t", "test1/test.jmx"},
				},
				dataDir: "/tmp/data",
			},
			wantTestPath:   "/tmp/data/repo/tests/test1/test.jmx",
			wantWorkingDir: "/tmp/data/repo/tests",
			wantTestFile:   "test.jmx",
			wantErr:        false,
			setup: func(fs *filesystem.MockFileSystem) {
				gomock.InOrder(
					fs.EXPECT().Stat("/tmp/data/repo/tests/test1").Return(&filesystem.MockFileInfo{FIsDir: true}, nil),
					fs.EXPECT().Stat("/tmp/data/repo/tests/test1/test.jmx").Return(&filesystem.MockFileInfo{FIsDir: false}, nil),
				)
			},
		},
		{
			name: "Get test path and working dir for -t relative without working dir",
			args: args{
				fs: filesystem.NewMockFileSystem(mockCtrl),
				execution: testkube.Execution{
					Content: &testkube.TestContent{
						Type_: string(testkube.TestContentTypeGitFile),
						Repository: &testkube.Repository{
							Path: "tests/test1",
						},
					},
					Args: []string{"-t", "repo/tests/test1/test.jmx"},
				},
				dataDir: "/tmp/data",
			},
			wantTestPath:   "/tmp/data/repo/tests/test1/test.jmx",
			wantWorkingDir: "/tmp/data",
			wantTestFile:   "test.jmx",
			wantErr:        false,
			setup: func(fs *filesystem.MockFileSystem) {
				gomock.InOrder(
					fs.EXPECT().Stat("/tmp/data/repo/tests/test1").Return(&filesystem.MockFileInfo{FIsDir: true}, nil),
					fs.EXPECT().Stat("/tmp/data/repo/tests/test1/test.jmx").Return(&filesystem.MockFileInfo{FIsDir: false}, nil),
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			tt.setup(tt.args.fs)
			gotTestPath, gotWorkingDir, gotTestFile, err := getTestPathAndWorkingDir(tt.args.fs, &tt.args.execution, tt.args.dataDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("getTestPathAndWorkingDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.wantTestPath, gotTestPath)
			assert.Equal(t, tt.wantWorkingDir, gotWorkingDir)
			assert.Equal(t, tt.wantTestFile, gotTestFile)
		})
	}
}

func TestFindTestFile(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tests := []struct {
		name          string
		executionArgs []string
		wantArgs      []string
		testPath      string
		testExtension string
		mockSetup     func(fs *filesystem.MockFileSystem)
		cleanup       func()
		want          string
		wantErr       bool
	}{
		{
			name:          "Test file found in args",
			executionArgs: []string{"arg1", "arg2", "testfile.jmx"},
			wantArgs:      []string{"arg1", "arg2"},
			testPath:      "/test/path",
			testExtension: "jmx",
			mockSetup:     func(fs *filesystem.MockFileSystem) {},
			want:          "testfile.jmx",
			wantErr:       false,
		},
		{
			name:          "Test file found in args as envvar",
			executionArgs: []string{"arg1", "arg2", "${JMETER_TEST_EXPAND_VAR_ARG}"},
			wantArgs:      []string{"arg1", "arg2"},
			testPath:      "/test/path",
			testExtension: "jmx",
			mockSetup: func(fs *filesystem.MockFileSystem) {
				os.Setenv("JMETER_TEST_EXPAND_VAR_ARG", "testfile.jmx")
			},
			cleanup: func() {
				os.Unsetenv("JMETER_TEST_EXPAND_VAR_ARG")
			},
			want:    "testfile.jmx",
			wantErr: false,
		},
		{
			name:          "Test file found in directory",
			executionArgs: []string{"arg1", "arg2"},
			wantArgs:      []string{"arg1", "arg2"},
			testPath:      "/test/path",
			testExtension: "jmx",
			mockSetup: func(fs *filesystem.MockFileSystem) {
				mockFile1 := &filesystem.MockDirEntry{FIsDir: false, FName: "anotherTestfile.jmx"}
				mockFile2 := &filesystem.MockDirEntry{FIsDir: false, FName: "data.csv"}
				fs.EXPECT().ReadDir("/test/path").Return([]os.DirEntry{mockFile1, mockFile2}, nil)
			},
			want:    "anotherTestfile.jmx",
			wantErr: false,
		},
		{
			name:          "Test file not found",
			executionArgs: []string{"arg1", "arg2"},
			wantArgs:      []string{"arg1", "arg2"},
			testPath:      "/test/path",
			testExtension: "jmx",
			mockSetup: func(fs *filesystem.MockFileSystem) {
				fs.EXPECT().ReadDir("/test/path").Return(nil, errors.New("error reading directory"))
			},
			want:    "",
			wantErr: true,
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.cleanup != nil {
				t.Cleanup(tt.cleanup)
			}

			mockFS := filesystem.NewMockFileSystem(mockCtrl)
			tt.mockSetup(mockFS)
			execution := &testkube.Execution{Args: tt.executionArgs}
			got, err := findTestFile(mockFS, execution, tt.testPath, tt.testExtension)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
			assert.Equal(t, tt.wantArgs, execution.Args)
		})
	}
}

func TestInjectAndExpandEnvVars(t *testing.T) {
	t.Parallel()

	// Define test cases
	tests := []struct {
		name     string
		envvars  map[string]string
		args     []string
		params   []string
		expected []string
	}{
		{
			name:     "Placeholder Replaced and Env Var Expanded",
			args:     []string{"before", "<envVars>", "after"},
			params:   []string{"injected1", "${JMETER_TEST_EXPAND_VAR}"},
			expected: []string{"before", "injected1", "expanded_value", "after"},
			envvars:  map[string]string{"JMETER_TEST_EXPAND_VAR": "expanded_value"},
		},
		{
			name:     "No Placeholder",
			args:     []string{"start", "end"},
			params:   []string{"middle"},
			expected: []string{"start", "end"},
		},
		{
			name:     "Placeholder with No Params",
			args:     []string{"start", "<envVars>", "end"},
			params:   []string{},
			expected: []string{"start", "end"},
		},
		{
			name:     "Empty Args",
			args:     []string{},
			params:   []string{"middle"},
			expected: []string{},
		},
	}

	// Run the test cases
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			for key, value := range tt.envvars {
				assert.NoError(t, os.Setenv(key, value))
				t.Cleanup(func() {
					assert.NoError(t, os.Unsetenv(key))
				})
			}

			got := injectAndExpandEnvVars(tt.args, tt.params)
			assert.ElementsMatch(t, tt.expected, got)
		})
	}
}

func TestGetParamValue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		args     []string
		param    string
		expected string
		wantErr  error
	}{
		{name: "get last param successfully", args: []string{"-n", "-o", "/data", "-t", "/data/repo"}, param: "-t", expected: "/data/repo", wantErr: nil},
		{name: "get middle param successfully", args: []string{"-n", "-o", "/data", "-t", "/data/repo"}, param: "-o", expected: "/data", wantErr: nil},
		{name: "param missing value returns error", args: []string{"-n", "-o", "/data", "-t"}, param: "-t", expected: "", wantErr: ErrParamMissingValue},
		{name: "param missing", args: []string{"-n", "-o", "/data", "-t", "/data/repo"}, param: "-x", expected: "", wantErr: ErrMissingParam},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			value, err := getParamValue(tc.args, tc.param)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Empty(t, value)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, value)
			}
		})
	}
}
