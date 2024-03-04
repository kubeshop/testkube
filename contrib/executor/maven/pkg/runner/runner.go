package runner

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/joshdk/go-junit"
	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/agent"
	"github.com/kubeshop/testkube/pkg/executor/content"
	"github.com/kubeshop/testkube/pkg/executor/env"
	outputPkg "github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
	"github.com/kubeshop/testkube/pkg/executor/scraper"
	"github.com/kubeshop/testkube/pkg/executor/scraper/factory"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewRunner(ctx context.Context, params envs.Params) (*MavenRunner, error) {
	outputPkg.PrintLogf("%s Preparing test runner", ui.IconTruck)

	var err error
	r := &MavenRunner{
		params: params,
	}

	r.Scraper, err = factory.TryGetScrapper(ctx, params)
	if err != nil {
		return nil, err
	}

	return r, nil
}

type MavenRunner struct {
	params  envs.Params
	Scraper scraper.Scraper
}

var _ runner.Runner = &MavenRunner{}

func (r *MavenRunner) Run(ctx context.Context, execution testkube.Execution) (result testkube.ExecutionResult, err error) {
	if r.Scraper != nil {
		defer r.Scraper.Close()
	}

	outputPkg.PrintLogf("%s Preparing for test run", ui.IconTruck)
	err = r.Validate(execution)
	if err != nil {
		return result, err
	}

	// check that the datadir exists
	_, err = os.Stat(r.params.DataDir)
	if errors.Is(err, os.ErrNotExist) {
		outputPkg.PrintLogf("%s Datadir %s does not exist", ui.IconCross, r.params.DataDir)
		return result, err
	}

	// check that pom.xml file exists
	directory := filepath.Join(r.params.DataDir, "repo", execution.Content.Repository.Path)

	fileInfo, err := os.Stat(directory)
	if err != nil {
		return result, err
	}

	if !fileInfo.IsDir() {
		outputPkg.PrintLogf("%s passing maven test as single file not implemented yet", ui.IconCross)
		return result, errors.New("passing maven test as single file not implemented yet")
	}

	pomXml := filepath.Join(directory, "pom.xml")

	_, pomXmlErr := os.Stat(pomXml)
	if errors.Is(pomXmlErr, os.ErrNotExist) {
		outputPkg.PrintLogf("%s No pom.xml found", ui.IconCross)
		return *result.Err(errors.New("no pom.xml found")), nil
	}

	// determine the Maven command to use
	// pass additional executor arguments/flags to Maven
	mavenCommand, args := executor.MergeCommandAndArgs(execution.Command, execution.Args)
	mavenWrapper := filepath.Join(directory, "mvnw")
	_, err = os.Stat(mavenWrapper)
	if mavenCommand == "mvn" && err == nil {
		// then we use the wrapper instead
		mavenCommand = "./mvnw"
	}

	envManager := env.NewManagerWithVars(execution.Variables)
	envManager.GetReferenceVars(envManager.Variables)

	var settingsXML string
	if execution.VariablesFile != "" {
		outputPkg.PrintLogf("%s Creating settings.xml file", ui.IconWorld)
		settingsXML, err = createSettingsXML(directory, execution.VariablesFile, execution.IsVariablesFileUploaded)
		if err != nil {
			outputPkg.PrintLogf("%s Could not create settings.xml", ui.IconCross)
			return *result.Err(errors.New("could not create settings.xml")), nil
		}
		outputPkg.PrintLogf("%s Successfully created settings.xml", ui.IconCheckMark)
	}

	goal := strings.Split(execution.TestType, "/")[1]
	var goalName string
	if !strings.EqualFold(goal, "project") {
		// use the test subtype as goal or phase when != project
		// in case of project there is need to pass additional args
		goalName = goal
	}

	// workaround for https://github.com/eclipse/che/issues/13926
	os.Unsetenv("MAVEN_CONFIG")

	currentUser, err := user.Current()
	var mavenHome string
	if err == nil && currentUser.Name == "maven" {
		mavenHome = "/home/maven"
	}

	runPath := directory
	if execution.Content.Repository != nil && execution.Content.Repository.WorkingDir != "" {
		runPath = filepath.Join(r.params.DataDir, "repo", execution.Content.Repository.WorkingDir)
	}

	for i := len(args) - 1; i >= 0; i-- {
		if goalName == "" && args[i] == "<goalName>" {
			args = append(args[:i], args[i+1:]...)
			continue
		}

		if settingsXML == "" && (args[i] == "--settings" || args[i] == "<settingsFile>") {
			args = append(args[:i], args[i+1:]...)
			continue
		}

		if mavenHome == "" && (args[i] == "--Duser.home" || args[i] == "<mavenHome>") {
			args = append(args[:i], args[i+1:]...)
			continue
		}

		if args[i] == "<goalName>" {
			args[i] = goalName
		}

		if args[i] == "<settingsFile>" {
			args[i] = settingsXML
		}

		if args[i] == "<mavenHome>" {
			args[i] = mavenHome
		}

		args[i] = os.ExpandEnv(args[i])
	}

	outputPkg.PrintEvent("Running goal: "+goal, mavenHome, mavenCommand, envManager.ObfuscateStringSlice(args))
	output, err := executor.Run(runPath, mavenCommand, envManager, args...)
	output = envManager.ObfuscateSecrets(output)

	if err == nil {
		result.Status = testkube.ExecutionStatusPassed
		outputPkg.PrintLogf("%s Test run successful", ui.IconCheckMark)
	} else {
		result.Status = testkube.ExecutionStatusFailed
		result.ErrorMessage = err.Error()
		outputPkg.PrintLogf("%s Test run failed: %s", ui.IconCross, err.Error())
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
		outputPkg.PrintLog(fmt.Sprintf("%s Running post run script...", ui.IconCheckMark))

		if runPath == "" {
			runPath = r.params.WorkingDir
		}

		if rerr = agent.RunScript(execution.PostRunScript, runPath); rerr != nil {
			outputPkg.PrintLogf("%s Failed to execute post run script %s", ui.IconWarning, rerr)
		}
	}

	// scrape artifacts first even if there are errors above
	if r.params.ScrapperEnabled && execution.ArtifactRequest != nil && len(execution.ArtifactRequest.Dirs) != 0 {
		outputPkg.PrintLogf("Scraping directories: %v with masks: %v", execution.ArtifactRequest.Dirs, execution.ArtifactRequest.Masks)

		if err := r.Scraper.Scrape(ctx, execution.ArtifactRequest.Dirs, execution.ArtifactRequest.Masks, execution); err != nil {
			return *result.WithErrors(err), nil
		}
	}

	result.Output = string(output)
	result.OutputType = "text/plain"

	junitReportPath := filepath.Join(directory, "target", "surefire-reports")
	err = filepath.Walk(junitReportPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
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

// createSettingsXML saves the settings.xml to maven config folder and adds it to the list of arguments.
// In case it is taken from storage, it will return the path to the file
func createSettingsXML(directory string, variablesFile string, isUploaded bool) (string, error) {
	if isUploaded {
		return filepath.Join(content.UploadsFolder, variablesFile), nil
	}

	settingsXML := filepath.Join(directory, "settings.xml")
	err := os.WriteFile(settingsXML, []byte(variablesFile), 0644)
	if err != nil {
		return "", errors.Errorf("could not create settings.xml: %v", err)
	}

	return settingsXML, nil
}

// GetType returns runner type
func (r *MavenRunner) GetType() runner.Type {
	return runner.TypeMain
}

// Validate checks if Execution has valid data in context of Maven executor
func (r *MavenRunner) Validate(execution testkube.Execution) error {

	if execution.Content == nil {
		outputPkg.PrintLogf("%s Can't find any content to run in execution data", ui.IconCross)
		return errors.Errorf("can't find any content to run in execution data: %+v", execution)
	}

	if execution.Content.Repository == nil {
		outputPkg.PrintLogf("%s Maven executor handles only repository based tests, but repository is nil", ui.IconCross)
		return errors.New("maven executor handles only repository based tests, but repository is nil")
	}

	if execution.Content.Repository.Branch == "" && execution.Content.Repository.Commit == "" {
		outputPkg.PrintLogf("%s Can't find branch or commit in params must use one or the other, repo %+v", ui.IconCross, execution.Content.Repository)
		return errors.Errorf("can't find branch or commit in params must use one or the other, repo:%+v", execution.Content.Repository)
	}

	return nil
}
