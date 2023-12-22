package runner

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/content"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/filesystem"
	"github.com/kubeshop/testkube/pkg/ui"
)

const (
	envVarPrefix = "$"
)

func getTestPathAndWorkingDir(fs filesystem.FileSystem, execution *testkube.Execution, dataDir string) (testPath string, workingDir, testFile string, err error) {
	testPath, workingDir, err = content.GetPathAndWorkingDir(execution.Content, dataDir)
	if err != nil {
		output.PrintLogf("%s Failed to resolve absolute directory for %s, using the path directly", ui.IconWarning, dataDir)
	}

	fileInfo, err := fs.Stat(testPath)
	if err != nil {
		return "", "", "", err
	}

	if fileInfo.IsDir() {
		testFile, err = findTestFile(fs, execution, testPath, jmxExtension)
		if err != nil {
			return "", "", "", errors.Wrapf(err, "error searching for %s file in test path %s", jmxExtension, testPath)
		}

		// sanity checking for test script
		testPath = filepath.Join(testPath, testFile)
		fileInfo, err := fs.Stat(testPath)
		if err != nil || fileInfo.IsDir() {
			output.PrintLogf("%s Could not find file %s in the directory, error: %s", ui.IconCross, testFile, err)
			return "", "", "", errors.Wrapf(err, "could not find file %s in the directory", testFile)

		}
	}
	return
}

// findTestFile tries to find test file in args or in testPath directory.
func findTestFile(fs filesystem.FileSystem, execution *testkube.Execution, testPath, testExtension string) (testFile string, err error) {
	if len(execution.Args) > 0 {
		testFile = execution.Args[len(execution.Args)-1]
		if strings.HasPrefix(testFile, envVarPrefix) {
			testFile = os.ExpandEnv(testFile)
		}
		if !strings.HasSuffix(testFile, testExtension) {
			testFile = ""
		} else {
			output.PrintLogf("%s %s file provided as last argument: %s", ui.IconWorld, testExtension, testFile)
			execution.Args = execution.Args[:len(execution.Args)-1]
		}
		if testFile == "" {
			testFile, err = searchInDir(fs, testPath, testExtension)
			if err != nil {
				return "", errors.Wrapf(err, "error searching for %s file in test path %s", testExtension, testPath)
			}
			if testFile != "" {
				output.PrintLogf("%s %s file found in test path: %s", ui.IconWorld, testExtension, testFile)
			}
		}
	}
	if testFile == "" {
		output.PrintLogf("%s %s file not found in args or test path!", ui.IconCross, testExtension)
		return "", errors.Errorf("no %s file found", testExtension)
	}
	return testFile, nil
}

// searchInDir searches for file with given extension in given directory.
func searchInDir(fs filesystem.FileSystem, dir, extension string) (string, error) {
	files, err := fs.ReadDir(dir)
	if err != nil {
		return "", err
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), "."+extension) {
			return file.Name(), nil
		}
	}
	return "", nil
}

// injectAndExpandEnvVars injects variables from params into args and expands them if args contains <envVars> placeholder.
// Returns new args with injected and expanded variables.
func injectAndExpandEnvVars(args []string, params []string) []string {
	copied := make([]string, len(args))
	copy(copied, args)
	for i := range copied {
		if copied[i] == "<envVars>" {
			newArgs := make([]string, len(copied)+len(params)-1)
			copy(newArgs, copied[:i])
			copy(newArgs[i:], params)
			copy(newArgs[i+len(params):], copied[i+1:])
			copied = newArgs
			break
		}
	}

	for i := range copied {
		copied[i] = os.ExpandEnv(copied[i])
	}

	return copied
}
