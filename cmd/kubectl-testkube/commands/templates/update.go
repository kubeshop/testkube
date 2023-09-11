package templates

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
)

func UpdateTemplateCmd() *cobra.Command {
	var (
		name         string
		templateType string
		labels       map[string]string
		body         string
	)

	cmd := &cobra.Command{
		Use:     "template",
		Aliases: []string{"templates", "tp"},
		Short:   "Update Template",
		Long:    `Update Template Custom Resource.`,
		Run: func(cmd *cobra.Command, args []string) {
			if name == "" {
				ui.Failf("pass valid name (in '--name' flag)")
			}

			client, namespace, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			template, _ := client.GetTemplate(name)
			if name != template.Name {
				ui.Failf("Template with name '%s' not exists in namespace %s", name, namespace)
			}

			options, err := NewUpdateTemplateOptionsFromFlags(cmd)
			ui.ExitOnError("getting template options", err)

			_, err = client.UpdateTemplate(options)
			ui.ExitOnError("updating template "+name+" in namespace "+namespace, err)

			ui.Success("Template updated", name)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique template name - mandatory")
	cmd.Flags().StringVarP(&templateType, "template-type", "", "", "template type one of job|container|cronjob|scraper|pvc|webhook")
	cmd.Flags().StringToStringVarP(&labels, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().StringVarP(&body, "body", "", "", "a path to template file to use as template body")

	return cmd
}
