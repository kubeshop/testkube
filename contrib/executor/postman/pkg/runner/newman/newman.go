package newman

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/content"
	"github.com/kubeshop/testkube/pkg/executor/env"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
	"github.com/kubeshop/testkube/pkg/executor/scraper"
	"github.com/kubeshop/testkube/pkg/executor/scraper/factory"
	"github.com/kubeshop/testkube/pkg/tmp"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewNewmanRunner(ctx context.Context, params envs.Params) (*NewmanRunner, error) {
	output.PrintLog(fmt.Sprintf("%s Preparing test runner", ui.IconTruck))

	var err error
	r := &NewmanRunner{
		Params:  params,
		Fetcher: content.NewFetcher(""),
	}

	r.Scraper, err = factory.TryGetScrapper(ctx, params)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// NewmanRunner struct for newman based runner
type NewmanRunner struct {
	Params  envs.Params
	Fetcher content.ContentFetcher
	Scraper scraper.Scraper
}

var _ runner.Runner = &NewmanRunner{}

// Run runs particular test content on top of newman binary
func (r *NewmanRunner) Run(ctx context.Context, execution testkube.Execution) (result testkube.ExecutionResult, err error) {
	if r.Scraper != nil {
		defer r.Scraper.Close()
	}

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

	variablesFileContent := execution.VariablesFile
	if execution.IsVariablesFileUploaded {
		b, err := os.ReadFile(filepath.Join(content.UploadsFolder, execution.VariablesFile))
		if err != nil {
			return result, fmt.Errorf("could not read uploaded variables file: %w", err)
		}
		variablesFileContent = string(b)
	}

	// write params to tmp file
	envReader, err := NewEnvFileReader(envManager.Variables, variablesFileContent, envManager.GetSecretEnvs())
	if err != nil {
		return result, err
	}
	envpath, err := tmp.ReaderToTmpfile(envReader)
	if err != nil {
		return result, err
	}

	tmpName := tmp.Name() + ".json"
	args := execution.Args
	for i := range args {
		if args[i] == "<envFile>" {
			args[i] = envpath
		}

		if args[i] == "<reportFile>" {
			args[i] = tmpName
		}

		if args[i] == "<runPath>" {
			args[i] = path
		}
	}

	runPath := ""
	if execution.Content.Repository != nil && execution.Content.Repository.WorkingDir != "" {
		runPath = filepath.Join(r.Params.DataDir, "repo", execution.Content.Repository.WorkingDir)
	}

	// we'll get error here in case of failed test too so we treat this as
	// starter test execution with failed status
	command := strings.Join(execution.Command, " ")
	output.PrintLogf("%s Test run command %s %s", ui.IconRocket, command, strings.Join(args, " "))
	out, err := executor.Run(runPath, command, envManager, args...)

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

	// scrape artifacts first even if there are errors above
	if r.Params.ScrapperEnabled && execution.ArtifactRequest != nil && len(execution.ArtifactRequest.Dirs) != 0 {
		output.PrintLogf("Scraping directories: %v", execution.ArtifactRequest.Dirs)

		if err := r.Scraper.Scrape(ctx, execution.ArtifactRequest.Dirs, execution); err != nil {
			return *result.WithErrors(err), nil
		}
	}

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
