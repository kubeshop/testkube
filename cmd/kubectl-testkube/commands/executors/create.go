package executors

import (
	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewCreateExecutorCmd() *cobra.Command {
	var (
		types                                                      []string
		name, executorType, image, uri, volueMount, volumeQuantity string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create new Executor",
		Long:  `Create new Executor Custom Resource`,
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()

			var err error

			client, namespace := GetClient(cmd)

			executor, _ := client.GetExecutor(name)
			if name == executor.Name {
				ui.Failf("Executor with name '%s' already exists in namespace %s", name, namespace)
			}

			options := apiClient.CreateExecutorOptions{
				Name:            name,
				Namespace:       namespace,
				Types:           types,
				ExecutorType:    executorType,
				Image:           image,
				Uri:             uri,
				VolumeMountPath: volueMount,
				VolumeQuantity:  volumeQuantity,
			}

			_, err = client.CreateExecutor(options)
			ui.ExitOnError("creating executor "+name+" in namespace "+namespace, err)

			ui.Success("Executor created", name)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique script name - mandatory")
	cmd.Flags().StringArrayVarP(&types, "types", "t", []string{}, "types handled by exeutor")
	cmd.Flags().StringVar(&executorType, "executor-type", "job", "executor type (defaults to job)")

	cmd.Flags().StringVarP(&uri, "uri", "u", "", "if resource need to be loaded from URI")
	cmd.Flags().StringVarP(&image, "image", "i", "", "if uri is git repository we can set additional branch parameter")
	cmd.Flags().StringVarP(&volueMount, "volume-mount", "", "", "where volume should be mounted")
	cmd.Flags().StringVarP(&volumeQuantity, "volume-quantity", "", "", "how big volume should be")

	return cmd
}
