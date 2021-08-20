package scripts

import (
	"io/ioutil"
	"os"

	"github.com/kubeshop/kubtest/pkg/ui"
	"github.com/spf13/cobra"
)

func init() {
	// TODO find a good way to handle short flags
	CreateScriptsCmd.Flags().StringP("name", "n", "", "unique script name - mandatory")
	CreateScriptsCmd.Flags().StringP("file", "f", "", "script file - will be read from stdin if not specified")

	CreateScriptsCmd.Flags().StringP("type", "t", "postman/collection", "script type (defaults to postman-collection)")
}

var CreateScriptsCmd = &cobra.Command{
	Use:   "create",
	Short: "Create new script",
	Long:  `Create new Script Custom Resource, `,
	Run: func(cmd *cobra.Command, args []string) {

		name := cmd.Flag("name").Value.String()
		namespace := cmd.Flag("namespace").Value.String()
		executorType := cmd.Flag("type").Value.String()
		file := cmd.Flag("file").Value.String()

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

		script, err = client.CreateScript(name, executorType, string(content), namespace)
		ui.ExitOnError("creating script "+name+" in namespace "+namespace, err)

		ui.Success("Script created", script.Name)
	},
}
