package tests

import (
	"errors"
	"fmt"
	"os"
	"path"
	"regexp"
	"strconv"
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

const (
	artifactsFormatFolder  = "folder"
	artifactsFormatArchive = "archive"
	maxArgSize             = int64(131072) // maximum argument size in linux-based systems is 128 KiB
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
		if execution.ExecutionResult != nil && execution.ExecutionResult.Status != nil {
			ui.Warn("Status:           ", string(*execution.ExecutionResult.Status))
		}
		ui.Warn("Start time:       ", execution.StartTime.String())
		ui.Warn("End time:         ", execution.EndTime.String())
		ui.Warn("Duration:         ", execution.Duration)
	}

	renderer.RenderVariables(execution.Variables)

	ui.NL()
	ui.NL()
}

func DownloadTestArtifacts(id, dir, format string, masks []string, client apiclientv1.Client) {
	artifacts, err := client.GetExecutionArtifacts(id)
	ui.ExitOnError("getting artifacts", err)

	downloadFile := func(artifact testkube.Artifact, dir string) (string, error) {
		return client.DownloadFile(id, artifact.Name, dir)
	}
	downloadArchive := func(dir string, masks []string) (string, error) {
		return client.DownloadArchive(id, dir, masks)
	}
	downloadArtifacts(dir, format, masks, artifacts, downloadFile, downloadArchive)
}

func DownloadTestWorkflowArtifacts(id, dir, format string, masks []string, client apiclientv1.Client) {
	artifacts, err := client.GetTestWorkflowExecutionArtifacts(id)
	ui.ExitOnError("getting artifacts", err)

	downloadFile := func(artifact testkube.Artifact, dir string) (string, error) {
		return client.DownloadTestWorkflowArtifact(id, artifact.Name, dir)
	}
	downloadArchive := func(dir string, masks []string) (string, error) {
		return client.DownloadTestWorkflowArtifactArchive(id, dir, masks)
	}
	downloadArtifacts(dir, format, masks, artifacts, downloadFile, downloadArchive)
}

func downloadArtifacts(
	dir, format string,
	masks []string,
	artifacts testkube.Artifacts,
	downloadFile func(artifact testkube.Artifact, dir string) (string, error),
	downloadArchive func(dir string, masks []string) (string, error),
) {
	err := os.MkdirAll(dir, os.ModePerm)
	ui.ExitOnError("creating dir "+dir, err)

	if len(artifacts) > 0 {
		ui.Info("Getting artifacts", fmt.Sprintf("count = %d", len(artifacts)), "\n")
	}

	if format != artifactsFormatFolder && format != artifactsFormatArchive {
		ui.Failf("invalid artifacts format: %s. use one of folder|archive", format)
	}

	var regexps []*regexp.Regexp
	for _, mask := range masks {
		values := strings.Split(mask, ",")
		for _, value := range values {
			re, err := regexp.Compile(value)
			ui.ExitOnError("checking mask "+value, err)

			regexps = append(regexps, re)
		}
	}

	if format == artifactsFormatFolder {
		for _, artifact := range artifacts {
			found := len(regexps) == 0
			for i := range regexps {
				if found = regexps[i].MatchString(artifact.Name); found {
					break
				}
			}

			if !found {
				continue
			}

			f, err := downloadFile(artifact, dir)
			ui.ExitOnError("downloading file: "+f, err)
			ui.Warn(" - downloading file ", f)
		}
	}

	if format == artifactsFormatArchive {
		const readinessCheckTimeout = time.Second
		ticker := time.NewTicker(readinessCheckTimeout)
		defer ticker.Stop()

		ch := make(chan string)
		defer close(ch)

		go func() {
			f, err := downloadArchive(dir, masks)
			ui.ExitOnError("downloading archive: "+f, err)

			ch <- f
		}()

		var archive string
		ui.Warn(" - preparing archive ")

	outloop:
		for {
			select {
			case <-ticker.C:
				ui.PrintDot()
			case archive = <-ch:
				ui.NL()
				break outloop
			}
		}

		ui.Warn(" - downloading archive ", archive)
	}

	ui.NL()
	ui.NL()
}

