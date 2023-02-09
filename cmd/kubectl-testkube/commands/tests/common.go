package tests

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/robfig/cron"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/renderer"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
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
	ui.Info("Getting logs from test job", id)

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
	testContentType := cmd.Flag("test-content-type").Value.String()
	uri := cmd.Flag("uri").Value.String()

	data, err := common.NewDataFromFlags(cmd)
	if err != nil {
		return nil, err
	}

	fileContent := ""
	if data != nil {
		fileContent = *data
	}

	if uri != "" && testContentType == "" {
		testContentType = string(testkube.TestContentTypeFileURI)
	}

	if len(fileContent) > 0 && testContentType == "" {
		testContentType = string(testkube.TestContentTypeString)
	}

	repository, err := common.NewRepositoryFromFlags(cmd)
	if err != nil {
		return nil, err
	}

	if repository != nil && testContentType == "" {
		testContentType = string(testkube.TestContentTypeGitDir)
	}

	content = &testkube.TestContent{
		Type_:      testContentType,
		Data:       fileContent,
		Repository: repository,
		Uri:        uri,
	}

	return content, nil
}

func newArtifactRequestFromFlags(cmd *cobra.Command) (request *testkube.ArtifactRequest, err error) {
	artifactStorageClassName := cmd.Flag("artifact-storage-class-name").Value.String()
	artifactVolumeMountPath := cmd.Flag("artifact-volume-mount-path").Value.String()
	dirs, err := cmd.Flags().GetStringArray("artifact-dir")
	if err != nil {
		return nil, err
	}

	if artifactStorageClassName != "" && artifactVolumeMountPath != "" {
		request = &testkube.ArtifactRequest{
			StorageClassName: artifactStorageClassName,
			VolumeMountPath:  artifactVolumeMountPath,
			Dirs:             dirs,
		}
	}

	return request, nil
}

func newExecutionRequestFromFlags(cmd *cobra.Command) (request *testkube.ExecutionRequest, err error) {
	variables, err := common.CreateVariables(cmd)
	if err != nil {
		return nil, err
	}

	binaryArgs, err := cmd.Flags().GetStringArray("executor-args")
	if err != nil {
		return nil, err
	}

	executorArgs, err := testkube.PrepareExecutorArgs(binaryArgs)
	if err != nil {
		return nil, err
	}

	executionName := cmd.Flag("execution-name").Value.String()
	envs, err := cmd.Flags().GetStringToString("env")
	if err != nil {
		return nil, err
	}

	secretEnvs, err := cmd.Flags().GetStringToString("secret-env")
	if err != nil {
		return nil, err
	}

	paramsFileContent := ""
	variablesFile := cmd.Flag("variables-file").Value.String()
	if variablesFile != "" {
		b, err := os.ReadFile(variablesFile)
		if err != nil {
			return nil, err
		}

		paramsFileContent = string(b)
	}

	httpProxy := cmd.Flag("http-proxy").Value.String()
	httpsProxy := cmd.Flag("https-proxy").Value.String()
	image := cmd.Flag("image").Value.String()
	command, err := cmd.Flags().GetStringArray("command")
	if err != nil {
		return nil, err
	}

	timeout, err := cmd.Flags().GetInt64("timeout")
	if err != nil {
		return nil, err
	}

	negativeTest, err := cmd.Flags().GetBool("negative-test")
	if err != nil {
		return nil, err
	}

	imagePullSecretNames, err := cmd.Flags().GetStringArray("image-pull-secrets")
	if err != nil {
		return nil, err
	}

	var imageSecrets []testkube.LocalObjectReference
	for _, secretName := range imagePullSecretNames {
		imageSecrets = append(imageSecrets, testkube.LocalObjectReference{Name: secretName})
	}

	jobTemplateContent := ""
	jobTemplate := cmd.Flag("job-template").Value.String()
	if jobTemplate != "" {
		b, err := os.ReadFile(jobTemplate)
		if err != nil {
			return nil, err
		}

		jobTemplateContent = string(b)
	}

	preRunScriptContent := ""
	preRunScript := cmd.Flag("prerun-script").Value.String()
	if preRunScript != "" {
		b, err := os.ReadFile(preRunScript)
		if err != nil {
			return nil, err
		}

		preRunScriptContent = string(b)
	}

	scraperTemplateContent := ""
	scraperTemplate := cmd.Flag("scraper-template").Value.String()
	if scraperTemplate != "" {
		b, err := os.ReadFile(scraperTemplate)
		if err != nil {
			return nil, err
		}

		scraperTemplateContent = string(b)
	}

	request = &testkube.ExecutionRequest{
		Name:                  executionName,
		VariablesFile:         paramsFileContent,
		Variables:             variables,
		Image:                 image,
		Command:               command,
		Args:                  executorArgs,
		ImagePullSecrets:      imageSecrets,
		Envs:                  envs,
		SecretEnvs:            secretEnvs,
		HttpProxy:             httpProxy,
		HttpsProxy:            httpsProxy,
		ActiveDeadlineSeconds: timeout,
		JobTemplate:           jobTemplateContent,
		PreRunScript:          preRunScriptContent,
		ScraperTemplate:       scraperTemplateContent,
		NegativeTest:          negativeTest,
	}

	request.ArtifactRequest, err = newArtifactRequestFromFlags(cmd)
	if err != nil {
		return nil, err
	}

	return request, nil
}

