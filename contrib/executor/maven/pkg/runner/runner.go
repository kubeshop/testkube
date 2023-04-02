package runner

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/joshdk/go-junit"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/env"
	outputPkg "github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
	"github.com/kubeshop/testkube/pkg/ui"
)

type Params struct {
	Datadir string // RUNNER_DATADIR
}

func NewRunner() *MavenRunner {
	outputPkg.PrintLogf("%s Preparing test runner", ui.IconTruck)

	outputPkg.PrintLogf("%s Reading environment variables...", ui.IconWorld)
	params := Params{
		Datadir: os.Getenv("RUNNER_DATADIR"),
	}
	outputPkg.PrintLogf("%s Environment variables read successfully", ui.IconCheckMark)
	outputPkg.PrintLogf("RUNNER_DATADIR=\"%s\"", params.Datadir)

	return &MavenRunner{
		params: params,
	}
}

type MavenRunner struct {
	params Params
}

var _ runner.Runner = &MavenRunner{}

func (r *MavenRunner) Run(ctx context.Context, execution testkube.Execution) (result testkube.ExecutionResult, err error) {
	outputPkg.PrintLogf("%s Preparing for test run", ui.IconTruck)
	err = r.Validate(execution)
	if err != nil {
		return result, err
	}

	// check that the datadir exists
	_, err = os.Stat(r.params.Datadir)
	if errors.Is(err, os.ErrNotExist) {
		outputPkg.PrintLogf("%s Datadir %s does not exist", ui.IconCross, r.params.Datadir)
		return result, err
	}

	// check that pom.xml file exists
	directory := filepath.Join(r.params.Datadir, "repo", execution.Content.Repository.Path)

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
	mavenCommand := "mvn"
	mavenWrapper := filepath.Join(directory, "mvnw")
	_, err = os.Stat(mavenWrapper)
	if err == nil {
		// then we use the wrapper instead
		mavenCommand = "./mvnw"
	}

	envManager := env.NewManagerWithVars(execution.Variables)
	envManager.GetReferenceVars(envManager.Variables)

	// pass additional executor arguments/flags to Gradle
	var args []string
	args = append(args, execution.Args...)

	if execution.VariablesFile != "" {
		outputPkg.PrintLogf("%s Creating settings.xml file", ui.IconWorld)
		settingsXML, err := createSettingsXML(directory, execution.VariablesFile)
		if err != nil {
			outputPkg.PrintLogf("%s Could not create settings.xml", ui.IconCross)
			return *result.Err(errors.New("could not create settings.xml")), nil
		}
		outputPkg.PrintLogf("%s Successfully created settings.xml", ui.IconCheckMark)
		args = append(args, "--settings", settingsXML)
	}

	goal := strings.Split(execution.TestType, "/")[1]
	if !strings.EqualFold(goal, "project") {
		// use the test subtype as goal or phase when != project
		// in case of project there is need to pass additional args
		args = append(args, goal)
	}

	// workaround for https://github.com/eclipse/che/issues/13926
	os.Unsetenv("MAVEN_CONFIG")

	currentUser, err := user.Current()
	if err == nil && currentUser.Name == "maven" {
		args = append(args, "-Duser.home=/home/maven")
	}

	runPath := directory
	if execution.Content.Repository != nil && execution.Content.Repository.WorkingDir != "" {
		runPath = filepath.Join(r.params.Datadir, "repo", execution.Content.Repository.WorkingDir)
	}

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

// createSettingsXML saves the settings.xml to maven config folder and adds it to the list of arguments
func createSettingsXML(directory string, content string) (string, error) {
	settingsXML := filepath.Join(directory, "settings.xml")
	err := os.WriteFile(settingsXML, []byte(content), 0644)
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
