package scripts

import (
	"io/ioutil"
	"os"

	apiClient "github.com/kubeshop/kubtest/pkg/api/client"
	"github.com/kubeshop/kubtest/pkg/api/kubtest"
	"github.com/kubeshop/kubtest/pkg/ui"
	"github.com/spf13/cobra"
)

func init() {
	CreateScriptsCmd.Flags().StringP("name", "n", "", "unique script name - mandatory")
	CreateScriptsCmd.Flags().StringP("file", "f", "", "script file - will be read from stdin if not specified")

	// TODO - type should be autodetected
	CreateScriptsCmd.Flags().StringP("type", "t", "postman/collection", "script type (defaults to postman-collection)")

	CreateScriptsCmd.Flags().StringP("uri", "", "", "if resource need to be loaded from URI")
	CreateScriptsCmd.Flags().StringP("git-branch", "", "", "if uri is git repository we can set additional branch parameter")
	CreateScriptsCmd.Flags().StringP("git-directory", "", "", "if repository is big we need to define additional directory to checkout partially")
}

var CreateScriptsCmd = &cobra.Command{
	Use:   "create",
	Short: "Create new script",
	Long:  `Create new Script Custom Resource, `,
	Run: func(cmd *cobra.Command, args []string) {
		ui.Logo()

		name := cmd.Flag("name").Value.String()
		namespace := cmd.Flag("namespace").Value.String()
		executorType := cmd.Flag("type").Value.String()
		file := cmd.Flag("file").Value.String()
		uri := cmd.Flag("uri").Value.String()
		gitBranch := cmd.Flag("git-branch").Value.String()
		gitDir := cmd.Flag("git-directory").Value.String()

		var content []byte
		var err error

		if file != "" {
			// read script content
			content, err = ioutil.ReadFile(file)
			ui.ExitOnError("reading file"+file, err)
		} else if stat, _ := os.Stdin.Stat(); (stat.Mode() & os.ModeCharDevice) == 0 {
			content, err = ioutil.ReadAll(os.Stdin)
			ui.ExitOnError("reading stdin", err)
		}

		client := GetClient(cmd)

		script, _ := client.GetScript(name)
		if name == script.Name {
			ui.Failf("Script with name '%s' already exists in namespace %s", name, namespace)
		}

		if len(content) == 0 && uri == "" {
			ui.Failf("Empty script content. Please pass some script content to create script")
		}

		var repository *kubtest.Repository
		if uri != "" && gitBranch != "" {
			repository = &kubtest.Repository{
				Type_:     "git",
				Uri:       uri,
				Branch:    gitBranch,
				Directory: gitDir,
			}
		}

		script, err = client.CreateScript(apiClient.CreateScriptOptions{
			Name:       name,
			Type_:      executorType,
			Content:    string(content),
			Namespace:  namespace,
			Repository: repository,
		})
		ui.ExitOnError("creating script "+name+" in namespace "+namespace, err)

		ui.Success("Script created", script.Name)
	},
}
