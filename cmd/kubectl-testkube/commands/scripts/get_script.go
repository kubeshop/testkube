package scripts

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func NewGetScriptsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "get",
		Aliases: []string{"g"},
		Short:   "Get script by name",
		Long:    `Getting script from given namespace - if no namespace given "testkube" namespace is used`,
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()

			if len(args) == 0 {
				ui.Failf("script name is not specified")
			}

			name := args[0]
			client, _ := GetClient(cmd)
			script, err := client.GetScript(name)
			ui.ExitOnError("getting script "+name, err)

			out, err := yaml.Marshal(script)
			ui.ExitOnError("getting yaml ", err)

			fmt.Printf("%s\n", out)
		},
	}
}
