package tests

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewUpdateTestsCmd() *cobra.Command {

	var (
		testName        string
		testNamespace   string
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
	)

	cmd := &cobra.Command{
		Use:   "test",
		Short: "Update test",
		Long:  `Update Test Custom Resource`,
		Run: func(cmd *cobra.Command, args []string) {
			var err error

			client, _ := common.GetClient(cmd)
			test, _ := client.GetTest(testName)
			if testName != test.Name {
				ui.Failf("Test with name '%s' not exists in namespace %s", testName, testNamespace)
			}

			options, err := NewUpsertTestOptionsFromFlags(cmd, test.Labels)
			ui.ExitOnError("getting test options", err)

			err = validateSchedule(options.Schedule)
			ui.ExitOnError("validating schedule", err)

			test, err = client.UpdateTest(options)
			ui.ExitOnError("updating test "+testName+" in namespace "+testNamespace, err)

			ui.Success("Test updated", testNamespace, "/", testName)
		},
	}

	cmd.Flags().StringVarP(&testName, "name", "n", "", "unique test name - mandatory")
	cmd.Flags().StringVarP(&file, "file", "f", "", "test file - will try to read content from stdin if not specified")
	cmd.Flags().StringVarP(&testContentType, "test-content-type", "", "", "content type of test one of string|file-uri|git-file|git-dir")

	cmd.Flags().StringVarP(&executorType, "type", "t", "", "test type (defaults to postman-collection)")

	cmd.Flags().StringVarP(&uri, "uri", "", "", "URI of resource - will be loaded by http GET")
	cmd.Flags().StringVarP(&gitUri, "git-uri", "", "", "Git repository uri")
	cmd.Flags().StringVarP(&gitBranch, "git-branch", "", "", "if uri is git repository we can set additional branch parameter")
	cmd.Flags().StringVarP(&gitPath, "git-path", "", "", "if repository is big we need to define additional path to directory/file to checkout partially")
	cmd.Flags().StringVarP(&gitUsername, "git-username", "", "", "if git repository is private we can use username as an auth parameter")
	cmd.Flags().StringVarP(&gitToken, "git-token", "", "", "if git repository is private we can use token as an auth parameter")
	cmd.Flags().StringToStringVarP(&labels, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().StringToStringVarP(&variables, "variable", "v", nil, "variable key value pair: -v key1=value1")
	cmd.Flags().StringToStringVarP(&secretVariables, "secret-variable", "s", nil, "secret variable key value pair: -s key1=value1")
	cmd.Flags().StringVarP(&schedule, "schedule", "", "", "test schedule in a cronjob form: * * * * *")

	return cmd
}
