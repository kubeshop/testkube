package tests

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	apiv1 "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

const WatchInterval = 2 * time.Second

func NewRunTestCmd() *cobra.Command {
	var (
		name                     string
		image                    string
		iterations               int
		watchEnabled             bool
		binaryArgs               []string
		variables                map[string]string
		secretVariables          map[string]string
		variablesFile            string
		downloadArtifactsEnabled bool
		downloadDir              string
		envs                     map[string]string
		secretEnvs               map[string]string
		selectors                []string
		concurrencyLevel         int
		httpProxy, httpsProxy    string
		executionLabels          map[string]string
		secretVariableReferences map[string]string
		copyFiles                []string
		artifactStorageClassName string
		artifactVolumeMountPath  string
		artifactDirs             []string
		jobTemplate              string
		gitBranch                string
		gitCommit                string
		gitPath                  string
		gitWorkingDir            string
		preRunScript             string
		scraperTemplate          string
		negativeTest             bool
	)

	cmd := &cobra.Command{
		Use:     "test <testName>",
		Aliases: []string{"t"},
		Short:   "Starts new test",
		Long:    `Starts new test based on Test Custom Resource name, returns results to console`,
		Run: func(cmd *cobra.Command, args []string) {
			paramsFileContent := ""
			if variablesFile != "" {
				b, err := os.ReadFile(variablesFile)
				ui.ExitOnError("reading variables file", err)
				paramsFileContent = string(b)
			}

			envs, err := cmd.Flags().GetStringToString("env")
			ui.WarnOnError("getting envs", err)

			variables, err := common.CreateVariables(cmd)
			ui.WarnOnError("getting variables", err)

			executorArgs, err := testkube.PrepareExecutorArgs(binaryArgs)
			ui.ExitOnError("getting args", err)

			err = validateArtifactRequest(artifactStorageClassName, artifactVolumeMountPath, artifactDirs)
			ui.ExitOnError("validating artifact flags", err)

			jobTemplateContent := ""
			if jobTemplate != "" {
				b, err := os.ReadFile(jobTemplate)
				ui.ExitOnError("reading job template", err)
				jobTemplateContent = string(b)
			}

			preRunScriptContent := ""
			if preRunScript != "" {
				b, err := os.ReadFile(preRunScript)
				ui.ExitOnError("reading pre run script", err)
				preRunScriptContent = string(b)
			}

			scraperTemplateContent := ""
			if scraperTemplate != "" {
				b, err := os.ReadFile(scraperTemplate)
				ui.ExitOnError("reading scraper template", err)
				scraperTemplateContent = string(b)
			}

			var executions []testkube.Execution
			client, namespace := common.GetClient(cmd)
			options := apiv1.ExecuteTestOptions{
				ExecutionVariables:            variables,
				ExecutionVariablesFileContent: paramsFileContent,
				ExecutionLabels:               executionLabels,
				Args:                          executorArgs,
				SecretEnvs:                    secretEnvs,
				HTTPProxy:                     httpProxy,
				HTTPSProxy:                    httpsProxy,
				Envs:                          envs,
				Image:                         image,
				JobTemplate:                   jobTemplateContent,
				PreRunScriptContent:           preRunScriptContent,
				ScraperTemplate:               scraperTemplateContent,
				IsNegativeTestChangedOnRun:    false,
			}

			if artifactStorageClassName != "" && artifactVolumeMountPath != "" {
				options.ArtifactRequest = &testkube.ArtifactRequest{
					StorageClassName: artifactStorageClassName,
					VolumeMountPath:  artifactVolumeMountPath,
					Dirs:             artifactDirs,
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

				if len(copyFiles) > 0 {
					options.BucketName = uuid.New().String()
					err = uploadFiles(client, options.BucketName, apiv1.Execution, copyFiles)
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

			var hasErrors bool
			for _, execution := range executions {
				printExecutionDetails(execution)

				if execution.ExecutionResult != nil && execution.ExecutionResult.ErrorMessage != "" {
					hasErrors = true
				}

				if execution.Id != "" {
					if watchEnabled && len(args) > 0 {
						watchLogs(execution.Id, client)
					}

					execution, err = client.GetExecution(execution.Id)
					ui.ExitOnError("getting recent execution data id:"+execution.Id, err)
				}

				render.RenderExecutionResult(&execution)

				if execution.Id != "" {
					if downloadArtifactsEnabled {
						DownloadArtifacts(execution.Id, downloadDir, client)
					}

					uiShellWatchExecution(execution.Name)
				}

				uiShellGetExecution(execution.Name)
			}

			if hasErrors {
				ui.ExitOnError("executions contain failed on errors")
			}
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "execution name, if empty will be autogenerated")
	cmd.Flags().StringVarP(&image, "image", "", "", "execution variable passed to executor")
	cmd.Flags().StringVarP(&variablesFile, "variables-file", "", "", "variables file path, e.g. postman env file - will be passed to executor if supported")
	cmd.Flags().StringToStringVarP(&variables, "variable", "v", map[string]string{}, "execution variable passed to executor")
	cmd.Flags().StringToStringVarP(&secretVariables, "secret-variable", "s", map[string]string{}, "execution secret variable passed to executor")
	cmd.Flags().StringArrayVarP(&binaryArgs, "args", "", []string{}, "executor binary additional arguments")
	cmd.Flags().BoolVarP(&watchEnabled, "watch", "f", false, "watch for changes after start")
	cmd.Flags().StringVar(&downloadDir, "download-dir", "artifacts", "download dir")
	cmd.Flags().BoolVarP(&downloadArtifactsEnabled, "download-artifacts", "d", false, "downlaod artifacts automatically")
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
	cmd.Flags().StringArrayVarP(&artifactDirs, "artifact-dir", "", []string{}, "artifact dirs for container executor")
	cmd.Flags().StringVar(&jobTemplate, "job-template", "", "job template file path for extensions to job template")
	cmd.Flags().StringVarP(&gitBranch, "git-branch", "", "", "if uri is git repository we can set additional branch parameter")
	cmd.Flags().StringVarP(&gitCommit, "git-commit", "", "", "if uri is git repository we can use commit id (sha) parameter")
	cmd.Flags().StringVarP(&gitPath, "git-path", "", "", "if repository is big we need to define additional path to directory/file to checkout partially")
	cmd.Flags().StringVarP(&gitWorkingDir, "git-working-dir", "", "", "if repository contains multiple directories with tests (like monorepo) and one starting directory we can set working directory parameter")
	cmd.Flags().StringVarP(&preRunScript, "prerun-script", "", "", "path to script to be run before test execution")
	cmd.Flags().StringVar(&scraperTemplate, "scraper-template", "", "scraper template file path for extensions to scraper template")
	cmd.Flags().BoolVar(&negativeTest, "negative-test", false, "negative test, if enabled, makes failure an expected and correct test result. If the test fails the result will be set to success, and vice versa")

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