// NewUpsertTestOptionsFromFlags creates upsert test options from command flags
func NewUpsertTestOptionsFromFlags(cmd *cobra.Command) (options apiclientv1.UpsertTestOptions, err error) {
	content, err := newContentFromFlags(cmd)
	if err != nil {
		return options, fmt.Errorf("creating content from passed parameters %w", err)
	}

	name := cmd.Flag("name").Value.String()
	executorType := cmd.Flag("type").Value.String()
	namespace := cmd.Flag("namespace").Value.String()
	labels, err := cmd.Flags().GetStringToString("label")
	if err != nil {
		return options, err
	}

	schedule := cmd.Flag("schedule").Value.String()
	if err = validateSchedule(schedule); err != nil {
		return options, err
	}

	copyFiles, err := cmd.Flags().GetStringArray("copy-files")
	if err != nil {
		return options, err
	}

	sourceName := cmd.Flag("source").Value.String()
	options = apiclientv1.UpsertTestOptions{
		Name:      name,
		Type_:     executorType,
		Content:   content,
		Source:    sourceName,
		Namespace: namespace,
		Schedule:  schedule,
		Uploads:   copyFiles,
		Labels:    labels,
	}

	options.ExecutionRequest, err = newExecutionRequestFromFlags(cmd)
	if err != nil {
		return options, err
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

// readCopyFiles reads files
func readCopyFiles(copyFiles []string) (map[string]string, error) {
	files := map[string]string{}
	for _, f := range copyFiles {
		paths := strings.Split(f, ":")
		if len(paths) != 2 {
			return nil, fmt.Errorf("invalid file format, expecting sourcePath:destinationPath")
		}
		contents, err := os.ReadFile(paths[0])
		if err != nil {
			return nil, fmt.Errorf("could not read executor copy file: %w", err)
		}
		files[paths[1]] = string(contents)
	}
	return files, nil
}

// mergeCopyFiles merges the lists of files to be copied into the running test
// the files set on execution overwrite the files set on test levels
func mergeCopyFiles(testFiles []string, executionFiles []string) ([]string, error) {
	if len(testFiles) == 0 {
		return executionFiles, nil
	}

	if len(executionFiles) == 0 {
		return testFiles, nil
	}

	files := map[string]string{}

	for _, fileMapping := range testFiles {
		fPair := strings.Split(fileMapping, ":")
		if len(fPair) != 2 {
			return []string{}, fmt.Errorf("invalid copy file mapping, expected source:destination, got: %s", fileMapping)
		}
		files[fPair[1]] = fPair[0]
	}

	for _, fileMapping := range executionFiles {
		fPair := strings.Split(fileMapping, ":")
		if len(fPair) != 2 {
			return []string{}, fmt.Errorf("invalid copy file mapping, expected source:destination, got: %s", fileMapping)
		}
		files[fPair[1]] = fPair[0]
	}

	result := []string{}
	for destination, source := range files {
		result = append(result, fmt.Sprintf("%s:%s", source, destination))
	}

	return result, nil
}

func uploadFiles(client client.Client, parentName string, parentType client.TestingType, files []string) error {
	for _, f := range files {
		paths := strings.Split(f, ":")
		if len(paths) != 2 {
			return fmt.Errorf("invalid file format, expecting sourcePath:destinationPath")
		}
		contents, err := os.ReadFile(paths[0])
		if err != nil {
			return fmt.Errorf("could not read file: %w", err)
		}

		err = client.UploadFile(parentName, parentType, paths[1], contents)
		if err != nil {
			return fmt.Errorf("could not upload file %s for %v with name %s: %w", paths[0], parentType, parentName, err)
		}
	}
	return nil
}

// NewUpdateTestOptionsFromFlags creates update test options from command flags
func NewUpdateTestOptionsFromFlags(cmd *cobra.Command) (options apiclientv1.UpdateTestOptions, err error) {
	contentUpdate, err := newContentUpdateFromFlags(cmd)
	if err != nil {
		return options, fmt.Errorf("creating content from passed parameters %w", err)
	}

	if contentUpdate != nil {
		options.Content = &contentUpdate
	}

	var fields = []struct {
		name        string
		destination **string
	}{
		{
			"name",
			&options.Name,
		},
		{
			"type",
			&options.Type_,
		},
		{
			"namespace",
			&options.Namespace,
		},
		{
			"source",
			&options.Source,
		},
	}

	for _, field := range fields {
		if cmd.Flag(field.name).Changed {
			value := cmd.Flag(field.name).Value.String()
			*field.destination = &value
		}
	}

	if cmd.Flag("schedule").Changed {
		value := cmd.Flag("schedule").Value.String()
		if err = validateSchedule(value); err != nil {
			return options, err
		}

		options.Schedule = &value
	}

	if cmd.Flag("label").Changed {
		labels, err := cmd.Flags().GetStringToString("label")
		if err != nil {
			return options, err
		}

		options.Labels = &labels
	}

	if cmd.Flag("copy-files").Changed {
		copyFiles, err := cmd.Flags().GetStringArray("copy-files")
		if err != nil {
			return options, err
		}

		options.Uploads = &copyFiles
	}

	executionRequest, err := newExecutionUpdateRequestFromFlags(cmd)
	if err != nil {
		return options, err
	}

	if executionRequest != nil {
		options.ExecutionRequest = &executionRequest
	}

	return options, nil
}

func newContentUpdateFromFlags(cmd *cobra.Command) (content *testkube.TestContentUpdate, err error) {
	content = &testkube.TestContentUpdate{}

	var fields = []struct {
		name        string
		destination **string
	}{
		{
			"test-content-type",
			&content.Type_,
		},
		{
			"uri",
			&content.Uri,
		},
	}

	var nonEmpty bool
	for _, field := range fields {
		if cmd.Flag(field.name).Changed {
			value := cmd.Flag(field.name).Value.String()
			*field.destination = &value
			nonEmpty = true
		}
	}

	data, err := common.NewDataFromFlags(cmd)
	if err != nil {
		return nil, err
	}

	if data != nil {
		content.Data = data
		nonEmpty = true
	}

	repository, err := common.NewRepositoryUpdateFromFlags(cmd)
	if err != nil {
		return nil, err
	}

	if repository != nil {
		content.Repository = &repository
		nonEmpty = true
	}

	if nonEmpty {
		var emptyValue string
		var emptyRepository = &testkube.RepositoryUpdate{}
		switch {
		case content.Data != nil:
			content.Repository = &emptyRepository
			content.Uri = &emptyValue
		case content.Repository != nil:
			content.Data = &emptyValue
			content.Uri = &emptyValue
		case content.Uri != nil:
			content.Data = &emptyValue
			content.Repository = &emptyRepository
		}

		return content, nil
	}

	return nil, nil
}

func newExecutionUpdateRequestFromFlags(cmd *cobra.Command) (request *testkube.ExecutionUpdateRequest, err error) {
	request = &testkube.ExecutionUpdateRequest{}

	var fields = []struct {
		name        string
		destination **string
	}{
		{
			"execution-name",
			&request.Name,
		},
		{
			"image",
			&request.Image,
		},
		{
			"http-proxy",
			&request.HttpProxy,
		},
		{
			"https-proxy",
			&request.HttpsProxy,
		},
	}

	var nonEmpty bool
	for _, field := range fields {
		if cmd.Flag(field.name).Changed {
			value := cmd.Flag(field.name).Value.String()
			*field.destination = &value
			nonEmpty = true
		}
	}

	if cmd.Flag("variable").Changed || cmd.Flag("secret-variable").Changed || cmd.Flag("secret-variable-reference").Changed {
		variables, err := common.CreateVariables(cmd)
		if err != nil {
			return nil, err
		}

		request.Variables = &variables
		nonEmpty = true
	}

	if cmd.Flag("executor-args").Changed {
		binaryArgs, err := cmd.Flags().GetStringArray("executor-args")
		if err != nil {
			return nil, err
		}

		executorArgs, err := testkube.PrepareExecutorArgs(binaryArgs)
		if err != nil {
			return nil, err
		}

		request.Args = &executorArgs
		nonEmpty = true
	}

	var hashes = []struct {
		name        string
		destination **map[string]string
	}{
		{
			"env",
			&request.Envs,
		},
		{
			"secret-env",
			&request.SecretEnvs,
		},
	}

	for _, hash := range hashes {
		if cmd.Flag(hash.name).Changed {
			value, err := cmd.Flags().GetStringToString(hash.name)
			if err != nil {
				return nil, err
			}

			*hash.destination = &value
			nonEmpty = true
		}
	}

	if cmd.Flag("variables-file").Changed {
		paramsFileContent := ""
		variablesFile := cmd.Flag("variables-file").Value.String()
		if variablesFile != "" {
			b, err := os.ReadFile(variablesFile)
			if err != nil {
				return nil, err
			}

			paramsFileContent = string(b)
			request.VariablesFile = &paramsFileContent
			nonEmpty = true
		}
	}

	if cmd.Flag("command").Changed {
		command, err := cmd.Flags().GetStringArray("command")
		if err != nil {
			return nil, err
		}

		request.Command = &command
		nonEmpty = true
	}

	if cmd.Flag("timeout").Changed {
		timeout, err := cmd.Flags().GetInt64("timeout")
		if err != nil {
			return nil, err
		}

		request.ActiveDeadlineSeconds = &timeout
		nonEmpty = true
	}

	if cmd.Flag("negative-test").Changed {
		negativeTest, err := cmd.Flags().GetBool("negative-test")
		if err != nil {
			return nil, err
		}
		request.NegativeTest = &negativeTest
		nonEmpty = true
	}

	if cmd.Flag("image-pull-secrets").Changed {
		imagePullSecretNames, err := cmd.Flags().GetStringArray("image-pull-secrets")
		if err != nil {
			return nil, err
		}

		var imageSecrets []testkube.LocalObjectReference
		for _, secretName := range imagePullSecretNames {
			imageSecrets = append(imageSecrets, testkube.LocalObjectReference{Name: secretName})
		}

		request.ImagePullSecrets = &imageSecrets
		nonEmpty = true
	}

	if cmd.Flag("job-template").Changed {
		jobTemplateContent := ""
		jobTemplate := cmd.Flag("job-template").Value.String()
		if jobTemplate != "" {
			b, err := os.ReadFile(jobTemplate)
			if err != nil {
				return nil, err
			}

			jobTemplateContent = string(b)
		}

		request.JobTemplate = &jobTemplateContent
		nonEmpty = true
	}

	if cmd.Flag("prerun-script").Changed {
		preRunScriptContent := ""
		preRunScript := cmd.Flag("prerun-script").Value.String()
		if preRunScript != "" {
			b, err := os.ReadFile(preRunScript)
			if err != nil {
				return nil, err
			}

			preRunScriptContent = string(b)
		}

		request.PreRunScript = &preRunScriptContent
		nonEmpty = true
	}

	if cmd.Flag("scraper-template").Changed {
		scraperTemplateContent := ""
		scraperTemplate := cmd.Flag("scraper-template").Value.String()
		if scraperTemplate != "" {
			b, err := os.ReadFile(scraperTemplate)
			if err != nil {
				return nil, err
			}

			scraperTemplateContent = string(b)
		}

		request.ScraperTemplate = &scraperTemplateContent
		nonEmpty = true
	}

	artifactRequest, err := newArtifactUpdateRequestFromFlags(cmd)
	if err != nil {
		return nil, err
	}

	var emptyRequest = &testkube.ArtifactUpdateRequest{}
	if artifactRequest != nil {
		request.ArtifactRequest = &artifactRequest
		nonEmpty = true
	} else {
		request.ArtifactRequest = &emptyRequest
	}

	if nonEmpty {
		return request, nil
	}

	return nil, nil
}

func newArtifactUpdateRequestFromFlags(cmd *cobra.Command) (request *testkube.ArtifactUpdateRequest, err error) {
	request = &testkube.ArtifactUpdateRequest{}

	var fields = []struct {
		name        string
		destination **string
	}{
		{
			"artifact-storage-class-name",
			&request.StorageClassName,
		},
		{
			"artifact-volume-mount-path",
			&request.VolumeMountPath,
		},
	}

	var nonEmpty bool
	for _, field := range fields {
		if cmd.Flag(field.name).Changed {
			value := cmd.Flag(field.name).Value.String()
			*field.destination = &value
			nonEmpty = true
		}
	}

	if cmd.Flag("artifact-dir").Changed {
		dirs, err := cmd.Flags().GetStringArray("artifact-dir")
		if err != nil {
			return nil, err
		}

		request.Dirs = &dirs
		nonEmpty = true
	}

	if nonEmpty {
		return request, nil
	}

	return nil, nil
}

func validateSchedule(schedule string) error {
	if schedule == "" {
		return nil
	}

	specParser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	if _, err := specParser.Parse(schedule); err != nil {
		return err
	}

	return nil
}
