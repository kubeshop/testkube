package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/agent"
	"github.com/kubeshop/testkube/pkg/executor/env"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
	"github.com/kubeshop/testkube/pkg/executor/scraper"
	"github.com/kubeshop/testkube/pkg/executor/scraper/factory"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewPlaywrightRunner(ctx context.Context, dependency string, params envs.Params) (*PlaywrightRunner, error) {
	output.PrintLogf("%s Preparing test runner", ui.IconTruck)

	var err error
	r := &PlaywrightRunner{
		Params:     params,
		dependency: dependency,
	}

	r.Scraper, err = factory.TryGetScrapper(ctx, params)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// PlaywrightRunner - implements runner interface used in worker to start test execution
type PlaywrightRunner struct {
	Params     envs.Params
	Scraper    scraper.Scraper
	dependency string
}

var _ runner.Runner = &PlaywrightRunner{}

func (r *PlaywrightRunner) Run(ctx context.Context, execution testkube.Execution) (result testkube.ExecutionResult, err error) {
	if r.Scraper != nil {
		defer r.Scraper.Close()
	}
	output.PrintLogf("%s Preparing for test run", ui.IconTruck)

	// check that the datadir exists
	_, err = os.Stat(r.Params.DataDir)
	if errors.Is(err, os.ErrNotExist) {
		output.PrintLogf("%s Datadir %s does not exist", ui.IconCross, r.Params.DataDir)
		return result, errors.Errorf("datadir not exist: %v", err)
	}

	runPath := filepath.Join(r.Params.DataDir, "repo", execution.Content.Repository.Path)
	if execution.Content.Repository != nil && execution.Content.Repository.WorkingDir != "" {
		runPath = filepath.Join(r.Params.DataDir, "repo", execution.Content.Repository.WorkingDir)
	}

	if _, err := os.Stat(filepath.Join(runPath, "package.json")); err == nil {
		out, err := executor.Run(runPath, r.dependency, nil, "install")
		if err != nil {
			output.PrintLogf("%s Dependency installation error %s", ui.IconCross, r.dependency)
			return result, errors.Errorf("%s install error: %v\n\n%s", r.dependency, err, out)
		}
		output.PrintLogf("%s Dependencies successfully installed", ui.IconBox)
	}

	var depManager, depCommand string
	if r.dependency == "pnpm" {
		depManager = "pnpm"
		depCommand = "dlx"
	} else {
		depManager = "npx"
	}

	args := execution.Args
	for i := range execution.Command {
		if execution.Command[i] == "<depManager>" {
			execution.Command[i] = depManager
		}
	}

	for i := len(args) - 1; i >= 0; i-- {
		if depCommand == "" && args[i] == "<depCommand>" {
			args = append(args[:i], args[i+1:]...)
			continue
		}

		if args[i] == "<depCommand>" {
			args[i] = depCommand
		}

		args[i] = os.ExpandEnv(args[i])
	}

	envManager := env.NewManagerWithVars(execution.Variables)
	envManager.GetReferenceVars(envManager.Variables)

	command, args := executor.MergeCommandAndArgs(execution.Command, args)
	output.PrintEvent("Running", runPath, command, envManager.ObfuscateStringSlice(args))
	out, runErr := executor.Run(runPath, command, envManager, args...)
	out = envManager.ObfuscateSecrets(out)

	if runErr != nil {
		output.PrintLogf("%s Test run failed", ui.IconCross)
		result = testkube.ExecutionResult{
			Status:     testkube.ExecutionStatusFailed,
			OutputType: "text/plain",
			Output:     fmt.Sprintf("playwright test error: %s\n\n%s", runErr.Error(), out),
		}
	} else {
		result = testkube.ExecutionResult{
			Status:     testkube.ExecutionStatusPassed,
			OutputType: "text/plain",
			Output:     string(out),
		}
	}

	var rerr error
	if execution.PostRunScript != "" && execution.ExecutePostRunScriptBeforeScraping {
		output.PrintLog(fmt.Sprintf("%s Running post run script...", ui.IconCheckMark))

		if runPath == "" {
			runPath = r.Params.WorkingDir
		}

		if rerr = agent.RunScript(execution.PostRunScript, runPath); rerr != nil {
			output.PrintLogf("%s Failed to execute post run script %s", ui.IconWarning, rerr)
		}
	}

	if r.Params.ScrapperEnabled {
		reportFile := "playwright-report"
		if err = scrapeArtifacts(ctx, r, execution, reportFile); err != nil {
			return result, err
		}
	}

	if rerr != nil {
		return result, rerr
	}

	if runErr == nil {
		output.PrintLogf("%s Test run successful", ui.IconCheckMark)
	}
	return result, runErr
}

// GetType returns runner type
func (r *PlaywrightRunner) GetType() runner.Type {
	return runner.TypeMain
}

func scrapeArtifacts(ctx context.Context, r *PlaywrightRunner, execution testkube.Execution, reportName string) (err error) {
	projectPath := filepath.Join(r.Params.DataDir, "repo", execution.Content.Repository.Path)

	compressedName := reportName + "-zip"
	if _, err := executor.Run(projectPath, "mkdir", nil, compressedName); err != nil {
		output.PrintLogf("%s Artifact scraping failed: making dir %s", ui.IconCross, compressedName)
		return errors.Errorf("mkdir error: %v", err)
	}

	if _, err := executor.Run(projectPath, "zip", nil, compressedName+"/"+reportName+".zip", "-r", reportName); err != nil {
		output.PrintLogf("%s Artifact scraping failed: zipping reports %s", ui.IconCross, reportName)
		return errors.Errorf("zip error: %v", err)
	}

	directories := []string{
		filepath.Join(projectPath, compressedName),
	}

	var masks []string
	if execution.ArtifactRequest != nil {
		directories = append(directories, execution.ArtifactRequest.Dirs...)
		masks = execution.ArtifactRequest.Masks
	}

	output.PrintLogf("Scraping directories: %v with masks: %v", directories, masks)

	if err := r.Scraper.Scrape(ctx, directories, masks, execution); err != nil {
		return errors.Wrap(err, "error scraping artifacts")
	}

	return nil
}