func watchLogs(id string, silentMode bool, client apiclientv1.Client) error {
	ui.Info("Getting logs from test job", id)

	logs, err := client.Logs(id)
	ui.ExitOnError("getting logs from executor", err)

	var result error
	for l := range logs {
		switch l.Type_ {
		case output.TypeError:
			ui.UseStderr()
			ui.Errf(l.Content)
			if l.Result != nil {
				ui.Errf("Error: %s", l.Result.ErrorMessage)
				ui.Debug("Output: %s", l.Result.Output)
			}
			result = errors.New(l.Content)
		case output.TypeResult:
			ui.Info("Execution completed", l.Result.Output)
		default:
			if !silentMode {
				ui.LogLine(l.String())
			}
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
			return result
		}
	}

	return result
}

func watchLogsV2(id string, silentMode bool, client apiclientv1.Client) error {
	ui.Info("Getting logs from test job", id)

	logs, err := client.LogsV2(id)
	ui.ExitOnError("getting logs from executor", err)

	var result error
	for l := range logs {
		if l.Error_ {
			ui.UseStderr()
			ui.Errf(l.Content)
			result = errors.New(l.Content)
			continue
		}

		if !silentMode {
			ui.LogLine(l.Content)
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
			ui.Info("Execution completed")
			return result
		}
	}

	return result
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
	var repository *testkube.Repository
	if cmd.Flag("git-uri") != nil {
		repository, err = common.NewRepositoryFromFlags(cmd)
		if err != nil {
			return nil, err
		}
	}

	if repository != nil && testContentType == "" {
		testContentType = string(testkube.TestContentTypeGit)
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

	masks, err := cmd.Flags().GetStringArray("artifact-mask")
	if err != nil {
		return nil, err
	}

	artifactStorageBucket := cmd.Flag("artifact-storage-bucket").Value.String()
	artifactOmitFolderPerExecution, err := cmd.Flags().GetBool("artifact-omit-folder-per-execution")
	if err != nil {
		return nil, err
	}

	artifactSharedBetweenPods, err := cmd.Flags().GetBool("artifact-shared-between-pods")
	if err != nil {
		return nil, err
	}

	if artifactStorageClassName != "" || artifactVolumeMountPath != "" || len(dirs) != 0 || len(masks) != 0 ||
		artifactStorageBucket != "" || artifactOmitFolderPerExecution || artifactSharedBetweenPods {
		request = &testkube.ArtifactRequest{
			StorageClassName:       artifactStorageClassName,
			VolumeMountPath:        artifactVolumeMountPath,
			Dirs:                   dirs,
			Masks:                  masks,
			StorageBucket:          artifactStorageBucket,
			OmitFolderPerExecution: artifactOmitFolderPerExecution,
			SharedBetweenPods:      artifactSharedBetweenPods,
		}
	}

	return request, nil
}

func newSlavePodRequestFromFlags(cmd *cobra.Command) (request *testkube.PodRequest, err error) {
	slavePodTemplate := cmd.Flag("slave-pod-template").Value.String()
	slavePodTemplateReference := cmd.Flag("slave-pod-template-reference").Value.String()
	slavePodRequestsCpu := cmd.Flag("slave-pod-requests-cpu").Value.String()
	slavePodRequestsMemory := cmd.Flag("slave-pod-requests-memory").Value.String()
	slavePodLimitsCpu := cmd.Flag("slave-pod-limits-cpu").Value.String()
	slavePodLimitsMemory := cmd.Flag("slave-pod-limits-memory").Value.String()

	if slavePodRequestsCpu != "" || slavePodRequestsMemory != "" || slavePodLimitsCpu != "" ||
		slavePodLimitsMemory != "" || slavePodTemplate != "" || slavePodTemplateReference != "" {
		request = &testkube.PodRequest{
			PodTemplateReference: slavePodTemplateReference,
		}

		if slavePodTemplate != "" {
			b, err := os.ReadFile(slavePodTemplate)
			ui.ExitOnError("reading slave pod template", err)
			request.PodTemplate = string(b)
		}

		if slavePodRequestsCpu != "" || slavePodRequestsMemory != "" {
			if request.Resources == nil {
				request.Resources = &testkube.PodResourcesRequest{}
			}

			request.Resources.Requests = &testkube.ResourceRequest{
				Cpu:    slavePodRequestsCpu,
				Memory: slavePodRequestsMemory,
			}
		}

		if slavePodLimitsCpu != "" || slavePodLimitsMemory != "" {
			if request.Resources == nil {
				request.Resources = &testkube.PodResourcesRequest{}
			}

			request.Resources.Limits = &testkube.ResourceRequest{
				Cpu:    slavePodLimitsCpu,
				Memory: slavePodLimitsMemory,
			}
		}
	}

	return request, nil
}

func newEnvReferencesFromFlags(cmd *cobra.Command) (envConfigMaps, envSecrets []testkube.EnvReference, err error) {
	mountConfigMaps, err := cmd.Flags().GetStringToString("mount-configmap")
	if err != nil {
		return nil, nil, err
	}

	variableConfigMaps, err := cmd.Flags().GetStringArray("variable-configmap")
	if err != nil {
		return nil, nil, err
	}

	mountSecrets, err := cmd.Flags().GetStringToString("mount-secret")
	if err != nil {
		return nil, nil, err
	}

	variableSecrets, err := cmd.Flags().GetStringArray("variable-secret")
	if err != nil {
		return nil, nil, err
	}

	mapConfigMaps := make(map[string]testkube.EnvReference)
	for configMap, path := range mountConfigMaps {
		mapConfigMaps[configMap] = testkube.EnvReference{
			Reference: &testkube.LocalObjectReference{
				Name: configMap,
			},
			Mount:     true,
			MountPath: path,
		}
	}

	for _, configMap := range variableConfigMaps {
		if value, ok := mapConfigMaps[configMap]; ok {
			value.MapToVariables = true
			mapConfigMaps[configMap] = value
		} else {
			mapConfigMaps[configMap] = testkube.EnvReference{
				Reference: &testkube.LocalObjectReference{
					Name: configMap,
				},
				MapToVariables: true,
			}
		}
	}

	for _, value := range mapConfigMaps {
		envConfigMaps = append(envConfigMaps, value)
	}

	mapSecrets := make(map[string]testkube.EnvReference)
	for secret, path := range mountSecrets {
		mapSecrets[secret] = testkube.EnvReference{
			Reference: &testkube.LocalObjectReference{
				Name: secret,
			},
			Mount:     true,
			MountPath: path,
		}
	}

	for _, secret := range variableSecrets {
		if value, ok := mapSecrets[secret]; ok {
			value.MapToVariables = true
			mapSecrets[secret] = value
		} else {
			mapSecrets[secret] = testkube.EnvReference{
				Reference: &testkube.LocalObjectReference{
					Name: secret,
				},
				MapToVariables: true,
			}
		}
	}

	for _, value := range mapSecrets {
		envSecrets = append(envSecrets, value)
	}

	return envConfigMaps, envSecrets, nil
}

func newExecutionRequestFromFlags(cmd *cobra.Command) (request *testkube.ExecutionRequest, err error) {
	crdOnly, err := cmd.Flags().GetBool("crd-only")
	if err != nil {
		return nil, err
	}

	disableSecretCreation := false
	if !crdOnly {
		client, _, err := common.GetClient(cmd)
		if err != nil {
			return nil, err
		}

		info, err := client.GetServerInfo()
		if err != nil {
			return nil, err
		}

		disableSecretCreation = info.DisableSecretCreation
	}

	variables, err := common.CreateVariables(cmd, disableSecretCreation)
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

	mode := ""
	if cmd.Flag("args-mode").Changed {
		mode = cmd.Flag("args-mode").Value.String()
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

	jobTemplateReference := cmd.Flag("job-template-reference").Value.String()
	cronJobTemplateReference := cmd.Flag("cronjob-template-reference").Value.String()
	scraperTemplateReference := cmd.Flag("scraper-template-reference").Value.String()
	pvcTemplateReference := cmd.Flag("pvc-template-reference").Value.String()
	executionNamespace := cmd.Flag("execution-namespace").Value.String()
	executePostRunScriptBeforeScraping, err := cmd.Flags().GetBool("execute-postrun-script-before-scraping")
	if err != nil {
		return nil, err
	}
	sourceScripts, err := cmd.Flags().GetBool("source-scripts")
	if err != nil {
		return nil, err
	}

	request = &testkube.ExecutionRequest{
		Name:                               executionName,
		Variables:                          variables,
		Image:                              image,
		Command:                            command,
		Args:                               executorArgs,
		ArgsMode:                           mode,
		ImagePullSecrets:                   imageSecrets,
		Envs:                               envs,
		SecretEnvs:                         secretEnvs,
		HttpProxy:                          httpProxy,
		HttpsProxy:                         httpsProxy,
		ActiveDeadlineSeconds:              timeout,
		JobTemplateReference:               jobTemplateReference,
		CronJobTemplateReference:           cronJobTemplateReference,
		ScraperTemplateReference:           scraperTemplateReference,
		PvcTemplateReference:               pvcTemplateReference,
		NegativeTest:                       negativeTest,
		ExecutePostRunScriptBeforeScraping: executePostRunScriptBeforeScraping,
		SourceScripts:                      sourceScripts,
		ExecutionNamespace:                 executionNamespace,
	}

	var fields = []struct {
		source      string
		destination *string
	}{
		{
			cmd.Flag("job-template").Value.String(),
			&request.JobTemplate,
		},
		{
			cmd.Flag("cronjob-template").Value.String(),
			&request.CronJobTemplate,
		},
		{
			cmd.Flag("prerun-script").Value.String(),
			&request.PreRunScript,
		},
		{
			cmd.Flag("postrun-script").Value.String(),
			&request.PostRunScript,
		},
		{
			cmd.Flag("scraper-template").Value.String(),
			&request.ScraperTemplate,
		},
		{
			cmd.Flag("pvc-template").Value.String(),
			&request.PvcTemplate,
		},
	}

	for _, field := range fields {
		if field.source != "" {
			b, err := os.ReadFile(field.source)
			if err != nil {
				return nil, err
			}

			*field.destination = string(b)
		}
	}

	request.EnvConfigMaps, request.EnvSecrets, err = newEnvReferencesFromFlags(cmd)
	if err != nil {
		return nil, err
	}

	request.ArtifactRequest, err = newArtifactRequestFromFlags(cmd)
	if err != nil {
		return nil, err
	}

	request.SlavePodRequest, err = newSlavePodRequestFromFlags(cmd)
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
	file := cmd.Flag("file").Value.String()
	executorType := cmd.Flag("type").Value.String()
	namespace := cmd.Flag("namespace").Value.String()
	description := cmd.Flag("description").Value.String()
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

	sourceName := ""
	if cmd.Flag("source") != nil {
		sourceName = cmd.Flag("source").Value.String()
	}
	options = apiclientv1.UpsertTestOptions{
		Name:        name,
		Description: description,
		Type_:       executorType,
		Content:     content,
		Source:      sourceName,
		Namespace:   namespace,
		Schedule:    schedule,
		Uploads:     copyFiles,
		Labels:      labels,
	}

	options.ExecutionRequest, err = newExecutionRequestFromFlags(cmd)
	if err != nil {
		return options, err
	}

	// try to detect type if none passed
	if executorType == "" {
		d := detector.NewDefaultDetector()
		if detectedType, ok := d.Detect(file, options); ok {
			crdOnly, _ := strconv.ParseBool(cmd.Flag("crd-only").Value.String())
			if !crdOnly {
				ui.Info("Detected test type", detectedType)
			}
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

func uploadFiles(client client.Client, parentName string, parentType client.TestingType, files []string, timeout time.Duration) error {
	for _, f := range files {
		paths := strings.Split(f, ":")
		if len(paths) != 2 {
			return fmt.Errorf("invalid file format, expecting sourcePath:destinationPath")
		}
		contents, err := os.ReadFile(paths[0])
		if err != nil {
			return fmt.Errorf("could not read file: %w", err)
		}

		err = client.UploadFile(parentName, parentType, paths[1], contents, timeout)
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
		{
			"description",
			&options.Description,
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
		{
			"args-mode",
			&request.ArgsMode,
		},
		{
			"job-template-reference",
			&request.JobTemplateReference,
		},
		{
			"cronjob-template-reference",
			&request.CronJobTemplateReference,
		},
		{
			"scraper-template-reference",
			&request.ScraperTemplateReference,
		},
		{
			"pvc-template-reference",
			&request.PvcTemplateReference,
		},
		{
			"execution-namespace",
			&request.ExecutionNamespace,
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
		client, _, err := common.GetClient(cmd)
		if err != nil {
			return nil, err
		}

		info, err := client.GetServerInfo()
		if err != nil {
			return nil, err
		}

		variables, err := common.CreateVariables(cmd, info.DisableSecretCreation)
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

	var values = []struct {
		source      string
		destination **string
	}{
		{
			"job-template",
			&request.JobTemplate,
		},
		{
			"cronjob-template",
			&request.CronJobTemplate,
		},
		{
			"prerun-script",
			&request.PreRunScript,
		},
		{
			"postrun-script",
			&request.PostRunScript,
		},
		{
			"scraper-template",
			&request.ScraperTemplate,
		},
		{
			"pvc-template",
			&request.PvcTemplate,
		},
	}

	for _, value := range values {
		if cmd.Flag(value.source).Changed {
			data := ""
			name := cmd.Flag(value.source).Value.String()
			if name != "" {
				b, err := os.ReadFile(name)
				if err != nil {
					return nil, err
				}

				data = string(b)
			}

			*value.destination = &data
			nonEmpty = true
		}
	}

	if cmd.Flag("mount-configmap").Changed || cmd.Flag("variable-configmap").Changed {
		envConfigMaps, _, err := newEnvReferencesFromFlags(cmd)
		if err != nil {
			return nil, err
		}
		request.EnvConfigMaps = &envConfigMaps
		nonEmpty = true
	}

	if cmd.Flag("mount-secret").Changed || cmd.Flag("variable-secret").Changed {
		_, envSecrets, err := newEnvReferencesFromFlags(cmd)
		if err != nil {
			return nil, err
		}
		request.EnvSecrets = &envSecrets
		nonEmpty = true
	}

	if cmd.Flag("execute-postrun-script-before-scraping").Changed {
		executePostRunScriptBeforeScraping, err := cmd.Flags().GetBool("execute-postrun-script-before-scraping")
		if err != nil {
			return nil, err
		}
		request.ExecutePostRunScriptBeforeScraping = &executePostRunScriptBeforeScraping
		nonEmpty = true
	}

	if cmd.Flag("source-scripts").Changed {
		sourceScripts, err := cmd.Flags().GetBool("source-scripts")
		if err != nil {
			return nil, err
		}
		request.SourceScripts = &sourceScripts
		nonEmpty = true
	}

	artifactRequest, err := newArtifactUpdateRequestFromFlags(cmd)
	if err != nil {
		return nil, err
	}

	var emptyArtifactRequest = &testkube.ArtifactUpdateRequest{}
	if artifactRequest != nil {
		request.ArtifactRequest = &artifactRequest
		nonEmpty = true
	} else {
		request.ArtifactRequest = &emptyArtifactRequest
	}

	slavePodRequest, err := newSlavePodUpdateRequestFromFlags(cmd)
	if err != nil {
		return nil, err
	}

	var emptyPodRequest = &testkube.PodUpdateRequest{}
	if slavePodRequest != nil {
		request.SlavePodRequest = &slavePodRequest
		nonEmpty = true
	} else {
		request.SlavePodRequest = &emptyPodRequest
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
		{
			"artifact-storage-bucket",
			&request.StorageBucket,
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

	if cmd.Flag("artifact-mask").Changed {
		masks, err := cmd.Flags().GetStringArray("artifact-mask")
		if err != nil {
			return nil, err
		}

		request.Masks = &masks
		nonEmpty = true
	}

	if cmd.Flag("artifact-omit-folder-per-execution").Changed {
		value, err := cmd.Flags().GetBool("artifact-omit-folder-per-execution")
		if err != nil {
			return nil, err
		}

		request.OmitFolderPerExecution = &value
		nonEmpty = true
	}

	if cmd.Flag("artifact-shared-between-pods").Changed {
		value, err := cmd.Flags().GetBool("artifact-shared-between-pods")
		if err != nil {
			return nil, err
		}

		request.SharedBetweenPods = &value
		nonEmpty = true
	}

	if nonEmpty {
		return request, nil
	}

	return nil, nil
}

func newSlavePodUpdateRequestFromFlags(cmd *cobra.Command) (request *testkube.PodUpdateRequest, err error) {
	var nonEmpty bool
	request = &testkube.PodUpdateRequest{}
	if cmd.Flag("slave-pod-template-reference").Changed {
		value := cmd.Flag("slave-pod-template-reference").Value.String()
		request.PodTemplateReference = &value
		nonEmpty = true
	}

	if cmd.Flag("slave-pod-template").Changed {
		value := cmd.Flag("slave-pod-template").Value.String()
		b, err := os.ReadFile(value)
		if err != nil {
			return nil, err
		}

		data := string(b)
		request.PodTemplate = &data
		nonEmpty = true
	}

	if cmd.Flag("slave-pod-requests-cpu").Changed {
		value := cmd.Flag("slave-pod-requests-cpu").Value.String()
		if request.Resources == nil {
			data := &testkube.PodResourcesUpdateRequest{}
			request.Resources = &data
		}

		if (*request.Resources).Requests == nil {
			(*request.Resources).Requests = &testkube.ResourceUpdateRequest{}
		}

		(*(*request.Resources).Requests).Cpu = &value
		nonEmpty = true
	}

	if cmd.Flag("slave-pod-requests-memory").Changed {
		value := cmd.Flag("slave-pod-requests-memory").Value.String()
		if request.Resources == nil {
			data := &testkube.PodResourcesUpdateRequest{}
			request.Resources = &data
		}

		if (*request.Resources).Requests == nil {
			(*request.Resources).Requests = &testkube.ResourceUpdateRequest{}
		}

		(*(*request.Resources).Requests).Memory = &value
		nonEmpty = true
	}

	if cmd.Flag("slave-pod-limits-cpu").Changed {
		value := cmd.Flag("slave-pod-limits-cpu").Value.String()
		if request.Resources == nil {
			data := &testkube.PodResourcesUpdateRequest{}
			request.Resources = &data
		}

		if (*request.Resources).Limits == nil {
			(*request.Resources).Limits = &testkube.ResourceUpdateRequest{}
		}

		(*(*request.Resources).Limits).Cpu = &value
		nonEmpty = true
	}

	if cmd.Flag("slave-pod-limits-memory").Changed {
		value := cmd.Flag("slave-pod-limits-memory").Value.String()
		if request.Resources == nil {
			data := &testkube.PodResourcesUpdateRequest{}
			request.Resources = &data
		}

		if (*request.Resources).Limits == nil {
			(*request.Resources).Limits = &testkube.ResourceUpdateRequest{}
		}

		(*(*request.Resources).Limits).Memory = &value
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

// isFileTooBigForCLI checks the file size found on path and compares it with maxArgSize
func isFileTooBigForCLI(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("could not open file %s: %w", path, err)
	}
	defer func() {
		err := f.Close()
		if err != nil {
			output.PrintLog(fmt.Sprintf("%s could not close file %s: %v", ui.IconWarning, f.Name(), err))
		}
	}()

	fileInfo, err := f.Stat()
	if err != nil {
		return false, fmt.Errorf("could not get info on file %s: %w", path, err)
	}

	return fileInfo.Size() < maxArgSize, nil
}

// PrepareVariablesFile reads variables file, or if the file size is too big
// it uploads them
func PrepareVariablesFile(client client.Client, parentName string, parentType client.TestingType, filePath string, timeout time.Duration) (string, bool, error) {
	isFileSmall, err := isFileTooBigForCLI(filePath)
	if err != nil {
		return "", false, fmt.Errorf("could not determine if variables file %s needs to be uploaded: %w", filePath, err)
	}

	b, err := os.ReadFile(filePath)
	if err != nil {
		return "", false, fmt.Errorf("could not read file %s: %w", filePath, err)
	}
	if isFileSmall {
		return string(b), false, nil
	}

	fileName := path.Base(filePath)

	err = client.UploadFile(parentName, parentType, fileName, b, timeout)
	if err != nil {
		return "", false, fmt.Errorf("could not upload variables file for %v with name %s: %w", parentType, parentName, err)
	}
	return fileName, true, nil
}
