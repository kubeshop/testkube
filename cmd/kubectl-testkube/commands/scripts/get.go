package scripts

import (
	"fmt"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func NewGetScriptsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "get <scriptName>",
		Aliases: []string{"g"},
		Short:   "Get script by name",
		Long:    `Getting script from given namespace - if no namespace given "testkube" namespace is used`,
		Args:    validator.ScriptName,
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()

			name := args[0]
			namespace := cmd.Flag("namespace").Value.String()

			client, _ := common.GetClient(cmd)
			script, err := client.GetScript(name, namespace)
			ui.ExitOnError("getting script "+name, err)

			out, err := yaml.Marshal(script)
			ui.ExitOnError("getting yaml ", err)

			fmt.Printf("%s\n", out)
		},
	}
}
