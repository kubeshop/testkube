package executors

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/crd"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewCreateExecutorCmd() *cobra.Command {
	var (
		types, command, executorArgs, imagePullSecretNames, features, contentTypes          []string
		name, executorType, image, uri, jobTemplate, iconURI, docsURI, jobTemplateReference string
		labels, tooltips                                                                    map[string]string
		update, useDataDirAsWorkingDir                                                      bool
	)

	cmd := &cobra.Command{
		Use:     "executor",
		Aliases: []string{"exec", "ex"},
		Short:   "Create new Executor",
		Long:    `Create new Executor Custom Resource`,
		Run: func(cmd *cobra.Command, args []string) {
			crdOnly, err := strconv.ParseBool(cmd.Flag("crd-only").Value.String())
			ui.ExitOnError("parsing flag value", err)

			if name == "" {
				ui.Failf("pass valid name (in '--name' flag)")
			}

			namespace := cmd.Flag("namespace").Value.String()
			var client apiClient.Client
			if !crdOnly {
				client, namespace, err = common.GetClient(cmd)
				ui.ExitOnError("getting client", err)

				executor, _ := client.GetExecutor(name)
				if name == executor.Name {
					if cmd.Flag("update").Changed {
						if !update {
							ui.Failf("Executor with name '%s' already exists in namespace %s, ", executor.Name, namespace)
						}
					} else {
						ok := ui.Confirm(fmt.Sprintf("Executor with name '%s' already exists in namespace %s, ", executor.Name, namespace) +
							"do you want to overwrite it?")
						if !ok {
							ui.Failf("Executor creation was aborted")
						}
					}

					options, err := NewUpdateExecutorOptionsFromFlags(cmd)
					ui.ExitOnError("getting executor options", err)

					_, err = client.UpdateExecutor(options)
					ui.ExitOnError("updating executor "+name+" in namespace "+namespace, err)

					ui.SuccessAndExit("Executor updated", name)
				}
			}

			options, err := NewUpsertExecutorOptionsFromFlags(cmd)
			ui.ExitOnError("getting executor options", err)

			if !crdOnly {
				_, err = client.CreateExecutor(options)
				ui.ExitOnError("creating executor "+name+" in namespace "+namespace, err)

				ui.Success("Executor created", name)
			} else {
				(*testkube.ExecutorUpsertRequest)(&options).QuoteExecutorTextFields()
				data, err := crd.ExecuteTemplate(crd.TemplateExecutor, options)
				ui.ExitOnError("executing crd template", err)

				ui.Info(data)
			}
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique executor name - mandatory")
	cmd.Flags().StringArrayVarP(&types, "types", "t", []string{}, "test types handled by executor")
	cmd.Flags().StringVar(&executorType, "executor-type", "job", "executor type, container or job (defaults to job)")

	cmd.Flags().StringVarP(&uri, "uri", "u", "", "if resource need to be loaded from URI")
	cmd.Flags().StringVar(&image, "image", "", "image used for executor")
	cmd.Flags().StringArrayVar(&imagePullSecretNames, "image-pull-secrets", []string{}, "secret name used to pull the image in executor")
	cmd.Flags().StringArrayVar(&command, "command", []string{}, "command passed to image in executor")
	cmd.Flags().StringArrayVar(&executorArgs, "args", []string{}, "args passed to image in executor")
	cmd.Flags().StringVarP(&jobTemplate, "job-template", "j", "", "if executor needs to be launched using custom job specification, then a path to template file should be provided")
	cmd.Flags().StringVarP(&jobTemplateReference, "job-template-reference", "", "", "reference to job template for using with executor")
	cmd.Flags().StringToStringVarP(&labels, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().StringArrayVar(&features, "feature", []string{}, "feature provided by executor")
	cmd.Flags().StringVarP(&iconURI, "icon-uri", "", "", "URI to executor icon")
	cmd.Flags().StringVarP(&docsURI, "docs-uri", "", "", "URI to executor docs")
	cmd.Flags().StringArrayVar(&contentTypes, "content-type", []string{}, "list of supported content types for executor")
	cmd.Flags().StringToStringVarP(&tooltips, "tooltip", "", nil, "tooltip key value pair: --tooltip key1=value1")
	cmd.Flags().BoolVar(&update, "update", false, "update, if executor already exists")
	cmd.Flags().BoolVar(&useDataDirAsWorkingDir, "use-data-dir-as-working-dir", false, "use data dir as working dir for all tests")

	return cmd
}
