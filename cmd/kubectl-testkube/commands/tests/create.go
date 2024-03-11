package tests

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
	apiv1 "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/crd"
	"github.com/kubeshop/testkube/pkg/ui"
)

// CreateCommonFlags are common flags for creating all test types
type CreateCommonFlags struct {
	ExecutorType                       string
	Labels                             map[string]string
	Variables                          []string
	SecretVariables                    []string
	Schedule                           string
	ExecutorArgs                       []string
	ArgsMode                           string
	ExecutionName                      string
	VariablesFile                      string
	Envs                               map[string]string
	SecretEnvs                         map[string]string
	HttpProxy, HttpsProxy              string
	SecretVariableReferences           map[string]string
	CopyFiles                          []string
	Image                              string
	Command                            []string
	ImagePullSecretNames               []string
	Timeout                            int64
	ArtifactStorageClassName           string
	ArtifactVolumeMountPath            string
	ArtifactDirs                       []string
	ArtifactMasks                      []string
	JobTemplate                        string
	JobTemplateReference               string
	CronJobTemplate                    string
	CronJobTemplateReference           string
	PreRunScript                       string
	PostRunScript                      string
	ExecutePostRunScriptBeforeScraping bool
	SourceScripts                      bool
	ScraperTemplate                    string
	ScraperTemplateReference           string
	PvcTemplate                        string
	PvcTemplateReference               string
	NegativeTest                       bool
	MountConfigMaps                    map[string]string
	VariableConfigMaps                 []string
	MountSecrets                       map[string]string
	VariableSecrets                    []string
	UploadTimeout                      string
	ArtifactStorageBucket              string
	ArtifactOmitFolderPerExecution     bool
	ArtifactSharedBetweenPods          bool
	Description                        string
	SlavePodRequestsCpu                string
	SlavePodRequestsMemory             string
	SlavePodLimitsCpu                  string
	SlavePodLimitsMemory               string
	SlavePodTemplate                   string
	SlavePodTemplateReference          string
	ExecutionNamespace                 string
}

