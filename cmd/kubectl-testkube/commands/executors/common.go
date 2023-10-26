package executors

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

// NewUpsertExecutorOptionsFromFlags creates upsert executor options from command flags
func NewUpsertExecutorOptionsFromFlags(cmd *cobra.Command) (options apiClient.UpsertExecutorOptions, err error) {
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

	jobTemplateReference := cmd.Flag("job-template-reference").Value.String()
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

	tooltips, err := cmd.Flags().GetStringToString("tooltip")
	if err != nil {
		return options, err
	}

	contentTypes, err := cmd.Flags().GetStringArray("content-type")
	if err != nil {
		return options, err
	}

	iconURI := cmd.Flag("icon-uri").Value.String()
	docsURI := cmd.Flag("docs-uri").Value.String()
	useDataDirAsWorkingDir, err := cmd.Flags().GetBool("use-data-dir-as-working-dir")
	if err != nil {
		return options, err
	}

	var meta *testkube.ExecutorMeta
	if iconURI != "" || docsURI != "" || len(tooltips) != 0 {
		meta = &testkube.ExecutorMeta{
			IconURI:  iconURI,
			DocsURI:  docsURI,
			Tooltips: tooltips,
		}
	}

	options = apiClient.UpsertExecutorOptions{
		Name:                   name,
		Types:                  types,
		ExecutorType:           executorType,
		Image:                  image,
		ImagePullSecrets:       imageSecrets,
		Command:                command,
		Args:                   executorArgs,
		Uri:                    uri,
		ContentTypes:           contentTypes,
		JobTemplate:            jobTemplateContent,
		JobTemplateReference:   jobTemplateReference,
		Features:               features,
		Labels:                 labels,
		Meta:                   meta,
		UseDataDirAsWorkingDir: useDataDirAsWorkingDir,
	}

	return options, nil
}

// NewUpsertExecutorOptionsFromFlags creates update executor options fom command flags
func NewUpdateExecutorOptionsFromFlags(cmd *cobra.Command) (options apiClient.UpdateExecutorOptions, err error) {
	var fields = []struct {
		name        string
		destination **string
	}{
		{
			"name",
			&options.Name,
		},
		{
			"executor-type",
			&options.ExecutorType,
		},
		{
			"uri",
			&options.Uri,
		},
		{
			"image",
			&options.Image,
		},
		{
			"job-template-reference",
			&options.JobTemplateReference,
		},
	}

	for _, field := range fields {
		if cmd.Flag(field.name).Changed {
			value := cmd.Flag(field.name).Value.String()
			*field.destination = &value
		}
	}

	var slices = []struct {
		name        string
		destination **[]string
	}{
		{
			"types",
			&options.Types,
		},
		{
			"command",
			&options.Command,
		},
		{
			"args",
			&options.Args,
		},
		{
			"feature",
			&options.Features,
		},
		{
			"content-type",
			&options.ContentTypes,
		},
	}

	for _, slice := range slices {
		if cmd.Flag(slice.name).Changed {
			value, err := cmd.Flags().GetStringArray(slice.name)
			if err != nil {
				return options, err
			}

			*slice.destination = &value
		}
	}

	if cmd.Flag("job-template").Changed {
		jobTemplate := cmd.Flag("job-template").Value.String()
		b, err := os.ReadFile(jobTemplate)
		if err != nil {
			return options, fmt.Errorf("reading job template %w", err)
		}

		value := string(b)
		options.JobTemplate = &value
	}

	if cmd.Flag("image-pull-secrets").Changed {
		imagePullSecretNames, err := cmd.Flags().GetStringArray("image-pull-secrets")
		if err != nil {
			return options, err
		}

		var imageSecrets []testkube.LocalObjectReference
		for _, secretName := range imagePullSecretNames {
			imageSecrets = append(imageSecrets, testkube.LocalObjectReference{Name: secretName})
		}

		options.ImagePullSecrets = &imageSecrets
	}

	if cmd.Flag("label").Changed {
		labels, err := cmd.Flags().GetStringToString("label")
		if err != nil {
			return options, err
		}

		options.Labels = &labels
	}

	if cmd.Flag("icon-uri").Changed || cmd.Flag("docs-uri").Changed || cmd.Flag("tooltip").Changed {
		meta := &testkube.ExecutorMetaUpdate{}
		if cmd.Flag("icon-uri").Changed {
			value := cmd.Flag("icon-uri").Value.String()
			meta.IconURI = &value
		}

		if cmd.Flag("docs-uri").Changed {
			value := cmd.Flag("docs-uri").Value.String()
			meta.DocsURI = &value
		}

		if cmd.Flag("tooltip").Changed {
			tooltips, err := cmd.Flags().GetStringToString("tooltip")
			if err != nil {
				return options, err
			}

			meta.Tooltips = &tooltips
		}

		options.Meta = &meta
	}

	if cmd.Flag("use-data-dir-as-working-dir").Changed {
		value, err := cmd.Flags().GetBool("use-data-dir-as-working-dir")
		if err != nil {
			return options, err
		}

		options.UseDataDirAsWorkingDir = &value
	}

	return options, nil
}
