package agent

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
)

func PreRun(ctx context.Context) {
	params, err := envs.LoadTestkubeVariables()
	if err != nil {
		output.PrintError(os.Stderr, errors.Wrap(err, "error loading env vars"))
		os.Exit(1)
	}

	if params.IstioProxyWait {
		output.PrintEvent("waiting for istio's proxy to become ready")
		for {
			// TODO(emil): should this be head or get?
			resp, err := http.Head("http://localhost:15021/healthz/ready")
			if err == nil && resp.StatusCode == http.StatusOK {
				break
			}
			output.PrintEvent("still waiting for istio's proxy to become ready")
			time.Sleep(3 * time.Second)
		}
	}
}

func PostRun(ctx context.Context) {
	params, err := envs.LoadTestkubeVariables()
	if err != nil {
		output.PrintError(os.Stderr, errors.Wrap(err, "error loading env vars"))
		os.Exit(1)
	}

	if params.IstioProxyExit {
		output.PrintEvent("sending exit signal to istio's proxy")
		for {
			resp, err := http.Post("http://localhost:15020/quitquitquit", "", nil)
			if err == nil && resp.StatusCode == http.StatusOK {
				break
			}
			output.PrintEvent("still sending exit signal to istio's proxy")
			time.Sleep(3 * time.Second)
		}
	}
}

// Run starts test runner, test runner can have 3 states
// - pod:success, test execution: success
// - pod:success, test execution: failed
// - pod:failed,  test execution: failed - this one is unusual behaviour
func Run(ctx context.Context, r runner.Runner, args []string) {

	var test []byte
	var err error

	// Validation
	stat, _ := os.Stdin.Stat()
	switch {
	case (stat.Mode() & os.ModeCharDevice) == 0:
		test, err = io.ReadAll(os.Stdin)
		if err != nil {
			output.PrintError(os.Stderr, errors.Errorf("can't read stdin input: %v", err))
			os.Exit(1)
		}
	case len(args) > 1:
		test = []byte(args[1])
		hasFileFlag := args[1] == "-f" || args[1] == "--file"
		if hasFileFlag {
			test, err = os.ReadFile(args[2])
			if err != nil {
				output.PrintError(os.Stderr, errors.Errorf("error reading JSON file: %v", err))
				os.Exit(1)
			}
		}
	default:
		output.PrintError(os.Stderr, errors.Errorf("execution json must be provided using stdin, program argument or -f|--file flag"))
		os.Exit(1)
	}

	e := testkube.Execution{}

	err = json.Unmarshal(test, &e)
	if err != nil {
		output.PrintError(os.Stderr, errors.Wrap(err, "error unmarshalling execution json"))
		os.Exit(1)
	}

	params, err := envs.LoadTestkubeVariables()
	if err != nil {
		output.PrintError(os.Stderr, errors.Wrap(err, "error loading env vars"))
		os.Exit(1)
	}

	if r.GetType().IsMain() {
		envs.PrintParams(params)
	}

	if r.GetType().IsMain() && e.PreRunScript != "" {
		output.PrintEvent("running prerun script", e.Id)

		if serr := RunScript(e.PreRunScript, params.WorkingDir); serr != nil {
			output.PrintError(os.Stderr, serr)
			os.Exit(1)
		}
	}

	// get event only when it's test run in main container
	if r.GetType().IsMain() {
		output.PrintEvent("running test", e.Id)
	}

	result, err := r.Run(ctx, e)

	if r.GetType().IsMain() && e.PostRunScript != "" && !e.ExecutePostRunScriptBeforeScraping {
		output.PrintEvent("running postrun script", e.Id)

		if serr := RunScript(e.PostRunScript, params.WorkingDir); serr != nil {
			output.PrintError(os.Stderr, serr)
			os.Exit(1)
		}
	}

	if err != nil {
		output.PrintError(os.Stderr, err)
		os.Exit(1)
	}

	if r.GetType().IsMain() {
		output.PrintEvent("test execution finished", e.Id)
		output.PrintResult(result)
	}
}

// RunScript runs script
func RunScript(body, workingDir string) error {
	scriptFile, err := os.CreateTemp("", "runscript*.sh")
	if err != nil {
		return err
	}

	filename := scriptFile.Name()
	if _, err = io.Copy(scriptFile, strings.NewReader(body)); err != nil {
		return err
	}

	if err = scriptFile.Close(); err != nil {
		return err
	}

	if err = os.Chmod(filename, 0777); err != nil {
		return err
	}

	if _, err = executor.Run(workingDir, "/bin/sh", nil, filename); err != nil {
		return err
	}

	return nil
}

// GetDefaultWorkingDir gets default working directory
func GetDefaultWorkingDir(dataDir string, e testkube.Execution) string {
	workingDir := dataDir
	if e.Content != nil {
		isGitFileContentType := e.Content.Type_ == string(testkube.TestContentTypeGitFile)
		isGitDirContentType := e.Content.Type_ == string(testkube.TestContentTypeGitDir)
		isGitContentType := e.Content.Type_ == string(testkube.TestContentTypeGit)
		if isGitFileContentType || isGitDirContentType || isGitContentType {
			workingDir = filepath.Join(dataDir, "repo")
		}
	}

	return workingDir
}
