package newman

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/agent"
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
		Params: params,
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
	Scraper scraper.Scraper
}

var _ runner.Runner = &NewmanRunner{}

// Run runs particular test content on top of newman binary
func (r *NewmanRunner) Run(ctx context.Context, execution testkube.Execution) (result testkube.ExecutionResult, err error) {
	if r.Scraper != nil {
		defer r.Scraper.Close()
	}

	output.PrintLog(fmt.Sprintf("%s Preparing for test run", ui.IconTruck))

	path, workingDir, err := content.GetPathAndWorkingDir(execution.Content, r.Params.DataDir)
	if err != nil {
		output.PrintLogf("%s Failed to resolve absolute directory for %s, using the path directly", ui.IconWarning, r.Params.DataDir)
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		return result, err
	}

	if fileInfo.IsDir() {
		scriptName := execution.Args[len(execution.Args)-1]
		if workingDir != "" {
			path = ""
			if execution.Content != nil && execution.Content.Repository != nil {
				scriptName = filepath.Join(execution.Content.Repository.Path, scriptName)
			}
		}

		execution.Args = execution.Args[:len(execution.Args)-1]
		output.PrintLogf("%s It is a directory test - trying to find file from the last executor argument %s in directory %s", ui.IconWorld, scriptName, path)

		// sanity checking for test script
		scriptFile := filepath.Join(path, workingDir, scriptName)
		fileInfo, errFile := os.Stat(scriptFile)
		if errors.Is(errFile, os.ErrNotExist) || fileInfo.IsDir() {
			output.PrintLogf("%s Could not find file %s in the directory, error: %s", ui.IconCross, scriptName, errFile)
			return *result.Err(errors.Errorf("could not find file %s in the directory: %v", scriptName, errFile)), nil
		}
		path = scriptFile
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
	hasJunit := false
	hasReport := false
	for i := range args {
		if args[i] == "<envFile>" {
			args[i] = envpath
		}

		if args[i] == "<reportFile>" {
			args[i] = tmpName
			hasReport = true
		}

		if args[i] == "<runPath>" {
			args[i] = path
		}

		if args[i] == "--reporter-json-export" {
			hasJunit = true
		}

		args[i] = os.ExpandEnv(args[i])
	}

	runPath := ""
	if workingDir != "" {
		runPath = workingDir
	}
	// we'll get error here in case of failed test too so we treat this as
	// starter test execution with failed status
	command, args := executor.MergeCommandAndArgs(execution.Command, args)
	output.PrintLogf("%s Test run command %s %s", ui.IconRocket, command, strings.Join(envManager.ObfuscateStringSlice(args), " "))
	out, err := executor.Run(runPath, command, envManager, args...)

	out = envManager.ObfuscateSecrets(out)

	var nerr error
	if hasJunit && hasReport {
		var newmanResult NewmanExecutionResult
		// try to get json result even if process returned error (could be invalid test)
		newmanResult, nerr = r.GetNewmanResult(tmpName, out)
		if nerr != nil {
			output.PrintLog(fmt.Sprintf("%s Could not get Newman result: %s", ui.IconCross, nerr.Error()))
		} else {
			output.PrintLog(fmt.Sprintf("%s Got Newman result successfully", ui.IconCheckMark))
		}
		// convert newman result to OpenAPI struct
		result = MapMetadataToResult(newmanResult)
		output.PrintLog(fmt.Sprintf("%s Mapped Newman result successfully", ui.IconCheckMark))
	} else {
		result = makeSuccessExecution(out)
	}

	var rerr error
	if execution.PostRunScript != "" && execution.ExecutePostRunScriptBeforeScraping {
		output.PrintLog(fmt.Sprintf("%s Running post run script...", ui.IconCheckMark))

		if rerr = agent.RunScript(execution.PostRunScript, r.Params.WorkingDir); rerr != nil {
			output.PrintLogf("%s Failed to execute post run script %s", ui.IconWarning, rerr)
		}
	}

	// scrape artifacts first even if there are errors above
	if r.Params.ScrapperEnabled && execution.ArtifactRequest != nil && len(execution.ArtifactRequest.Dirs) != 0 {
		output.PrintLogf("Scraping directories: %v with masks: %v", execution.ArtifactRequest.Dirs, execution.ArtifactRequest.Masks)

		if err := r.Scraper.Scrape(ctx, execution.ArtifactRequest.Dirs, execution.ArtifactRequest.Masks, execution); err != nil {
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

	if rerr != nil {
		return *result.Err(rerr), nil
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
