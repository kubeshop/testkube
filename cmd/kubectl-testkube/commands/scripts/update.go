package scripts

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewUpdateScriptsCmd() *cobra.Command {

	var (
		scriptName        string
		scriptNamespace   string
		scriptContentType string
		file              string
		executorType      string
		uri               string
		gitUri            string
		gitBranch         string
		gitPath           string
		gitUsername       string
		gitToken          string
		tags              []string
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update script",
		Long:  `Update Script Custom Resource, `,
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()
			var err error

			client, _ := common.GetClient(cmd)
			script, _ := client.GetScript(scriptName, scriptNamespace)
			if scriptName != script.Name {
				ui.Failf("Script with name '%s' not exists in namespace %s", scriptName, scriptNamespace)
			}

			options, err := NewUpsertScriptOptionsFromFlags(cmd, script)
			ui.ExitOnError("getting script options", err)

			script, err = client.UpdateScript(options)
			ui.ExitOnError("updating script "+scriptName+" in namespace "+scriptNamespace, err)

			ui.Success("Script updated", scriptNamespace, "/", scriptName)
		},
	}

	cmd.Flags().StringVarP(&scriptName, "name", "n", "", "unique script name - mandatory")
	cmd.Flags().StringVarP(&file, "file", "f", "", "script file - will try to read content from stdin if not specified")
	cmd.Flags().StringVarP(&scriptNamespace, "script-namespace", "", "testkube", "namespace where script will be created defaults to 'testkube' namespace")
	cmd.Flags().StringVarP(&scriptContentType, "script-content-type", "", "", "content type of script one of string|file-uri|git-file|git-dir")

	cmd.Flags().StringVarP(&executorType, "type", "t", "", "script type (defaults to postman-collection)")

	cmd.Flags().StringVarP(&uri, "uri", "", "", "URI of resource - will be loaded by http GET")
	cmd.Flags().StringVarP(&gitUri, "git-uri", "", "", "Git repository uri")
	cmd.Flags().StringVarP(&gitBranch, "git-branch", "", "", "if uri is git repository we can set additional branch parameter")
	cmd.Flags().StringVarP(&gitPath, "git-path", "", "", "if repository is big we need to define additional path to directory/file to checkout partially")
	cmd.Flags().StringVarP(&gitUsername, "git-username", "", "", "if git repository is private we can use username as an auth parameter")
	cmd.Flags().StringVarP(&gitToken, "git-token", "", "", "if git repository is private we can use token as an auth parameter")
	cmd.Flags().StringSliceVar(&tags, "tags", nil, "comma separated list of tags: --tags tag1,tag2,tag3")

	return cmd
}
