package testsuites

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	apiv1 "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

const (
	maxErrorMessageLength = 100000
)

func NewRunTestSuiteCmd() *cobra.Command {
	var (
		name                     string
		watchEnabled             bool
		variables                []string
		secretVariables          []string
		executionLabels          map[string]string
		selectors                []string
		concurrencyLevel         int
		httpProxy, httpsProxy    string
		secretVariableReferences map[string]string
		gitBranch                string
		gitCommit                string
		gitPath                  string
		gitWorkingDir            string
		runningContext           string
		jobTemplate              string
		scraperTemplate          string
		pvcTemplate              string
		jobTemplateReference     string
		scraperTemplateReference string
		pvcTemplateReference     string
		downloadArtifactsEnabled bool
		downloadDir              string
		format                   string
		masks                    []string
		silentMode               bool
	)

	cmd := &cobra.Command{
		Use:     "testsuite <testSuiteName>",
		Aliases: []string{"ts"},
		Short:   "Starts new test suite",
		Long:    `Starts new test suite based on TestSuite Custom Resource name, returns results to console`,
		Run: func(cmd *cobra.Command, args []string) {
			startTime := time.Now()
			client, namespace, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			var executions []testkube.TestSuiteExecution

			options := apiv1.ExecuteTestSuiteOptions{
				HTTPProxy:       httpProxy,
				HTTPSProxy:      httpsProxy,
				ExecutionLabels: executionLabels,
				RunningContext: &testkube.RunningContext{
					Type_:   string(testkube.RunningContextTypeUserCLI),
					Context: runningContext,
				},
				ConcurrencyLevel:         int32(concurrencyLevel),
				JobTemplateReference:     jobTemplateReference,
				ScraperTemplateReference: scraperTemplateReference,
				PvcTemplateReference:     pvcTemplateReference,
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

			info, err := client.GetServerInfo()
			ui.ExitOnError("getting server info", err)

			options.ExecutionVariables, err = common.CreateVariables(cmd, info.DisableSecretCreation)
			ui.WarnOnError("getting variables", err)

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
				testSuiteName := args[0]
				namespacedName := fmt.Sprintf("%s/%s", namespace, testSuiteName)

				execution, err := client.ExecuteTestSuite(testSuiteName, name, options)
				ui.ExitOnError("starting test suite execution "+namespacedName, err)
				executions = append(executions, execution)
			case len(selectors) != 0:
				selector := strings.Join(selectors, ",")
				executions, err = client.ExecuteTestSuites(selector, concurrencyLevel, options)
				ui.ExitOnError("starting test suite executions "+selector, err)
			default:
				ui.Failf("Pass Test suite name or labels to run by labels ")
			}

			go func() {
				<-cmd.Context().Done()
				if errors.Is(cmd.Context().Err(), context.Canceled) {
					os.Exit(0)
				}
			}()

			var execErrors []error
			for _, execution := range executions {
				if execution.IsFailed() {
					execErrors = append(execErrors, errors.New("failed execution"))
				}

				if execution.Id != "" {
					if watchEnabled && len(args) > 0 {
						watchResp := client.WatchTestSuiteExecution(execution.Id)
						for resp := range watchResp {
							ui.ExitOnError("watching test suite execution", resp.Error)
							if !silentMode {
								execution.TruncateErrorMessages(maxErrorMessageLength)
								printExecution(execution, startTime)
							}
						}
					}

					execution, err = client.GetTestSuiteExecution(execution.Id)
				}

				execution.TruncateErrorMessages(maxErrorMessageLength)
				printExecution(execution, startTime)
				ui.ExitOnError("getting recent execution data id:"+execution.Id, err)

				if err = uiPrintExecutionStatus(client, execution); err != nil {
					execErrors = append(execErrors, err)
				}

				uiShellTestSuiteGetCommandBlock(execution.Id)
				if execution.Id != "" {
					if watchEnabled && len(args) > 0 {
						if downloadArtifactsEnabled {
							DownloadArtifacts(execution.Id, downloadDir, format, masks, client)
						}
					}

					if !watchEnabled || len(args) == 0 {
						uiShellTestSuiteWatchCommandBlock(execution.Id)
					}
				}
			}

			ui.ExitOnError("executions contain failed on errors", execErrors...)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "execution name, if empty will be autogenerated")
	cmd.Flags().StringArrayVarP(&variables, "variable", "v", []string{}, "execution variables passed to executor")
	cmd.Flags().StringArrayVarP(&secretVariables, "secret-variable", "s", []string{}, "execution variables passed to executor")
	cmd.Flags().BoolVarP(&watchEnabled, "watch", "f", false, "watch for changes after start")
	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().IntVar(&concurrencyLevel, "concurrency", 10, "concurrency level for multiple test suite execution")
	cmd.Flags().StringVar(&httpProxy, "http-proxy", "", "http proxy for executor containers")
	cmd.Flags().StringVar(&httpsProxy, "https-proxy", "", "https proxy for executor containers")
	cmd.Flags().StringToStringVarP(&executionLabels, "execution-label", "", nil, "execution-label adds a label to execution in form of key value pair: --execution-label key1=value1")
	cmd.Flags().StringToStringVarP(&secretVariableReferences, "secret-variable-reference", "", nil, "secret variable references in a form name1=secret_name1=secret_key1")
	cmd.Flags().StringVarP(&gitBranch, "git-branch", "", "", "if uri is git repository we can set additional branch parameter")
	cmd.Flags().StringVarP(&gitCommit, "git-commit", "", "", "if uri is git repository we can use commit id (sha) parameter")
	cmd.Flags().StringVarP(&gitPath, "git-path", "", "", "if repository is big we need to define additional path to directory/file to checkout partially")
	cmd.Flags().StringVarP(&gitWorkingDir, "git-working-dir", "", "", "if repository contains multiple directories with tests (like monorepo) and one starting directory we can set working directory parameter")
	cmd.Flags().StringVar(&runningContext, "context", "", "running context description for test suite execution")
	cmd.Flags().StringVar(&jobTemplate, "job-template", "", "job template file path for extensions to job template")
	cmd.Flags().StringVar(&scraperTemplate, "scraper-template", "", "scraper template file path for extensions to scraper template")
	cmd.Flags().StringVar(&pvcTemplate, "pvc-template", "", "pvc template file path for extensions to pvc template")
	cmd.Flags().StringVar(&jobTemplateReference, "job-template-reference", "", "reference to job template to use for the test")
	cmd.Flags().StringVar(&scraperTemplateReference, "scraper-template-reference", "", "reference to scraper template to use for the test")
	cmd.Flags().StringVar(&pvcTemplateReference, "pvc-template-reference", "", "reference to pvc template to use for the test")
	cmd.Flags().StringVar(&downloadDir, "download-dir", "artifacts", "download dir")
	cmd.Flags().BoolVarP(&downloadArtifactsEnabled, "download-artifacts", "d", false, "download artifacts automatically")
	cmd.Flags().StringVar(&format, "format", "folder", "data format for storing files, one of folder|archive")
	cmd.Flags().StringArrayVarP(&masks, "mask", "", []string{}, "regexp to filter downloaded files, single or comma separated, like report/.* or .*\\.json,.*\\.js$")
	cmd.Flags().BoolVarP(&silentMode, "silent", "", false, "don't print intermediate test suite execution")

	return cmd
}
