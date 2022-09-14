package testsources

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	apiv1 "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewUpdateTestSourceCmd() *cobra.Command {
	var (
		name, uri string
		labels    map[string]string
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
	cmd.Flags().StringVarP(&uri, "uri", "u", "", "URI which should be called when given event occurs")
	cmd.Flags().StringToStringVarP(&labels, "label", "l", nil, "label key value pair: --label key1=value1")

	return cmd
}
