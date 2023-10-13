package tests

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewUpdateTestsCmd() *cobra.Command {

	var (
		testName             string
		testContentType      string
		file                 string
		uri                  string
		gitUri               string
		gitBranch            string
		gitCommit            string
		gitPath              string
		gitUsername          string
		gitToken             string
		sourceName           string
		gitUsernameSecret    map[string]string
		gitTokenSecret       map[string]string
		gitWorkingDir        string
		gitCertificateSecret string
		gitAuthType          string
	)

	cmd := &cobra.Command{
		Use:   "test",
		Short: "Update test",
		Long:  `Update Test Custom Resource`,
		Run: func(cmd *cobra.Command, args []string) {
			var err error

			if testName == "" {
				ui.Failf("pass valid test name (in '--name' flag)")
			}

			client, namespace, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			test, _ := client.GetTest(testName)
			if testName != test.Name {
				ui.Failf("Test with name '%s' not exists in namespace %s", testName, namespace)
			}

			options, err := NewUpdateTestOptionsFromFlags(cmd)
			ui.ExitOnError("getting test options", err)

			test, err = client.UpdateTest(options)
			ui.ExitOnError("updating test "+testName+" in namespace "+namespace, err)

			ui.Success("Test updated", namespace, "/", testName)
		},
	}

	cmd.Flags().StringVarP(&testName, "name", "n", "", "unique test name - mandatory")
	cmd.Flags().StringVarP(&file, "file", "f", "", "test file - will try to read content from stdin if not specified")
	cmd.Flags().StringVarP(&testContentType, "test-content-type", "", "", "content type of test one of string|file-uri|git")
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
	cmd.Flags().MarkDeprecated("env", "env is deprecated use variable instead")
	cmd.Flags().MarkDeprecated("secret-env", "secret-env is deprecated use secret-variable instead")

	AddCreateFlags(cmd, &CreateCommonFlags{})

	return cmd
}
