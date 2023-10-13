package testsources

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
)

func UpdateTestSourceCmd() *cobra.Command {
	var (
		name, uri            string
		sourceType           string
		file                 string
		gitUri               string
		gitBranch            string
		gitCommit            string
		gitPath              string
		gitUsername          string
		gitToken             string
		gitWorkingDir        string
		labels               map[string]string
		gitUsernameSecret    map[string]string
		gitTokenSecret       map[string]string
		gitCertificateSecret string
		gitAuthType          string
	)

	cmd := &cobra.Command{
		Use:     "testsource",
		Aliases: []string{"testsources", "tsc"},
		Short:   "Update TestSource",
		Long:    `Update TestSource Custom Resource`,
		Run: func(cmd *cobra.Command, args []string) {
			if name == "" {
				ui.Failf("pass valid name (in '--name' flag)")
			}

			client, namespace, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			testSource, _ := client.GetTestSource(name)
			if name != testSource.Name {
				ui.Failf("Test source with name '%s' not exists in namespace %s", name, namespace)
			}

			options, err := NewUpdateTestSourceOptionsFromFlags(cmd)
			ui.ExitOnError("getting test source options", err)

			_, err = client.UpdateTestSource(options)
			ui.ExitOnError("updating test source "+name+" in namespace "+namespace, err)

			ui.Success("TestSource updated", name)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique test source name - mandatory")
	cmd.Flags().StringToStringVarP(&labels, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().StringVarP(&sourceType, "source-type", "", "", "source type of test one of string|file-uri|git")
	cmd.Flags().StringVarP(&file, "file", "f", "", "source file - will be read from stdin if not specified")
	cmd.Flags().StringVarP(&uri, "uri", "u", "", "URI which should be called to get test content")
	cmd.Flags().StringVarP(&gitUri, "git-uri", "", "", "Git repository uri")
	cmd.Flags().StringVarP(&gitBranch, "git-branch", "", "", "if uri is git repository we can set additional branch parameter")
	cmd.Flags().StringVarP(&gitCommit, "git-commit", "", "", "if uri is git repository we can use commit id (sha) parameter")
	cmd.Flags().StringVarP(&gitPath, "git-path", "", "", "if repository is big we need to define additional path to directory/file to checkout partially")
	cmd.Flags().StringVarP(&gitUsername, "git-username", "", "", "if git repository is private we can use username as an auth parameter")
	cmd.Flags().StringVarP(&gitToken, "git-token", "", "", "if git repository is private we can use token as an auth parameter")
	cmd.Flags().StringToStringVarP(&gitUsernameSecret, "git-username-secret", "", map[string]string{}, "git username secret in a form of secret_name1=secret_key1 for private repository")
	cmd.Flags().StringToStringVarP(&gitTokenSecret, "git-token-secret", "", map[string]string{}, "git token secret in a form of secret_name1=secret_key1 for private repository")
	cmd.Flags().StringVarP(&gitCertificateSecret, "git-certificate-secret", "", "", "if git repository is private we can use certificate as an auth parameter stored in a kubernetes secret name")
	cmd.Flags().StringVarP(&gitWorkingDir, "git-working-dir", "", "", "if repository contains multiple directories with tests (like monorepo) and one starting directory we can set working directory parameter")
	cmd.Flags().StringVarP(&gitAuthType, "git-auth-type", "", "basic", "auth type for git requests one of basic|header")

	return cmd
}
