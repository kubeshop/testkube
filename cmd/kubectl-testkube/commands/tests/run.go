package tests

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	apiv1 "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewRunTestCmd() *cobra.Command {
	var (
		name                               string
		image                              string
		iterations                         int
		watchEnabled                       bool
		binaryArgs                         []string
		variables                          []string
		secretVariables                    []string
		variablesFile                      string
		downloadArtifactsEnabled           bool
		downloadDir                        string
		envs                               map[string]string
		secretEnvs                         map[string]string
		selectors                          []string
		concurrencyLevel                   int
		httpProxy, httpsProxy              string
		executionLabels                    map[string]string
		secretVariableReferences           map[string]string
		copyFiles                          []string
		artifactStorageClassName           string
		artifactVolumeMountPath            string
		artifactDirs                       []string
		artifactMasks                      []string
		jobTemplate                        string
		jobTemplateReference               string
		gitBranch                          string
		gitCommit                          string
		gitPath                            string
		gitWorkingDir                      string
		preRunScript                       string
		postRunScript                      string
		executePostRunScriptBeforeScraping bool
		sourceScripts                      bool
		scraperTemplate                    string
		scraperTemplateReference           string
		pvcTemplate                        string
		pvcTemplateReference               string
		negativeTest                       bool
		mountConfigMaps                    map[string]string
		variableConfigMaps                 []string
		mountSecrets                       map[string]string
		variableSecrets                    []string
		uploadTimeout                      string
		format                             string
		masks                              []string
		runningContext                     string
		command                            []string
		argsMode                           string
		artifactStorageBucket              string
		artifactOmitFolderPerExecution     bool
		artifactSharedBetweenPods          bool
		silentMode                         bool
		slavePodRequestsCpu                string
		slavePodRequestsMemory             string
		slavePodLimitsCpu                  string
		slavePodLimitsMemory               string
		slavePodTemplate                   string
		slavePodTemplateReference          string
		executionNamespace                 string
	)

	cmd := &cobra.Command{
		Use:     "test <testName>",
		Aliases: []string{"t"},
		Short:   "Starts new test",
		Long:    `Starts new test based on Test Custom Resource name, returns results to console`,
		Run: func(cmd *cobra.Command, args []string) {
			envs, err := cmd.Flags().GetStringToString("env")
			ui.WarnOnError("getting envs", err)

			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			info, err := client.GetServerInfo()
			ui.ExitOnError("getting server info", err)

			variables, err := common.CreateVariables(cmd, info.DisableSecretCreation)
			ui.WarnOnError("getting variables", err)

			executorArgs, err := testkube.PrepareExecutorArgs(binaryArgs)
			ui.ExitOnError("getting args", err)

			envConfigMaps, envSecrets, err := newEnvReferencesFromFlags(cmd)
			ui.WarnOnError("getting env config maps and secrets", err)

			mode := ""
			if cmd.Flag("args-mode").Changed {
				mode = argsMode
			}

			options := apiv1.ExecuteTestOptions{
				ExecutionVariables:         variables,
				ExecutionLabels:            executionLabels,
				Command:                    command,
				Args:                       executorArgs,
				ArgsMode:                   mode,
				SecretEnvs:                 secretEnvs,
				HTTPProxy:                  httpProxy,
				HTTPSProxy:                 httpsProxy,
				Envs:                       envs,
				Image:                      image,
				JobTemplateReference:       jobTemplateReference,
				ScraperTemplateReference:   scraperTemplateReference,
				PvcTemplateReference:       pvcTemplateReference,
				IsNegativeTestChangedOnRun: false,
				EnvConfigMaps:              envConfigMaps,
				EnvSecrets:                 envSecrets,
				RunningContext: &testkube.RunningContext{
					Type_:   string(testkube.RunningContextTypeUserCLI),
					Context: runningContext,
				},
				ExecutePostRunScriptBeforeScraping: executePostRunScriptBeforeScraping,
				SourceScripts:                      sourceScripts,
				ExecutionNamespace:                 executionNamespace,
			}

			var fields = []struct {
				source      string
				title       string
				destination *string
			}{
				{
					jobTemplate,
					"job template",
					&options.JobTemplate,
				},
				{
					preRunScript,
					"pre run script",
					&options.PreRunScriptContent,
				},
				{
					postRunScript,
					"post run script",
					&options.PostRunScriptContent,
				},
				{
					scraperTemplate,
					"scraper template",
					&options.ScraperTemplate,
				},
				{
					pvcTemplate,
					"pvc template",
					&options.PvcTemplate,
				},
			}

			for _, field := range fields {
				if field.source != "" {
					b, err := os.ReadFile(field.source)
					ui.ExitOnError("reading "+field.title, err)
					*field.destination = string(b)
				}
			}

			var executions []testkube.Execution
			client, namespace, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			if artifactStorageClassName != "" || artifactVolumeMountPath != "" || len(artifactDirs) != 0 ||
				artifactStorageBucket != "" || artifactOmitFolderPerExecution {
				options.ArtifactRequest = &testkube.ArtifactRequest{
					StorageClassName:       artifactStorageClassName,
					VolumeMountPath:        artifactVolumeMountPath,
					Dirs:                   artifactDirs,
					StorageBucket:          artifactStorageBucket,
					OmitFolderPerExecution: artifactOmitFolderPerExecution,
				}
			}

			if cmd.Flag("negative-test").Changed {
				options.NegativeTest = negativeTest
				options.IsNegativeTestChangedOnRun = true
			}

			if gitBranch != "" || gitCommit != "" || gitPath != "" || gitWorkingDir != "" {
				options.ContentRequest = &testkube.TestContentRequest{
					Repository: &testkube.RepositoryParameters{
						Branch:     gitBranch,
						Commit:     gitCommit,
						Path:       gitPath,
						WorkingDir: gitWorkingDir,
					},
				}
			}

			if slavePodRequestsCpu != "" || slavePodRequestsMemory != "" || slavePodLimitsCpu != "" ||
				slavePodLimitsMemory != "" || slavePodTemplate != "" || slavePodTemplateReference != "" {
				options.SlavePodRequest = &testkube.PodRequest{
					PodTemplateReference: slavePodTemplateReference,
				}

				if slavePodTemplate != "" {
					b, err := os.ReadFile(slavePodTemplate)
					ui.ExitOnError("reading slave pod template", err)
					options.SlavePodRequest.PodTemplate = string(b)
				}

				if slavePodRequestsCpu != "" || slavePodRequestsMemory != "" {
					if options.SlavePodRequest.Resources == nil {
						options.SlavePodRequest.Resources = &testkube.PodResourcesRequest{}
					}

					options.SlavePodRequest.Resources.Requests = &testkube.ResourceRequest{
						Cpu:    slavePodRequestsCpu,
						Memory: slavePodRequestsMemory,
					}
				}

				if slavePodLimitsCpu != "" || slavePodLimitsMemory != "" {
					if options.SlavePodRequest.Resources == nil {
						options.SlavePodRequest.Resources = &testkube.PodResourcesRequest{}
					}

					options.SlavePodRequest.Resources.Limits = &testkube.ResourceRequest{
						Cpu:    slavePodLimitsCpu,
						Memory: slavePodLimitsMemory,
					}
				}
			}

			switch {
			case len(args) > 0:
				testName := args[0]
				namespacedName := fmt.Sprintf("%s/%s", namespace, testName)

				test, err := client.GetTest(testName)
				if err != nil {
					ui.UseStderr()
					ui.Errf("Can't get test with name '%s'. Test does not exist in namespace '%s'", testName, namespace)
					ui.Debug(err.Error())
					os.Exit(1)
				}

				var timeout time.Duration
				if uploadTimeout != "" {
					timeout, err = time.ParseDuration(uploadTimeout)
					if err != nil {
						ui.ExitOnError("invalid upload timeout duration", err)
					}
				}

				options.BucketName = uuid.New().String()
				if len(variablesFile) > 0 {
					options.ExecutionVariablesFileContent, options.IsVariablesFileUploaded, err = PrepareVariablesFile(client, options.BucketName, apiv1.Execution, variablesFile, timeout)
					if err != nil {
						ui.ExitOnError("could not prepare variables file", err)
					}
				}

				if len(copyFiles) > 0 {
					err = uploadFiles(client, options.BucketName, apiv1.Execution, copyFiles, timeout)
					ui.ExitOnError("could not upload files", err)
				}

				if len(test.Uploads) != 0 || len(copyFiles) != 0 {
					copyFileList, err := mergeCopyFiles(test.Uploads, copyFiles)
					ui.ExitOnError("could not merge files", err)

					ui.Warn("Testkube will use the following file mappings:", copyFileList...)
				}

				for i := 0; i < iterations; i++ {
					execution, err := client.ExecuteTest(testName, name, options)
					ui.ExitOnError("starting test execution "+namespacedName, err)
					executions = append(executions, execution)
				}
			case len(selectors) != 0:
				selector := strings.Join(selectors, ",")
				executions, err = client.ExecuteTests(selector, concurrencyLevel, options)
				ui.ExitOnError("starting test executions "+selector, err)
			default:
				ui.Failf("Pass Test name or labels to run by labels ")
			}

			go func() {
				<-cmd.Context().Done()
				if errors.Is(cmd.Context().Err(), context.Canceled) {
					os.Exit(0)
				}
			}()

			var execErrors []error
			for _, execution := range executions {
				printExecutionDetails(execution)

				if execution.ExecutionResult != nil && execution.ExecutionResult.ErrorMessage != "" {
					execErrors = append(execErrors, errors.New(execution.ExecutionResult.ErrorMessage))
				}

				if execution.Id != "" {
					if watchEnabled && len(args) > 0 {
						info, err := client.GetServerInfo()
						ui.ExitOnError("getting server info", err)

						if info.Features != nil && info.Features.LogsV2 {
							if err = watchLogsV2(execution.Id, silentMode, client); err != nil {
								execErrors = append(execErrors, err)
							}
						} else {
							if err = watchLogs(execution.Id, silentMode, client); err != nil {
								execErrors = append(execErrors, err)
							}
						}
					}

					execution, err = client.GetExecution(execution.Id)
					ui.ExitOnError("getting recent execution data id:"+execution.Id, err)
				}

				if err = render.RenderExecutionResult(client, &execution, false, !watchEnabled); err != nil {
					execErrors = append(execErrors, err)
				}

				if execution.Id != "" {
					if watchEnabled && len(args) > 0 {
						if downloadArtifactsEnabled && (execution.IsPassed() || execution.IsFailed()) {
							DownloadTestArtifacts(execution.Id, downloadDir, format, masks, client)
						}
					}

					uiShellWatchExecution(execution.Name)
				}

				uiShellGetExecution(execution.Name)
			}

			ui.ExitOnError("executions contain failed on errors", execErrors...)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "execution name, if empty will be autogenerated")
	cmd.Flags().StringVarP(&image, "image", "", "", "override executor container image")
	cmd.Flags().StringVarP(&variablesFile, "variables-file", "", "", "variables file path, e.g. postman env file - will be passed to executor if supported")
	cmd.Flags().StringArrayVarP(&variables, "variable", "v", []string{}, "execution variable passed to executor")
	cmd.Flags().StringArrayVarP(&secretVariables, "secret-variable", "s", []string{}, "execution secret variable passed to executor")
	cmd.Flags().StringArrayVar(&command, "command", []string{}, "command passed to image in executor")
	cmd.Flags().StringArrayVarP(&binaryArgs, "args", "", []string{}, "executor binary additional arguments")
	cmd.Flags().StringVarP(&argsMode, "args-mode", "", "append", "usage mode for argumnets. one of append|override|replace")
	cmd.Flags().BoolVarP(&watchEnabled, "watch", "f", false, "watch for changes after start")
	cmd.Flags().StringVar(&downloadDir, "download-dir", "artifacts", "download dir")
	cmd.Flags().BoolVarP(&downloadArtifactsEnabled, "download-artifacts", "d", false, "download artifacts automatically")
	cmd.Flags().StringToStringVarP(&envs, "env", "", map[string]string{}, "envs in a form of name1=val1 passed to executor")
	cmd.Flags().StringToStringVarP(&secretEnvs, "secret", "", map[string]string{}, "secret envs in a form of secret_key1=secret_name1 passed to executor")
	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().IntVar(&concurrencyLevel, "concurrency", 10, "concurrency level for multiple test execution")
	cmd.Flags().IntVar(&iterations, "iterations", 1, "how many times to run the test")
	cmd.Flags().StringVar(&httpProxy, "http-proxy", "", "http proxy for executor containers")
	cmd.Flags().StringVar(&httpsProxy, "https-proxy", "", "https proxy for executor containers")
	cmd.Flags().StringToStringVarP(&executionLabels, "execution-label", "", nil, "execution-label key value pair: --execution-label key1=value1")
	cmd.Flags().StringToStringVarP(&secretVariableReferences, "secret-variable-reference", "", nil, "secret variable references in a form name1=secret_name1=secret_key1")
	cmd.Flags().StringArrayVarP(&copyFiles, "copy-files", "", []string{}, "file path mappings from host to pod of form source:destination")
	cmd.Flags().StringVar(&artifactStorageClassName, "artifact-storage-class-name", "", "artifact storage class name for container executor")
	cmd.Flags().StringVar(&artifactVolumeMountPath, "artifact-volume-mount-path", "", "artifact volume mount path for container executor")
	cmd.Flags().StringArrayVarP(&artifactDirs, "artifact-dir", "", []string{}, "artifact dirs for scraping")
	cmd.Flags().StringArrayVarP(&artifactMasks, "artifact-mask", "", []string{}, "regexp to filter scraped artifacts, single or comma separated, like report/.* or .*\\.json,.*\\.js$")
	cmd.Flags().StringVar(&jobTemplate, "job-template", "", "job template file path for extensions to job template")
	cmd.Flags().StringVar(&jobTemplateReference, "job-template-reference", "", "reference to job template to use for the test")
	cmd.Flags().StringVarP(&gitBranch, "git-branch", "", "", "if uri is git repository we can set additional branch parameter")
	cmd.Flags().StringVarP(&gitCommit, "git-commit", "", "", "if uri is git repository we can use commit id (sha) parameter")
	cmd.Flags().StringVarP(&gitPath, "git-path", "", "", "if repository is big we need to define additional path to directory/file to checkout partially")
	cmd.Flags().StringVarP(&gitWorkingDir, "git-working-dir", "", "", "if repository contains multiple directories with tests (like monorepo) and one starting directory we can set working directory parameter")
	cmd.Flags().StringVarP(&preRunScript, "prerun-script", "", "", "path to script to be run before test execution")
	cmd.Flags().StringVarP(&postRunScript, "postrun-script", "", "", "path to script to be run after test execution")
	cmd.Flags().BoolVarP(&executePostRunScriptBeforeScraping, "execute-postrun-script-before-scraping", "", false, "whether to execute postrun scipt before scraping or not (prebuilt executor only)")
	cmd.Flags().BoolVarP(&sourceScripts, "source-scripts", "", false, "run scripts using source command (container executor only)")
	cmd.Flags().StringVar(&scraperTemplate, "scraper-template", "", "scraper template file path for extensions to scraper template")
	cmd.Flags().StringVar(&scraperTemplateReference, "scraper-template-reference", "", "reference to scraper template to use for the test")
	cmd.Flags().StringVar(&pvcTemplate, "pvc-template", "", "pvc template file path for extensions to pvc template")
	cmd.Flags().StringVar(&pvcTemplateReference, "pvc-template-reference", "", "reference to pvc template to use for the test")
	cmd.Flags().BoolVar(&negativeTest, "negative-test", false, "negative test, if enabled, makes failure an expected and correct test result. If the test fails the result will be set to success, and vice versa")
	cmd.Flags().StringToStringVarP(&mountConfigMaps, "mount-configmap", "", map[string]string{}, "config map value pair for mounting it to executor pod: --mount-configmap configmap_name=configmap_mountpath")
	cmd.Flags().StringArrayVar(&variableConfigMaps, "variable-configmap", []string{}, "config map name used to map all keys to basis variables")
	cmd.Flags().StringToStringVarP(&mountSecrets, "mount-secret", "", map[string]string{}, "secret value pair for mounting it to executor pod: --mount-secret secret_name=secret_mountpath")
	cmd.Flags().StringArrayVar(&variableSecrets, "variable-secret", []string{}, "secret name used to map all keys to secret variables")
	cmd.Flags().MarkDeprecated("env", "env is deprecated use variable instead")
	cmd.Flags().MarkDeprecated("secret", "secret-env is deprecated use secret-variable instead")
	cmd.Flags().StringVar(&uploadTimeout, "upload-timeout", "", "timeout to use when uploading files, example: 30s")
	cmd.Flags().StringVar(&format, "format", "folder", "data format for storing files, one of folder|archive")
	cmd.Flags().StringArrayVarP(&masks, "mask", "", []string{}, "regexp to filter downloaded files, single or comma separated, like report/.* or .*\\.json,.*\\.js$")
	cmd.Flags().StringVar(&runningContext, "context", "", "running context description for test execution")
	cmd.Flags().StringVar(&artifactStorageBucket, "artifact-storage-bucket", "", "artifact storage bucket")
	cmd.Flags().BoolVarP(&artifactOmitFolderPerExecution, "artifact-omit-folder-per-execution", "", false, "don't store artifacts in execution folder")
	cmd.Flags().BoolVarP(&artifactSharedBetweenPods, "artifact-shared-between-pods", "", false, "whether to share volume between pods")
	cmd.Flags().BoolVarP(&silentMode, "silent", "", false, "don't print intermediate test execution")
	cmd.Flags().StringVar(&slavePodRequestsCpu, "slave-pod-requests-cpu", "", "slave pod resource requests cpu")
	cmd.Flags().StringVar(&slavePodRequestsMemory, "slave-pod-requests-memory", "", "slave pod resource requests memory")
	cmd.Flags().StringVar(&slavePodLimitsCpu, "slave-pod-limits-cpu", "", "slave pod resource limits cpu")
	cmd.Flags().StringVar(&slavePodLimitsMemory, "slave-pod-limits-memory", "", "slave pod resource limits memory")
	cmd.Flags().StringVar(&slavePodTemplate, "slave-pod-template", "", "slave pod template file path for extensions to slave pod template")
	cmd.Flags().StringVar(&slavePodTemplateReference, "slave-pod-template-reference", "", "reference to slave pod template to use for the test")
	cmd.Flags().StringVar(&executionNamespace, "execution-namespace", "", "namespace for test execution (Pro edition only)")

	return cmd
}

func uiShellGetExecution(id string) {
	ui.ShellCommand(
		"Use following command to get test execution details",
		"kubectl testkube get execution "+id,
	)

	ui.NL()
}

func uiShellWatchExecution(id string) {
	ui.ShellCommand(
		"Watch test execution until complete",
		"kubectl testkube watch execution "+id,
	)

	ui.NL()
}
