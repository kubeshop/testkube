package tests

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
	apiv1 "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/crd"
	"github.com/kubeshop/testkube/pkg/ui"
)

// NewCreateTestsCmd is a command tp create new Test Custom Resource
func NewCreateTestsCmd() *cobra.Command {

	var (
		testName                 string
		testContentType          string
		file                     string
		executorType             string
		uri                      string
		gitUri                   string
		gitBranch                string
		gitCommit                string
		gitPath                  string
		gitWorkingDir            string
		gitUsername              string
		gitToken                 string
		gitUsernameSecret        map[string]string
		gitTokenSecret           map[string]string
		gitCertificateSecret     string
		sourceName               string
		labels                   map[string]string
		variables                map[string]string
		secretVariables          map[string]string
		schedule                 string
		executorArgs             []string
		executionName            string
		variablesFile            string
		envs                     map[string]string
		secretEnvs               map[string]string
		httpProxy, httpsProxy    string
		secretVariableReferences map[string]string
		copyFiles                []string
		image                    string
		command                  []string
		imagePullSecretNames     []string
		timeout                  int64
		artifactStorageClassName string
		artifactVolumeMountPath  string
		artifactDirs             []string
		jobTemplate              string
		preRunScript             string
		scraperTemplate          string
		negativeTest             bool
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
				client, namespace = common.GetClient(cmd)
				test, _ := client.GetTest(testName)

				if testName == test.Name {
					ui.Failf("Test with name '%s' already exists in namespace %s", testName, namespace)
				}
			}

			err = validateCreateOptions(cmd)
			ui.ExitOnError("validating passed flags", err)

			err = validateArtifactRequest(artifactStorageClassName, artifactVolumeMountPath, artifactDirs)
			ui.ExitOnError("validating artifact flags", err)

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

				if len(copyFiles) > 0 {
					err := uploadFiles(client, testName, apiv1.Test, copyFiles)
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
	cmd.Flags().StringVarP(&testContentType, "test-content-type", "", "", "content type of test one of string|file-uri|git-file|git-dir")

	cmd.Flags().StringVarP(&executorType, "type", "t", "", "test type (defaults to postman/collection)")

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
	cmd.Flags().StringVarP(&sourceName, "source", "", "", "source name - will be used together with content parameters")
	cmd.Flags().StringToStringVarP(&labels, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().StringToStringVarP(&variables, "variable", "v", nil, "variable key value pair: --variable key1=value1")
	cmd.Flags().StringToStringVarP(&secretVariables, "secret-variable", "s", nil, "secret variable key value pair: --secret-variable key1=value1")
	cmd.Flags().StringVarP(&schedule, "schedule", "", "", "test schedule in a cronjob form: * * * * *")
	cmd.Flags().StringArrayVarP(&executorArgs, "executor-args", "", []string{}, "executor binary additional arguments")
	cmd.Flags().StringVarP(&executionName, "execution-name", "", "", "execution name, if empty will be autogenerated")
	cmd.Flags().StringVarP(&variablesFile, "variables-file", "", "", "variables file path, e.g. postman env file - will be passed to executor if supported")
	cmd.Flags().StringToStringVarP(&envs, "env", "", map[string]string{}, "envs in a form of name1=val1 passed to executor")
	cmd.Flags().StringToStringVarP(&secretEnvs, "secret-env", "", map[string]string{}, "secret envs in a form of secret_key1=secret_name1 passed to executor")
	cmd.Flags().StringVar(&httpProxy, "http-proxy", "", "http proxy for executor containers")
	cmd.Flags().StringVar(&httpsProxy, "https-proxy", "", "https proxy for executor containers")
	cmd.Flags().StringToStringVarP(&secretVariableReferences, "secret-variable-reference", "", nil, "secret variable references in a form name1=secret_name1=secret_key1")
	cmd.Flags().StringArrayVarP(&copyFiles, "copy-files", "", []string{}, "file path mappings from host to pod of form source:destination")
	cmd.Flags().StringVar(&image, "image", "", "image for container executor")
	cmd.Flags().StringArrayVar(&imagePullSecretNames, "image-pull-secrets", []string{}, "secret name used to pull the image in container executor")
	cmd.Flags().StringArrayVar(&command, "command", []string{}, "command passed to image in container executor")
	cmd.Flags().Int64Var(&timeout, "timeout", 0, "duration in seconds for test to timeout. 0 disables timeout.")
	cmd.Flags().StringVar(&artifactStorageClassName, "artifact-storage-class-name", "", "artifact storage class name for container executor")
	cmd.Flags().StringVar(&artifactVolumeMountPath, "artifact-volume-mount-path", "", "artifact volume mount path for container executor")
	cmd.Flags().StringArrayVarP(&artifactDirs, "artifact-dir", "", []string{}, "artifact dirs for container executor")
	cmd.Flags().StringVar(&jobTemplate, "job-template", "", "job template file path for extensions to job template")
	cmd.Flags().StringVarP(&preRunScript, "prerun-script", "", "", "path to script to be run before test execution")
	cmd.Flags().StringVar(&scraperTemplate, "scraper-template", "", "scraper template file path for extensions to scraper template")
	cmd.Flags().BoolVar(&negativeTest, "negative-test", false, "negative test, if enabled, makes failure an expected and correct test result. If the test fails the result will be set to success, and vice versa")

	return cmd
}

func validateCreateOptions(cmd *cobra.Command) error {
	gitUri := cmd.Flag("git-uri").Value.String()
	gitBranch := cmd.Flag("git-branch").Value.String()
	gitCommit := cmd.Flag("git-commit").Value.String()
	gitPath := cmd.Flag("git-path").Value.String()
	gitUsername := cmd.Flag("git-username").Value.String()
	gitToken := cmd.Flag("git-token").Value.String()
	gitUsernameSecret, err := cmd.Flags().GetStringToString("git-username-secret")
	if err != nil {
		return err
	}

	gitTokenSecret, err := cmd.Flags().GetStringToString("git-token-secret")
	if err != nil {
		return err
	}

	gitCertificateSecret, err := cmd.Flags().GetString("git-certificate-secret")
	if err != nil {
		return err
	}

	gitWorkingDir := cmd.Flag("git-working-dir").Value.String()
	file := cmd.Flag("file").Value.String()
	uri := cmd.Flag("uri").Value.String()
	sourceName := cmd.Flag("source").Value.String()

	hasGitParams := gitBranch != "" || gitCommit != "" || gitPath != "" || gitUri != "" || gitToken != "" || gitUsername != "" ||
		len(gitUsernameSecret) > 0 || len(gitTokenSecret) > 0 || gitWorkingDir != "" || gitCertificateSecret != ""

	if hasGitParams && uri != "" {
		return fmt.Errorf("found git params and `--uri` flag, please use `--git-uri` for git based repo or `--uri` without git based params")
	}
	if hasGitParams && file != "" {
		return fmt.Errorf("found git params and `--file` flag, please use `--git-uri` for git based repo or `--file` without git based params")
	}

	if file != "" && uri != "" {
		return fmt.Errorf("please pass only one of `--file` and `--uri`")
	}

	if hasGitParams {
		if gitUri == "" && sourceName == "" {
			return fmt.Errorf("please pass valid `--git-uri` flag")
		}
		if gitBranch != "" && gitCommit != "" {
			return fmt.Errorf("please pass only one of `--git-branch` or `--git-commit`")
		}
	}

	if len(gitUsernameSecret) > 1 {
		return fmt.Errorf("please pass only one secret reference for git username")
	}

	if len(gitTokenSecret) > 1 {
		return fmt.Errorf("please pass only one secret reference for git token")
	}

	if (gitUsername != "" || gitToken != "" || gitCertificateSecret != "") && (len(gitUsernameSecret) > 0 || len(gitTokenSecret) > 0) {
		return fmt.Errorf("please pass git credentials either as direct values or as secret references")
	}

	return nil
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

func validateArtifactRequest(artifactStorageClassName, artifactVolumeMountPath string, artifactDirs []string) error {
	if artifactStorageClassName != "" || artifactVolumeMountPath != "" || len(artifactDirs) != 0 {
		if artifactStorageClassName == "" || artifactVolumeMountPath == "" {
			return fmt.Errorf("both artifact storage class name and mount path should be provided")
		}
	}

	return nil
}
