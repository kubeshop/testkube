package executors

import (
	"io/ioutil"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewCreateExecutorCmd() *cobra.Command {
	var (
		types                                       []string
		name, executorType, image, uri, jobTemplate string
	)

	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"c"},
		Short:   "Create new Executor",
		Long:    `Create new Executor Custom Resource`,
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()

			var err error

			client, namespace := common.GetClient(cmd)

			executor, _ := client.GetExecutor(name)
			if name == executor.Name {
				ui.Failf("Executor with name '%s' already exists in namespace %s", name, namespace)
			}

			jobTemplateContent := ""
			if jobTemplate != "" {
				b, err := ioutil.ReadFile(jobTemplate)
				ui.ExitOnError("reading job template", err)
				jobTemplateContent = string(b)
			}

			options := apiClient.CreateExecutorOptions{
				Name:         name,
				Namespace:    namespace,
				Types:        types,
				ExecutorType: executorType,
				Image:        image,
				Uri:          uri,
				JobTemplate:  jobTemplateContent,
			}

			_, err = client.CreateExecutor(options)
			ui.ExitOnError("creating executor "+name+" in namespace "+namespace, err)

			ui.Success("Executor created", name)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique test name - mandatory")
	cmd.Flags().StringArrayVarP(&types, "types", "t", []string{}, "types handled by executor")
	cmd.Flags().StringVar(&executorType, "executor-type", "job", "executor type (defaults to job)")

	cmd.Flags().StringVarP(&uri, "uri", "u", "", "if resource need to be loaded from URI")
	cmd.Flags().StringVarP(&image, "image", "i", "", "if uri is git repository we can set additional branch parameter")
	cmd.Flags().StringVarP(&jobTemplate, "job-template", "j", "", "if executor needs to be launched using custom job specification")

	return cmd
}