// NewCreateTestsCmd is a command tp create new Test Custom Resource
func NewCreateTestsCmd() *cobra.Command {

	var (
		testName             string
		testContentType      string
		file                 string
		uri                  string
		gitUri               string
		gitBranch            string
		gitCommit            string
		gitPath              string
		gitWorkingDir        string
		gitUsername          string
		gitToken             string
		gitUsernameSecret    map[string]string
		gitTokenSecret       map[string]string
		gitCertificateSecret string
		gitAuthType          string
		sourceName           string
		flags                CreateCommonFlags
		update               bool
	)

	cmd := &cobra.Command{
		Use:     "test",
		Aliases: []string{"tests", "t"},
		Short:   "Create new Test",
		Long:    `Create new Test Custom Resource`,
		Run: func(cmd *cobra.Command, args []string) {
			crdOnly, err := strconv.ParseBool(cmd.Flag("crd-only").Value.String())
			ui.ExitOnError("parsing flag value", err)

			if testName == "" {
				ui.Failf("pass valid test name (in '--name' flag)")
			}

			namespace := cmd.Flag("namespace").Value.String()
			var client client.Client
			if !crdOnly {
				client, namespace, err = common.GetClient(cmd)
				ui.ExitOnError("getting client", err)

				test, _ := client.GetTest(testName)

				if testName == test.Name {
					if cmd.Flag("update").Changed {
						if !update {
							ui.Failf("Test with name '%s' already exists in namespace %s, ", testName, namespace)
						}
					} else {
						var ok bool
						if stat, _ := os.Stdin.Stat(); (stat.Mode() & os.ModeCharDevice) != 0 {
							ok = ui.Confirm(fmt.Sprintf("Test with name '%s' already exists in namespace %s, ", testName, namespace) +
								"do you want to overwrite it?")
						}

						if !ok {
							ui.Failf("Test creation was aborted")
						}
					}

					options, err := NewUpdateTestOptionsFromFlags(cmd)
					ui.ExitOnError("getting test options", err)

					test, err = client.UpdateTest(options)
					ui.ExitOnError("updating test "+testName+" in namespace "+namespace, err)

					ui.SuccessAndExit("Test updated", namespace, "/", testName)
				}
			}

			if cmd.Flag("git-uri") != nil {
				err = common.ValidateUpsertOptions(cmd, sourceName)
				ui.ExitOnError("validating passed flags", err)
			}

			options, err := NewUpsertTestOptionsFromFlags(cmd)
			ui.ExitOnError("getting test options", err)

			if !crdOnly {
				executors, err := client.ListExecutors("")
				ui.ExitOnError("getting available executors", err)

				contentType := ""
				if options.Content != nil {
					contentType = options.Content.Type_
				}

				err = validateExecutorTypeAndContent(options.Type_, contentType, executors)
				ui.ExitOnError("validating executor type", err)

				var timeout time.Duration
				if flags.UploadTimeout != "" {
					timeout, err = time.ParseDuration(flags.UploadTimeout)
					if err != nil {
						ui.ExitOnError("invalid upload timeout duration", err)
					}
				}

				if len(flags.VariablesFile) > 0 {
					options.ExecutionRequest.VariablesFile, options.ExecutionRequest.IsVariablesFileUploaded, err = PrepareVariablesFile(client, testName, apiv1.Test, flags.VariablesFile, timeout)
					if err != nil {
						ui.ExitOnError("could not prepare variables file", err)
					}
				}

				if len(flags.CopyFiles) > 0 {
					err := uploadFiles(client, testName, apiv1.Test, flags.CopyFiles, timeout)
					ui.ExitOnError("could not upload files", err)
				}

				_, err = client.CreateTest(options)
				ui.ExitOnError("creating test "+testName+" in namespace "+namespace, err)

				ui.Success("Test created", namespace, "/", testName)
			} else {
				(*testkube.TestUpsertRequest)(&options).QuoteTestTextFields()
				data, err := crd.ExecuteTemplate(crd.TemplateTest, options)
				ui.ExitOnError("executing crd template", err)

				ui.Info(data)
			}
		},
	}

	cmd.Flags().StringVarP(&testName, "name", "n", "", "unique test name - mandatory")
	cmd.Flags().StringVarP(&testContentType, "test-content-type", "", "", "content type of test one of string|file-uri|git")

	// create options
	cmd.Flags().StringVarP(&file, "file", "f", "", "test file - will be read from stdin if not specified")
	cmd.Flags().StringVarP(&uri, "uri", "", "", "URI of resource - will be loaded by http GET")
	cmd.Flags().StringVarP(&gitUri, "git-uri", "", "", "Git repository uri")
	cmd.Flags().StringVarP(&gitBranch, "git-branch", "", "", "if uri is git repository we can set additional branch parameter")
	cmd.Flags().StringVarP(&gitCommit, "git-commit", "", "", "if uri is git repository we can use commit id (sha) parameter")
	cmd.Flags().StringVarP(&gitPath, "git-path", "", "", "if repository is big we need to define additional path to directory/file to checkout partially")
	cmd.Flags().StringVarP(&gitWorkingDir, "git-working-dir", "", "", "if repository contains multiple directories with tests (like monorepo) and one starting directory we can set working directory parameter")
	cmd.Flags().StringVarP(&gitUsername, "git-username", "", "", "if git repository is private we can use username as an auth parameter")
	cmd.Flags().StringVarP(&gitToken, "git-token", "", "", "if git repository is private we can use token as an auth parameter")
	cmd.Flags().StringToStringVarP(&gitUsernameSecret, "git-username-secret", "", map[string]string{}, "git username secret in a form of secret_name1=secret_key1 for private repository")
	cmd.Flags().StringToStringVarP(&gitTokenSecret, "git-token-secret", "", map[string]string{}, "git token secret in a form of secret_name1=secret_key1 for private repository")
	cmd.Flags().StringVarP(&gitCertificateSecret, "git-certificate-secret", "", "", "if git repository is private we can use certificate as an auth parameter stored in a kubernetes secret name")
	cmd.Flags().StringVarP(&gitAuthType, "git-auth-type", "", "basic", "auth type for git requests one of basic|header")
	cmd.Flags().StringVarP(&sourceName, "source", "", "", "source name - will be used together with content parameters")
	cmd.Flags().BoolVar(&update, "update", false, "update, if test already exists")
	cmd.Flags().MarkDeprecated("env", "env is deprecated use variable instead")
	cmd.Flags().MarkDeprecated("secret-env", "secret-env is deprecated use secret-variable instead")

	AddCreateFlags(cmd, &flags)

	return cmd
}

