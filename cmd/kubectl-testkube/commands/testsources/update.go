package testsources

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	apiv1 "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewUpdateTestSourceCmd() *cobra.Command {
	var (
		name, uri         string
		sourceType        string
		file              string
		gitUri            string
		gitBranch         string
		gitCommit         string
		gitPath           string
		gitUsername       string
		gitToken          string
		labels            map[string]string
		gitUsernameSecret map[string]string
		gitTokenSecret    map[string]string
	)

	cmd := &cobra.Command{
		Use:   "testsource",
		Short: "Update TestSource",
		Long:  `Update new TestSource Custom Resource`,
		Run: func(cmd *cobra.Command, args []string) {
			if name == "" {
				ui.Failf("pass valid name (in '--name' flag)")
			}

			namespace := cmd.Flag("namespace").Value.String()
			var client apiv1.Client
			client, namespace = common.GetClient(cmd)

			testsource, _ := client.GetTestSource(name)
			if name != testsource.Name {
				ui.Failf("Test source with name '%s' not exists in namespace %s", name, namespace)
			}

			options := apiv1.UpsertTestSourceOptions{
				Name:      name,
				Namespace: namespace,
				Uri:       uri,
				Labels:    labels,
			}

			_, err := client.UpdateTestSource(options)
			ui.ExitOnError("updating test source "+name+" in namespace "+namespace, err)

			ui.Success("TestSource updated", name)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique test source name - mandatory")
	cmd.Flags().StringToStringVarP(&labels, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().StringVarP(&sourceType, "source-type", "", "", "source type of test one of string|file-uri|git-file|git-dir")
	cmd.Flags().StringVarP(&file, "file", "f", "", "source file - will be read from stdin if not specified")
	cmd.Flags().StringVarP(&uri, "uri", "u", "", "URI which should be called when given event occurs")
	cmd.Flags().StringVarP(&gitUri, "git-uri", "", "", "Git repository uri")
	cmd.Flags().StringVarP(&gitBranch, "git-branch", "", "", "if uri is git repository we can set additional branch parameter")
	cmd.Flags().StringVarP(&gitCommit, "git-commit", "", "", "if uri is git repository we can use commit id (sha) parameter")
	cmd.Flags().StringVarP(&gitPath, "git-path", "", "", "if repository is big we need to define additional path to directory/file to checkout partially")
	cmd.Flags().StringVarP(&gitUsername, "git-username", "", "", "if git repository is private we can use username as an auth parameter")
	cmd.Flags().StringVarP(&gitToken, "git-token", "", "", "if git repository is private we can use token as an auth parameter")
	cmd.Flags().StringToStringVarP(&gitUsernameSecret, "git-username-secret", "", map[string]string{}, "git username secret in a form of secret_name1=secret_key1 for private repository")
	cmd.Flags().StringToStringVarP(&gitTokenSecret, "git-token-secret", "", map[string]string{}, "git token secret in a form of secret_name1=secret_key1 for private repository")

	return cmd
}
