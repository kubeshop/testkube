package tests

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/renderer"
	apiclientv1 "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/test/detector"
	"github.com/kubeshop/testkube/pkg/ui"
)

func printExecutionDetails(execution testkube.Execution) {
	ui.Warn("Type:             ", execution.TestType)
	ui.Warn("Name:             ", execution.TestName)
	if execution.Id != "" {
		ui.Warn("Execution ID:     ", execution.Id)
		ui.Warn("Execution name:   ", execution.Name)
		if execution.Number != 0 {
			ui.Warn("Execution number: ", fmt.Sprintf("%d", execution.Number))
		}
		ui.Warn("Status:           ", string(*execution.ExecutionResult.Status))
		ui.Warn("Start time:       ", execution.StartTime.String())
		ui.Warn("End time:         ", execution.EndTime.String())
		ui.Warn("Duration:         ", execution.Duration)
	}

	renderer.RenderVariables(execution.Variables)

	ui.NL()
	ui.NL()
}

func DownloadArtifacts(id, dir string, client apiclientv1.Client) {
	artifacts, err := client.GetExecutionArtifacts(id)
	ui.ExitOnError("getting artifacts ", err)

	err = os.MkdirAll(dir, os.ModePerm)
	ui.ExitOnError("creating dir "+dir, err)

	if len(artifacts) > 0 {
		ui.Info("Getting artifacts", fmt.Sprintf("count = %d", len(artifacts)), "\n")
	}
	for _, artifact := range artifacts {
		f, err := client.DownloadFile(id, artifact.Name, dir)
		ui.ExitOnError("downloading file: "+f, err)
		ui.Warn(" - downloading file ", f)
	}

	ui.NL()
	ui.NL()
}

func watchLogs(id string, client apiclientv1.Client) {
	ui.Info("Getting pod logs")

	logs, err := client.Logs(id)
	ui.ExitOnError("getting logs from executor", err)

	for l := range logs {
		switch l.Type_ {
		case output.TypeError:
			ui.UseStderr()
			ui.Errf(l.Content)
			if l.Result != nil {
				ui.Errf("Error: %s", l.Result.ErrorMessage)
				ui.Debug("Output: %s", l.Result.Output)
			}
			uiShellGetExecution(id)
			os.Exit(1)
			return
		case output.TypeResult:
			ui.Info("Execution completed", l.Result.Output)
		default:
			ui.LogLine(l.String())
		}
	}

	ui.NL()

	// TODO Websocket research + plug into Event bus (EventEmitter)
	// watch for success | error status - in case of connection error on logs watch need fix in 0.8
	for range time.Tick(time.Second) {
		execution, err := client.GetExecution(id)
		ui.ExitOnError("get test execution details", err)

		fmt.Print(".")

		if execution.ExecutionResult.IsCompleted() {
			fmt.Println()

			uiShellGetExecution(id)

			return
		}
	}

	uiShellGetExecution(id)
}

func newContentFromFlags(cmd *cobra.Command) (content *testkube.TestContent, err error) {
	var fileContent []byte

	testContentType := cmd.Flag("test-content-type").Value.String()
	file := cmd.Flag("file").Value.String()
	uri := cmd.Flag("uri").Value.String()
	gitUri := cmd.Flag("git-uri").Value.String()
	gitBranch := cmd.Flag("git-branch").Value.String()
	gitCommit := cmd.Flag("git-commit").Value.String()
	gitPath := cmd.Flag("git-path").Value.String()
	gitUsername := cmd.Flag("git-username").Value.String()
	gitToken := cmd.Flag("git-token").Value.String()
	gitUsernameSecret, err := cmd.Flags().GetStringToString("git-username-secret")
	if err != nil {
		return content, err
	}

	gitTokenSecret, err := cmd.Flags().GetStringToString("git-token-secret")
	if err != nil {
		return content, err
	}

	sourceName := cmd.Flag("source").Value.String()
	// get file content
	if file != "" {
		fileContent, err = os.ReadFile(file)
		if err != nil {
			return content, fmt.Errorf("reading file "+file+" error: %w", err)
		}
	} else if stat, _ := os.Stdin.Stat(); (stat.Mode() & os.ModeCharDevice) == 0 {
		fileContent, err = io.ReadAll(os.Stdin)
		if err != nil {
			return content, fmt.Errorf("reading stdin error: %w", err)
		}
	}

	// content is correct when is passed from file, by uri, by git repo or by test source
	if len(fileContent) == 0 && uri == "" && gitUri == "" && sourceName == "" {
		return content, fmt.Errorf("empty test content, please pass some test content to create test")
	}

	hasGitParams := gitBranch != "" || gitCommit != "" || gitPath != "" || gitUri != "" || gitToken != "" || gitUsername != "" ||
		len(gitUsernameSecret) > 0 || len(gitTokenSecret) > 0

	if hasGitParams && testContentType == "" {
		testContentType = string(testkube.TestContentTypeGitDir)
	}

	if uri != "" && testContentType == "" {
		testContentType = string(testkube.TestContentTypeFileURI)
	}

	if len(fileContent) > 0 {
		testContentType = string(testkube.TestContentTypeString)
	}

	var repository *testkube.Repository
	if hasGitParams {
		if testContentType == "" {
			testContentType = "git-dir"
		}

		repository = &testkube.Repository{
			Type_:    "git",
			Uri:      gitUri,
			Branch:   gitBranch,
			Commit:   gitCommit,
			Path:     gitPath,
			Username: gitUsername,
			Token:    gitToken,
		}

		for key, val := range gitUsernameSecret {
			repository.UsernameSecret = &testkube.SecretRef{
				Name: key,
				Key:  val,
			}
		}

		for key, val := range gitTokenSecret {
			repository.TokenSecret = &testkube.SecretRef{
				Name: key,
				Key:  val,
			}
		}
	}

	content = &testkube.TestContent{
		Type_:      testContentType,
		Data:       string(fileContent),
		Repository: repository,
		Uri:        uri,
	}

	return content, nil
}

