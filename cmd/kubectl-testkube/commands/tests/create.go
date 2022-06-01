package tests

import (
	"fmt"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/crd"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/robfig/cron"
	"github.com/spf13/cobra"
)

// NewCreateTestsCmd is a command tp create new Test Custom Resource
func NewCreateTestsCmd() *cobra.Command {

	var (
		testName        string
		testContentType string
		file            string
		executorType    string
		uri             string
		gitUri          string
		gitBranch       string
		gitPath         string
		gitUsername     string
		gitToken        string
		labels          map[string]string
		variables       map[string]string
		secretVariables map[string]string
		schedule        string
		crdOnly         bool
	)

	cmd := &cobra.Command{
		Use:     "test",
		Aliases: []string{"tests", "t"},
		Short:   "Create new Test",
		Long:    `Create new Test Custom Resource`,
		Run: func(cmd *cobra.Command, args []string) {
			if testName == "" {
				ui.Failf("pass valid test name (in '--name' flag)")
			}

			namespace := cmd.Flag("namespace").Value.String()
			var client client.Client
			var testLabels map[string]string
			if !crdOnly {
				client, namespace = common.GetClient(cmd)
				test, _ := client.GetTest(testName)
				testLabels = test.Labels

				if testName == test.Name {
					ui.Failf("Test with name '%s' already exists in namespace %s", testName, namespace)
				}
			}

			err := validateCreateOptions(cmd)
			ui.ExitOnError("validating passed flags", err)

			options, err := NewUpsertTestOptionsFromFlags(cmd, testLabels)
			ui.ExitOnError("getting test options", err)

			if !crdOnly {
				executors, err := client.ListExecutors("")
				ui.ExitOnError("getting available executors", err)

				err = validateExecutorType(options.Type_, executors)
				ui.ExitOnError("validating executor type", err)
			}

			err = validateSchedule(options.Schedule)
			ui.ExitOnError("validating schedule", err)

			if !crdOnly {
				_, err = client.CreateTest(options)
				ui.ExitOnError("creating test "+testName+" in namespace "+namespace, err)

				ui.Success("Test created", namespace, "/", testName)
			} else {
				if options.Content != nil && options.Content.Data != "" {
					options.Content.Data = fmt.Sprintf("%q", options.Content.Data)
				}

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
	cmd.Flags().StringVarP(&gitPath, "git-path", "", "", "if repository is big we need to define additional path to directory/file to checkout partially")
	cmd.Flags().StringVarP(&gitUsername, "git-username", "", "", "if git repository is private we can use username as an auth parameter")
	cmd.Flags().StringVarP(&gitToken, "git-token", "", "", "if git repository is private we can use token as an auth parameter")
	cmd.Flags().StringToStringVarP(&labels, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().StringToStringVarP(&variables, "variable", "v", nil, "variable key value pair: --variable key1=value1")
	cmd.Flags().StringToStringVarP(&secretVariables, "secret-variable", "s", nil, "secret variable key value pair: --secret-variable key1=value1")
	cmd.Flags().StringVarP(&schedule, "schedule", "", "", "test schedule in a cronjob form: * * * * *")
	cmd.Flags().BoolVar(&crdOnly, "crd-only", false, "generate only test crd")

	return cmd
}

func validateCreateOptions(cmd *cobra.Command) error {
	gitUri := cmd.Flag("git-uri").Value.String()
	gitBranch := cmd.Flag("git-branch").Value.String()
	gitPath := cmd.Flag("git-path").Value.String()
	gitUsername := cmd.Flag("git-username").Value.String()
	gitToken := cmd.Flag("git-token").Value.String()

	file := cmd.Flag("file").Value.String()
	uri := cmd.Flag("uri").Value.String()

	hasGitParams := gitBranch != "" || gitPath != "" || gitUri != "" || gitToken != "" || gitUsername != ""

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
		if gitUri == "" {
			return fmt.Errorf("please pass valid `--git-uri` flag")
		}
		if gitBranch == "" {
			return fmt.Errorf("please pass valid `--git-branch` flag")
		}
	}

	return nil
}

func validateExecutorType(executorType string, executors testkube.ExecutorsDetails) error {
	typeValid := false
	executorTypes := []string{}

	for _, ed := range executors {
		executorTypes = append(executorTypes, ed.Executor.Types...)
		for _, et := range ed.Executor.Types {
			if et == executorType {
				typeValid = true
			}
		}
	}

	if !typeValid {
		return fmt.Errorf("invalid executor type '%s' use one of: %v", executorType, executorTypes)
	}

	return nil
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
