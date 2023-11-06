package executors

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/crd"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewGetExecutorCmd() *cobra.Command {
	var selectors []string
	var crdOnly bool

	cmd := &cobra.Command{
		Use:     "executor [executorName]",
		Aliases: []string{"executors", "er"},
		Short:   "Gets executor details",
		Long:    `Gets executor, you can change output format`,
		Run: func(cmd *cobra.Command, args []string) {
			client, namespace, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			firstEntry := true
			if len(args) > 0 {
				name := args[0]

				executor, err := client.GetExecutor(name)
				ui.ExitOnError("getting executor: "+name, err)

				if crdOnly {
					options := mapExecutorDetailsToCreateExecutorOptions(namespace, &executor)
					(*testkube.ExecutorUpsertRequest)(options).QuoteExecutorTextFields()
					common.UIPrintCRD(crd.TemplateExecutor, options, &firstEntry)
					return
				}

				err = render.Obj(cmd, executor, os.Stdout)
				ui.ExitOnError("rendering executor", err)

			} else {
				executors, err := client.ListExecutors(strings.Join(selectors, ","))
				ui.ExitOnError("listing executors: ", err)

				if crdOnly {
					for _, executor := range executors {
						options := mapExecutorDetailsToCreateExecutorOptions(namespace, &executor)
						(*testkube.ExecutorUpsertRequest)(options).QuoteExecutorTextFields()
						common.UIPrintCRD(crd.TemplateExecutor, options, &firstEntry)
					}

					return
				}

				err = render.List(cmd, executors, os.Stdout)
				ui.ExitOnError("rendering executors", err)
			}
		},
	}

	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().BoolVar(&crdOnly, "crd-only", false, "show only test crd ")

	return cmd
}

func mapExecutorDetailsToCreateExecutorOptions(namespace string, executor *testkube.ExecutorDetails) *apiClient.UpsertExecutorOptions {
	options := &apiClient.UpsertExecutorOptions{
		Name:      executor.Name,
		Namespace: namespace,
	}

	if executor.Executor != nil {
		options.Types = executor.Executor.Types
		options.ExecutorType = executor.Executor.ExecutorType
		options.Image = executor.Executor.Image
		options.ImagePullSecrets = executor.Executor.ImagePullSecrets
		options.Command = executor.Executor.Command
		options.Args = executor.Executor.Args
		options.Uri = executor.Executor.Uri
		options.Labels = executor.Executor.Labels
		options.JobTemplateReference = executor.Executor.JobTemplateReference
		if executor.Executor.JobTemplate != "" {
			options.JobTemplate = fmt.Sprintf("%q", executor.Executor.JobTemplate)
		}

		options.Features = executor.Executor.Features
		options.ContentTypes = executor.Executor.ContentTypes
		options.Meta = executor.Executor.Meta
		options.UseDataDirAsWorkingDir = executor.Executor.UseDataDirAsWorkingDir
	}

	return options
}
