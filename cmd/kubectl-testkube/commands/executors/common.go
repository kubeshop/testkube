package executors

import (
	"os"
	"reflect"

	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewUpsertExecutorOptionsFromFlags(cmd *cobra.Command, testLabels map[string]string) (options apiClient.UpsertExecutorOptions, err error) {
	name := cmd.Flag("name").Value.String()
	types, err := cmd.Flags().GetStringArray("types")
	if err != nil {
		return options, err
	}

	executorType := cmd.Flag("executor-type").Value.String()
	uri := cmd.Flag("uri").Value.String()
	image := cmd.Flag("image").Value.String()
	command, err := cmd.Flags().GetStringArray("command")
	if err != nil {
		return options, err
	}

	executorArgs, err := cmd.Flags().GetStringArray("args")
	if err != nil {
		return options, err
	}

	jobTemplate := cmd.Flag("job-template").Value.String()
	jobTemplateContent := ""
	if jobTemplate != "" {
		b, err := os.ReadFile(jobTemplate)
		ui.ExitOnError("reading job template", err)
		jobTemplateContent = string(b)
	}

	imagePullSecretNames, err := cmd.Flags().GetStringArray("image-pull-secrets")
	if err != nil {
		return options, err
	}

	var imageSecrets []testkube.LocalObjectReference
	for _, secretName := range imagePullSecretNames {
		imageSecrets = append(imageSecrets, testkube.LocalObjectReference{Name: secretName})
	}

	labels, err := cmd.Flags().GetStringToString("label")
	if err != nil {
		return options, err
	}

	features, err := cmd.Flags().GetStringArray("feature")
	if err != nil {
		return options, err
	}

	options = apiClient.UpsertExecutorOptions{
		Name:             name,
		Types:            types,
		ExecutorType:     executorType,
		Image:            image,
		ImagePullSecrets: imageSecrets,
		Command:          command,
		Args:             executorArgs,
		Uri:              uri,
		JobTemplate:      jobTemplateContent,
		Features:         features,
	}

	// if labels are passed and are different from the existing overwrite
	if len(labels) > 0 && !reflect.DeepEqual(testLabels, labels) {
		options.Labels = labels
	} else {
		options.Labels = testLabels
	}

	return options, nil
}
