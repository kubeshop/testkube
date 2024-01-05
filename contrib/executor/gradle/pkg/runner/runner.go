package runner

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/joshdk/go-junit"
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

func NewRunner(ctx context.Context, params envs.Params) (*GradleRunner, error) {
	output.PrintLogf("%s Preparing test runner", ui.IconTruck)

	var err error
	r := &GradleRunner{
		params: params,
	}

	r.Scraper, err = factory.TryGetScrapper(ctx, params)
	if err != nil {
		return nil, err
	}

	return r, nil
}

type GradleRunner struct {
	params  envs.Params
	Scraper scraper.Scraper
}

func (r *GradleRunner) Run(ctx context.Context, execution testkube.Execution) (result testkube.ExecutionResult, err error) {
	if r.Scraper != nil {
		defer r.Scraper.Close()
	}

	output.PrintLogf("%s Preparing for test run", ui.IconTruck)
	err = r.Validate(execution)
	if err != nil {
		return result, err
	}

	// check that the datadir exists
	_, err = os.Stat(r.params.DataDir)
	if errors.Is(err, os.ErrNotExist) {
		output.PrintLogf("%s Datadir %s does not exist", ui.IconCross, r.params.DataDir)
		return result, err
	}

	// TODO design it better for now just append variables as envs
	envManager := env.NewManagerWithVars(execution.Variables)
	envManager.GetReferenceVars(envManager.Variables)

	// check settings.gradle or settings.gradle.kts files exist
	directory := filepath.Join(r.params.DataDir, "repo", execution.Content.Repository.Path)

	fileInfo, err := os.Stat(directory)
	if err != nil {
		return result, err
	}

	if !fileInfo.IsDir() {
		output.PrintLogf("%s passing gradle test as single file not implemented yet", ui.IconCross)
		return result, errors.Errorf("passing gradle test as single file not implemented yet")
	}

	settingsGradle := filepath.Join(directory, "settings.gradle")
	settingsGradleKts := filepath.Join(directory, "settings.gradle.kts")

	_, settingsGradleErr := os.Stat(settingsGradle)
	_, settingsGradleKtsErr := os.Stat(settingsGradleKts)
	if errors.Is(settingsGradleErr, os.ErrNotExist) && errors.Is(settingsGradleKtsErr, os.ErrNotExist) {
		output.PrintLogf("%s no settings.gradle or settings.gradle.kts found", ui.IconCross)
		return *result.Err(errors.New("no settings.gradle or settings.gradle.kts found")), nil
	}

	// determine the Gradle command to use
	// pass additional executor arguments/flags to Gradle
	gradleCommand, args := executor.MergeCommandAndArgs(execution.Command, execution.Args)
	gradleWrapper := filepath.Join(directory, "gradlew")
	_, err = os.Stat(gradleWrapper)
	if gradleCommand == "gradle" && err == nil {
		// then we use the wrapper instead
		gradleCommand = "./gradlew"
	}

	var taskName string
	task := strings.Split(execution.TestType, "/")[1]
	if !strings.EqualFold(task, "project") {
		// then use the test subtype as task name
		taskName = task
	}

	runPath := directory
	var project string
	if execution.Content.Repository != nil && execution.Content.Repository.WorkingDir != "" {
		runPath = filepath.Join(r.params.DataDir, "repo", execution.Content.Repository.WorkingDir)
		project = directory
	}

	for i := len(args) - 1; i >= 0; i-- {
		if taskName == "" && args[i] == "<taskName>" {
			args = append(args[:i], args[i+1:]...)
			continue
		}

		if project == "" && (args[i] == "-p" || args[i] == "<projectDir>") {
			args = append(args[:i], args[i+1:]...)
			continue
		}

		if args[i] == "<taskName>" {
			args[i] = taskName
		}

		if args[i] == "<projectDir>" {
			args[i] = project
		}

		args[i] = os.ExpandEnv(args[i])
	}

	output.PrintEvent("Running task: "+task, project, gradleCommand, envManager.ObfuscateStringSlice(args))
	out, err := executor.Run(runPath, gradleCommand, envManager, args...)
	out = envManager.ObfuscateSecrets(out)

	var ls []string
	_ = filepath.Walk("/data", func(path string, info fs.FileInfo, err error) error {
		ls = append(ls, path)
		return nil
	})
	output.PrintEvent("/data content", ls)

	if err == nil {
		output.PrintLogf("%s Test execution passed", ui.IconCheckMark)
		result.Status = testkube.ExecutionStatusPassed
	} else {
		output.PrintLogf("%s Test execution failed: %s", ui.IconCross, err.Error())
		result.Status = testkube.ExecutionStatusFailed
		result.ErrorMessage = err.Error()
		if strings.Contains(result.ErrorMessage, "exit status 1") {
			// probably some tests have failed
			result.ErrorMessage = "build failed with an exception"
		} else {
			// Gradle was unable to run at all
			return result, nil
		}
	}

	var rerr error
	if execution.PostRunScript != "" && execution.ExecutePostRunScriptBeforeScraping {
		output.PrintLog(fmt.Sprintf("%s Running post run script...", ui.IconCheckMark))

		if runPath == "" {
			runPath = r.params.WorkingDir
		}

		if rerr = agent.RunScript(execution.PostRunScript, runPath); rerr != nil {
			output.PrintLogf("%s Failed to execute post run script %s", ui.IconWarning, rerr)
		}
	}

	// scrape artifacts first even if there are errors above
	if r.params.ScrapperEnabled && execution.ArtifactRequest != nil && len(execution.ArtifactRequest.Dirs) != 0 {
		output.PrintLogf("Scraping directories: %v with masks: %v", execution.ArtifactRequest.Dirs, execution.ArtifactRequest.Masks)

		if err := r.Scraper.Scrape(ctx, execution.ArtifactRequest.Dirs, execution.ArtifactRequest.Masks, execution); err != nil {
			return *result.WithErrors(err), nil
		}
	}

	result.Output = string(out)
	result.OutputType = "text/plain"

	junitReportPath := filepath.Join(directory, "build", "test-results")
	err = filepath.Walk(junitReportPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			output.PrintLogf("%s Could not process reports: %s", ui.IconCross, err.Error())
			return err
		}

		if !info.IsDir() && filepath.Ext(path) == ".xml" {
			suites, _ := junit.IngestFile(path)
			for _, suite := range suites {
				for _, test := range suite.Tests {
					result.Steps = append(
						result.Steps,
						testkube.ExecutionStepResult{
							Name:     fmt.Sprintf("%s - %s", suite.Name, test.Name),
							Duration: test.Duration.String(),
							Status:   mapStatus(test.Status),
						})
				}
			}
		}

		return nil
	})

	if err != nil {
		return *result.Err(err), nil
	}

	if rerr != nil {
		return *result.Err(rerr), nil
	}

	return result, nil
}

func mapStatus(in junit.Status) (out string) {
	switch string(in) {
	case "passed":
		return string(testkube.PASSED_ExecutionStatus)
	default:
		return string(testkube.FAILED_ExecutionStatus)
	}
}

// GetType returns runner type
func (r *GradleRunner) GetType() runner.Type {
	return runner.TypeMain
}

// Validate checks if Execution has valid data in context of Gradle executor
func (r *GradleRunner) Validate(execution testkube.Execution) error {

	if execution.Content == nil {
		output.PrintLogf("%s Can't find any content to run in execution data", ui.IconCross)
		return errors.Errorf("can't find any content to run in execution data: %+v", execution)
	}

	if execution.Content.Repository == nil {
		output.PrintLogf("%s Gradle executor handles only repository based tests, but repository is nil", ui.IconCross)
		return errors.Errorf("gradle executor handles only repository based tests, but repository is nil")
	}

	if execution.Content.Repository.Branch == "" && execution.Content.Repository.Commit == "" {
		output.PrintLogf("%s Can't find branch or commit in params must use one or the other, repo %+v", ui.IconCross, execution.Content.Repository)
		return errors.Errorf("can't find branch or commit in params must use one or the other, repo:%+v", execution.Content.Repository)
	}

	return nil
}