func NewUpsertTestOptionsFromFlags(cmd *cobra.Command, testLabels map[string]string) (options apiclientv1.UpsertTestOptions, err error) {
	content, err := newContentFromFlags(cmd)

	ui.ExitOnError("creating content from passed parameters", err)

	name := cmd.Flag("name").Value.String()
	executorType := cmd.Flag("type").Value.String()
	namespace := cmd.Flag("namespace").Value.String()
	labels, err := cmd.Flags().GetStringToString("label")
	if err != nil {
		return options, err
	}

	variables, err := common.CreateVariables(cmd)
	if err != nil {
		return options, err
	}

	schedule := cmd.Flag("schedule").Value.String()
	binrayArgs, err := cmd.Flags().GetStringArray("executor-args")
	if err != nil {
		return options, err
	}

	executorArgs, err := prepareExecutorArgs(binrayArgs)
	if err != nil {
		return options, err
	}

	options = apiclientv1.UpsertTestOptions{
		Name:      name,
		Type_:     executorType,
		Content:   content,
		Namespace: namespace,
		Schedule:  schedule,
	}

	executionName := cmd.Flag("execution-name").Value.String()
	envs, err := cmd.Flags().GetStringToString("env")
	if err != nil {
		return options, err
	}

	secretEnvs, err := cmd.Flags().GetStringToString("secret-env")
	if err != nil {
		return options, err
	}

	paramsFileContent := ""
	variablesFile := cmd.Flag("variables-file").Value.String()
	if variablesFile != "" {
		b, err := os.ReadFile(variablesFile)
		if err != nil {
			return options, err
		}

		paramsFileContent = string(b)
	}

	httpProxy := cmd.Flag("http-proxy").Value.String()
	httpsProxy := cmd.Flag("https-proxy").Value.String()
	options.ExecutionRequest = &testkube.ExecutionRequest{
		Name:          executionName,
		VariablesFile: paramsFileContent,
		Variables:     variables,
		Args:          executorArgs,
		Envs:          envs,
		SecretEnvs:    secretEnvs,
		HttpProxy:     httpProxy,
		HttpsProxy:    httpsProxy,
	}

	// if labels are passed and are different from the existing overwrite
	if len(labels) > 0 && !reflect.DeepEqual(testLabels, labels) {
		options.Labels = labels
	} else {
		options.Labels = testLabels
	}

	// try to detect type if none passed
	if executorType == "" {
		d := detector.NewDefaultDetector()
		if detectedType, ok := d.Detect(options); ok {
			ui.Info("Detected test test type", detectedType)
			options.Type_ = detectedType
		}
	}

	if options.Type_ == "" {
		return options, fmt.Errorf("can't detect executor type by passed file content (%s), please pass valid --type flag", executorType)
	}

	return options, nil

}

func prepareExecutorArgs(binaryArgs []string) ([]string, error) {
	executorArgs := make([]string, 0)
	for _, arg := range binaryArgs {
		r := csv.NewReader(strings.NewReader(arg))
		r.Comma = ' '
		r.LazyQuotes = true
		r.TrimLeadingSpace = true

		records, err := r.ReadAll()
		if err != nil {
			return nil, err
		}

		if len(records) != 1 {
			return nil, errors.New("single string expected")
		}

		executorArgs = append(executorArgs, records[0]...)
	}

	return executorArgs, nil
}
