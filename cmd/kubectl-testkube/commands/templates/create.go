package templates

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	apiv1 "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/crd"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewCreateTemplateCmd() *cobra.Command {
	var (
		name         string
		templateType string
		labels       map[string]string
		body         string
		update       bool
	)

	cmd := &cobra.Command{
		Use:     "template",
		Aliases: []string{"tp"},
		Short:   "Create a new Template.",
		Long:    `Create a new Template Custom Resource.`,
		Run: func(cmd *cobra.Command, args []string) {
			crdOnly, err := strconv.ParseBool(cmd.Flag("crd-only").Value.String())
			ui.ExitOnError("parsing flag value", err)

			if name == "" {
				ui.Failf("pass valid name (in '--name' flag)")
			}

			namespace := cmd.Flag("namespace").Value.String()
			var client apiv1.Client
			if !crdOnly {
				client, namespace, err = common.GetClient(cmd)
				ui.ExitOnError("getting client", err)

				template, _ := client.GetTemplate(name)
				if name == template.Name {
					if cmd.Flag("update").Changed {
						if !update {
							ui.Failf("Template with name '%s' already exists in namespace %s, ", template.Name, namespace)
						}
					} else {
						ok := ui.Confirm(fmt.Sprintf("Template with name '%s' already exists in namespace %s, ", template.Name, namespace) +
							"do you want to overwrite it?")
						if !ok {
							ui.Failf("Template creation was aborted")
						}
					}

					options, err := NewUpdateTemplateOptionsFromFlags(cmd)
					ui.ExitOnError("getting template options", err)

					_, err = client.UpdateTemplate(options)
					ui.ExitOnError("updating template "+name+" in namespace "+namespace, err)

					ui.SuccessAndExit("Template updated", name)
				}
			}

			options, err := NewCreateTemplateOptionsFromFlags(cmd)
			ui.ExitOnError("getting template options", err)

			if !crdOnly {
				_, err := client.CreateTemplate(options)
				ui.ExitOnError("creating template "+name+" in namespace "+namespace, err)

				ui.Success("Template created", name)
			} else {
				if options.Body != "" {
					options.Body = fmt.Sprintf("%q", options.Body)
				}

				data, err := crd.ExecuteTemplate(crd.TemplateTemplate, options)
				ui.ExitOnError("executing crd template", err)

				ui.Info(data)
			}
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique template name - mandatory")
	cmd.Flags().StringVarP(&templateType, "template-type", "", "", "template type one of job|container|cronjob|scraper|pvc|webhook")
	cmd.Flags().StringToStringVarP(&labels, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().StringVarP(&body, "body", "", "", "a path to template file to use as template body")
	cmd.Flags().BoolVar(&update, "update", false, "update, if template already exists")

	return cmd
}
