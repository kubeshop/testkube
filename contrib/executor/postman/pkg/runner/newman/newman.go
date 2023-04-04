package newman

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/content"
	"github.com/kubeshop/testkube/pkg/executor/env"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
	"github.com/kubeshop/testkube/pkg/tmp"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewNewmanRunner(params envs.Params) (*NewmanRunner, error) {
	output.PrintLog(fmt.Sprintf("%s Preparing test runner", ui.IconTruck))

	return &NewmanRunner{
		Params:  params,
		Fetcher: content.NewFetcher(""),
	}, nil
}

// NewmanRunner struct for newman based runner
type NewmanRunner struct {
	Params  envs.Params
	Fetcher content.ContentFetcher
}

var _ runner.Runner = &NewmanRunner{}

// Run runs particular test content on top of newman binary
func (r *NewmanRunner) Run(ctx context.Context, execution testkube.Execution) (result testkube.ExecutionResult, err error) {
	output.PrintLog(fmt.Sprintf("%s Preparing for test run", ui.IconTruck))

	if r.Params.GitUsername != "" || r.Params.GitToken != "" {
		if execution.Content != nil && execution.Content.Repository != nil {
			execution.Content.Repository.Username = r.Params.GitUsername
			execution.Content.Repository.Token = r.Params.GitToken
		}
	}

	path, err := r.Fetcher.Fetch(execution.Content)
	if err != nil {
		return result, err
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		return result, err
	}

	if fileInfo.IsDir() {
		return result, testkube.ErrTestContentTypeNotFile
	}

	envManager := env.NewManagerWithVars(execution.Variables)
	envManager.GetReferenceVars(envManager.Variables)
	// write params to tmp file
	envReader, err := NewEnvFileReader(envManager.Variables, execution.VariablesFile, envManager.GetSecretEnvs())
	if err != nil {
		return result, err
	}
	envpath, err := tmp.ReaderToTmpfile(envReader)
	if err != nil {
		return result, err
	}

	tmpName := tmp.Name() + ".json"

	args := []string{
		"run", path, "-e", envpath, "--reporters", "cli,json", "--reporter-json-export", tmpName,
	}
	args = append(args, execution.Args...)

	runPath := ""
	if execution.Content.Repository != nil && execution.Content.Repository.WorkingDir != "" {
		runPath = filepath.Join(r.Params.DataDir, "repo", execution.Content.Repository.WorkingDir)
	}

	// we'll get error here in case of failed test too so we treat this as
	// starter test execution with failed status
	out, err := executor.Run(runPath, "newman", envManager, args...)

	out = envManager.ObfuscateSecrets(out)

	// try to get json result even if process returned error (could be invalid test)
	newmanResult, nerr := r.GetNewmanResult(tmpName, out)
	if nerr != nil {
		output.PrintLog(fmt.Sprintf("%s Could not get Newman result: %s", ui.IconCross, nerr.Error()))
	} else {
		output.PrintLog(fmt.Sprintf("%s Got Newman result successfully", ui.IconCheckMark))
	}
	// convert newman result to OpenAPI struct
	result = MapMetadataToResult(newmanResult)
	output.PrintLog(fmt.Sprintf("%s Mapped Newman result successfully", ui.IconCheckMark))

	// catch errors if any
	if err != nil {
		return *result.Err(err), nil
	}

	if nerr != nil {
		return *result.Err(nerr), nil
	}

	return result, nil
}

func (r *NewmanRunner) GetNewmanResult(tmpName string, out []byte) (newmanResult NewmanExecutionResult, err error) {
	newmanResult.Output = string(out)

	// parse JSON output of newman test
	bytes, err := os.ReadFile(tmpName)
	if err != nil {
		return newmanResult, err
	}

	err = json.Unmarshal(bytes, &newmanResult.Metadata)
	if err != nil {
		return newmanResult, err
	}

	return
}

// GetType returns runner type
func (r NewmanRunner) GetType() runner.Type {
	return runner.TypeMain
}