// AddCreateFlags adds flags to the create command that can be used by the create from file
func AddCreateFlags(cmd *cobra.Command, flags *CreateCommonFlags) {

	cmd.Flags().StringVarP(&flags.ExecutorType, "type", "t", "", "test type")

	cmd.Flags().StringToStringVarP(&flags.Labels, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().StringArrayVarP(&flags.Variables, "variable", "v", nil, "variable key value pair: --variable key1=value1")
	cmd.Flags().StringArrayVarP(&flags.SecretVariables, "secret-variable", "s", nil, "secret variable key value pair: --secret-variable key1=value1")
	cmd.Flags().StringVarP(&flags.Schedule, "schedule", "", "", "test schedule in a cron job form: * * * * *")
	cmd.Flags().StringArrayVar(&flags.Command, "command", []string{}, "command passed to image in executor")
	cmd.Flags().StringArrayVarP(&flags.ExecutorArgs, "executor-args", "", []string{}, "executor binary additional arguments")
	cmd.Flags().StringVarP(&flags.ArgsMode, "args-mode", "", "append", "usage mode for arguments. one of append|override|replace")
	cmd.Flags().StringVarP(&flags.ExecutionName, "execution-name", "", "", "execution name, if empty will be autogenerated")
	cmd.Flags().StringVarP(&flags.VariablesFile, "variables-file", "", "", "variables file path, e.g. postman env file - will be passed to executor if supported")
	cmd.Flags().StringToStringVarP(&flags.Envs, "env", "", map[string]string{}, "envs in a form of name1=val1 passed to executor")
	cmd.Flags().StringToStringVarP(&flags.SecretEnvs, "secret-env", "", map[string]string{}, "secret envs in a form of secret_key1=secret_name1 passed to executor")
	cmd.Flags().StringVar(&flags.HttpProxy, "http-proxy", "", "http proxy for executor containers")
	cmd.Flags().StringVar(&flags.HttpsProxy, "https-proxy", "", "https proxy for executor containers")
	cmd.Flags().StringToStringVarP(&flags.SecretVariableReferences, "secret-variable-reference", "", nil, "secret variable references in a form name1=secret_name1=secret_key1")
	cmd.Flags().StringArrayVarP(&flags.CopyFiles, "copy-files", "", []string{}, "file path mappings from host to pod of form source:destination")
	cmd.Flags().StringVar(&flags.Image, "image", "", "override executor container image")
	cmd.Flags().StringArrayVar(&flags.ImagePullSecretNames, "image-pull-secrets", []string{}, "secret name used to pull the image in container executor")
	cmd.Flags().Int64Var(&flags.Timeout, "timeout", 0, "duration in seconds for test to timeout. 0 disables timeout.")
	cmd.Flags().StringVar(&flags.ArtifactStorageClassName, "artifact-storage-class-name", "", "artifact storage class name for container executor")
	cmd.Flags().StringVar(&flags.ArtifactVolumeMountPath, "artifact-volume-mount-path", "", "artifact volume mount path for container executor")
	cmd.Flags().StringArrayVarP(&flags.ArtifactDirs, "artifact-dir", "", []string{}, "artifact dirs for scraping")
	cmd.Flags().StringArrayVarP(&flags.ArtifactMasks, "artifact-mask", "", []string{}, "regexp to filter scraped artifacts, single or comma separated, like report/.* or .*\\.json,.*\\.js$")
	cmd.Flags().StringVar(&flags.JobTemplate, "job-template", "", "job template file path for extensions to job template")
	cmd.Flags().StringVar(&flags.JobTemplateReference, "job-template-reference", "", "reference to job template to use for the test")
	cmd.Flags().StringVar(&flags.CronJobTemplate, "cronjob-template", "", "cron job template file path for extensions to cron job template")
	cmd.Flags().StringVar(&flags.CronJobTemplateReference, "cronjob-template-reference", "", "reference to cron job template to use for the test")
	cmd.Flags().StringVarP(&flags.PreRunScript, "prerun-script", "", "", "path to script to be run before test execution")
	cmd.Flags().StringVarP(&flags.PostRunScript, "postrun-script", "", "", "path to script to be run after test execution")
	cmd.Flags().BoolVarP(&flags.ExecutePostRunScriptBeforeScraping, "execute-postrun-script-before-scraping", "", false, "whether to execute postrun scipt before scraping or not (prebuilt executor only)")
	cmd.Flags().BoolVarP(&flags.SourceScripts, "source-scripts", "", false, "run scripts using source command (container executor only)")
	cmd.Flags().StringVar(&flags.ScraperTemplate, "scraper-template", "", "scraper template file path for extensions to scraper template")
	cmd.Flags().StringVar(&flags.ScraperTemplateReference, "scraper-template-reference", "", "reference to scraper template to use for the test")
	cmd.Flags().StringVar(&flags.PvcTemplate, "pvc-template", "", "pvc template file path for extensions to pvc template")
	cmd.Flags().StringVar(&flags.PvcTemplateReference, "pvc-template-reference", "", "reference to pvc template to use for the test")
	cmd.Flags().BoolVar(&flags.NegativeTest, "negative-test", false, "negative test, if enabled, makes failure an expected and correct test result. If the test fails the result will be set to success, and vice versa")
	cmd.Flags().StringToStringVarP(&flags.MountConfigMaps, "mount-configmap", "", map[string]string{}, "config map value pair for mounting it to executor pod: --mount-configmap configmap_name=configmap_mountpath")
	cmd.Flags().StringArrayVar(&flags.VariableConfigMaps, "variable-configmap", []string{}, "config map name used to map all keys to basis variables")
	cmd.Flags().StringToStringVarP(&flags.MountSecrets, "mount-secret", "", map[string]string{}, "secret value pair for mounting it to executor pod: --mount-secret secret_name=secret_mountpath")
	cmd.Flags().StringArrayVar(&flags.VariableSecrets, "variable-secret", []string{}, "secret name used to map all keys to secret variables")
	cmd.Flags().StringVar(&flags.UploadTimeout, "upload-timeout", "", "timeout to use when uploading files, example: 30s")
	cmd.Flags().StringVar(&flags.ArtifactStorageBucket, "artifact-storage-bucket", "", "artifact storage bucket")
	cmd.Flags().BoolVarP(&flags.ArtifactOmitFolderPerExecution, "artifact-omit-folder-per-execution", "", false, "don't store artifacts in execution folder")
	cmd.Flags().BoolVarP(&flags.ArtifactSharedBetweenPods, "artifact-shared-between-pods", "", false, "whether to share volume between pods")
	cmd.Flags().StringVarP(&flags.Description, "description", "", "", "test description")
	cmd.Flags().StringVar(&flags.SlavePodRequestsCpu, "slave-pod-requests-cpu", "", "slave pod resource requests cpu")
	cmd.Flags().StringVar(&flags.SlavePodRequestsMemory, "slave-pod-requests-memory", "", "slave pod resource requests memory")
	cmd.Flags().StringVar(&flags.SlavePodLimitsCpu, "slave-pod-limits-cpu", "", "slave pod resource limits cpu")
	cmd.Flags().StringVar(&flags.SlavePodLimitsMemory, "slave-pod-limits-memory", "", "slave pod resource limits memory")
	cmd.Flags().StringVar(&flags.SlavePodTemplate, "slave-pod-template", "", "slave pod template file path for extensions to slave pod template")
	cmd.Flags().StringVar(&flags.SlavePodTemplateReference, "slave-pod-template-reference", "", "reference to slave pod template to use for the test")
	cmd.Flags().StringVar(&flags.ExecutionNamespace, "execution-namespace", "", "namespace for test execution (Pro edition only)")
}

func validateExecutorTypeAndContent(executorType, contentType string, executors testkube.ExecutorsDetails) error {
	typeValid := false
	executorTypes := []string{}
	contentTypes := []string{}

	for _, ed := range executors {
		executorTypes = append(executorTypes, ed.Executor.Types...)
		for _, et := range ed.Executor.Types {
			if et == executorType {
				typeValid = true
				contentTypes = ed.Executor.ContentTypes
				break
			}
		}
	}

	if !typeValid {
		return fmt.Errorf("invalid executor type '%s' use one of: %v", executorType, executorTypes)
	}

	if len(contentTypes) != 0 {
		contentValid := false
		for _, ct := range contentTypes {
			if ct == contentType {
				contentValid = true
				break
			}
		}

		if !contentValid {
			return fmt.Errorf("invalid content type '%s' use one of: %v", contentType, contentTypes)
		}
	}

	return nil
}
