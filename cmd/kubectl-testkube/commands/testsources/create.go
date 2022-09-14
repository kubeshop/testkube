package testsources

import (
	"strconv"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	apiv1 "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/crd"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewCreateTestSourceCmd() *cobra.Command {
	var (
		name, uri string
		labels    map[string]string
	)

	cmd := &cobra.Command{
		Use:   "testsource",
		Short: "Create new TestSource",
		Long:  `Create new TestSource Custom Resource`,
		Run: func(cmd *cobra.Command, args []string) {
			crdOnly, err := strconv.ParseBool(cmd.Flag("crd-only").Value.String())
			ui.ExitOnError("parsing flag value", err)

			if name == "" {
				ui.Failf("pass valid name (in '--name' flag)")
			}

			namespace := cmd.Flag("namespace").Value.String()
			var client apiv1.Client
			if !crdOnly {
				client, namespace = common.GetClient(cmd)

				testsource, _ := client.GetTestSource(name)
				if name == testsource.Name {
					ui.Failf("TestSource with name '%s' already exists in namespace %s", name, namespace)
				}
			}

			options := apiv1.UpsertTestSourceOptions{
				Name:      name,
				Namespace: namespace,
				Uri:       uri,
				Labels:    labels,
			}

			if !crdOnly {
				_, err := client.CreateTestSource(options)
				ui.ExitOnError("creating test source "+name+" in namespace "+namespace, err)

				ui.Success("TestSource created", name)
			} else {
				data, err := crd.ExecuteTemplate(crd.TemplateTestSource, options)
				ui.ExitOnError("executing crd template", err)

				ui.Info(data)
			}
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique test source name - mandatory")
	cmd.Flags().StringVarP(&uri, "uri", "u", "", "URI which should be called when given event occurs")
	cmd.Flags().StringToStringVarP(&labels, "label", "l", nil, "label key value pair: --label key1=value1")

	return cmd
}
