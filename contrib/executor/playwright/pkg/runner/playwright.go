package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/executor/scraper/factory"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/env"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewPlaywrightRunner(dependency string) (*PlaywrightRunner, error) {
	output.PrintLog(fmt.Sprintf("%s Preparing test runner", ui.IconTruck))
	params, err := envs.LoadTestkubeVariables()
	if err != nil {
		return nil, fmt.Errorf("could not initialize Playwright runner variables: %w", err)
	}

	return &PlaywrightRunner{
		Params:     params,
		dependency: dependency,
	}, nil
}

// PlaywrightRunner - implements runner interface used in worker to start test execution
type PlaywrightRunner struct {
	Params     envs.Params
	dependency string
}

func (r *PlaywrightRunner) Run(execution testkube.Execution) (result testkube.ExecutionResult, err error) {
	output.PrintLog(fmt.Sprintf("%s Preparing for test run", ui.IconTruck))

	// check that the datadir exists
	_, err = os.Stat(r.Params.DataDir)
	if errors.Is(err, os.ErrNotExist) {
		output.PrintLog(fmt.Sprintf("%s Datadir %s does not exist", ui.IconCross, r.Params.DataDir))
		return result, fmt.Errorf("datadir not exist: %w", err)
	}

	runPath := filepath.Join(r.Params.DataDir, "repo", execution.Content.Repository.Path)
	if execution.Content.Repository != nil && execution.Content.Repository.WorkingDir != "" {
		runPath = filepath.Join(r.Params.DataDir, "repo", execution.Content.Repository.WorkingDir)
	}

	if _, err := os.Stat(filepath.Join(runPath, "package.json")); err == nil {
		out, err := executor.Run(runPath, r.dependency, nil, "install")
		if err != nil {
			output.PrintLog(fmt.Sprintf("%s Dependency installation error %s", ui.IconCross, r.dependency))
			return result, fmt.Errorf("%s install error: %w\n\n%s", r.dependency, err, out)
		}
		output.PrintLog(fmt.Sprintf("%s Dependencies successfully installed", ui.IconBox))
	}

	var runner string
	var args []string

	if r.dependency == "pnpm" {
		runner = "pnpm"
		args = []string{"dlx", "playwright", "test"}
	} else {
		runner = "npx"
		args = []string{"playwright", "test"}
	}

	args = append(args, execution.Args...)

	envManager := env.NewManagerWithVars(execution.Variables)
	envManager.GetReferenceVars(envManager.Variables)

	output.PrintEvent("Running", runPath, "playwright", args)
	out, runErr := executor.Run(runPath, runner, envManager, args...)
	out = envManager.ObfuscateSecrets(out)
	if runErr != nil {
		output.PrintLog(fmt.Sprintf("%s Test run failed", ui.IconCross))
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

	if r.Params.ScrapperEnabled {
		if err = scrapeArtifacts(r, execution); err != nil {
			return result, err
		}
	}

	if runErr == nil {
		output.PrintLog(fmt.Sprintf("%s Test run successful", ui.IconCheckMark))
	}
	return result, runErr
}

// GetType returns runner type
func (r *PlaywrightRunner) GetType() runner.Type {
	return runner.TypeMain
}

func scrapeArtifacts(r *PlaywrightRunner, execution testkube.Execution) (err error) {
	projectPath := filepath.Join(r.Params.DataDir, "repo", execution.Content.Repository.Path)

	originalName := "playwright-report"
	compressedName := originalName + "-zip"

	if _, err := executor.Run(projectPath, "mkdir", nil, compressedName); err != nil {
		output.PrintLog(fmt.Sprintf("%s Artifact scraping failed: making dir %s", ui.IconCross, compressedName))
		return fmt.Errorf("mkdir error: %w", err)
	}

	if _, err := executor.Run(projectPath, "zip", nil, compressedName+"/"+originalName+".zip", "-r", originalName); err != nil {
		output.PrintLog(fmt.Sprintf("%s Artifact scraping failed: zipping reports %s", ui.IconCross, originalName))
		return fmt.Errorf("zip error: %w", err)
	}

	directories := []string{
		filepath.Join(projectPath, compressedName),
	}
	output.PrintLog(fmt.Sprintf("Scraping directories: %v", directories))

	if err := factory.Scrape(context.Background(), directories, execution, r.Params); err != nil {
		return errors.Wrap(err, "error scraping artifacts")
	}

	return nil
}
