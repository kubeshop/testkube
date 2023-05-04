package runner

import (
	"context"
	"os"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor/content"
	"github.com/kubeshop/testkube/pkg/executor/env"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewRunner(params envs.Params) *ExampleRunner {
	return &ExampleRunner{
		params: params,
	}
}

// ExampleRunner for template - change me to some valid runner
type ExampleRunner struct {
	params envs.Params
}

var _ runner.Runner = &ExampleRunner{}

func (r *ExampleRunner) Run(ctx context.Context, execution testkube.Execution) (result testkube.ExecutionResult, err error) {

	// use `execution.Variables` for variables passed from Test/Execution
	// variables of type "secret" will be automatically decoded
	env.NewManager().GetReferenceVars(execution.Variables)
	path, _, err := content.GetPathAndWorkingDir(execution.Content, r.params.DataDir)
	if err != nil {
		output.PrintLogf("%s Failed to resolve absolute directory for %s, using the path directly", ui.IconWarning, r.params.DataDir)
	}

	output.PrintEvent("created content path", path)

	fileInfo, err := os.Stat(path)
	if err != nil {
		return result, err
	}

	if !fileInfo.IsDir() {
		output.PrintEvent("using file", execution)
		// TODO implement file based test content for string, git-file, file-uri, git
		//      or remove if not used
	}

	if fileInfo.IsDir() {
		output.PrintEvent("using dir", execution)
		// TODO implement file based test content for git-dir, git
		//      or remove if not used
	}

	// TODO run executor here

	// error result should be returned if something is not ok
	// return result.Err(fmt.Errorf("some test execution related error occured"))

	// TODO return ExecutionResult
	return testkube.ExecutionResult{
		Status: testkube.ExecutionStatusPassed,
		Output: "exmaple test output",
	}, nil
}

// GetType returns runner type
func (r *ExampleRunner) GetType() runner.Type {
	return runner.TypeMain
}
