package templates

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	apiv1 "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

// NewCreateTemplateOptionsFromFlags creates create template options from command flags
func NewCreateTemplateOptionsFromFlags(cmd *cobra.Command) (options apiv1.CreateTemplateOptions, err error) {
	name := cmd.Flag("name").Value.String()
	namespace := cmd.Flag("namespace").Value.String()
	if err != nil {
		return options, err
	}

	templateType := testkube.TemplateType(cmd.Flag("template-type").Value.String())

	if templateType != testkube.JOB_TemplateType && templateType != testkube.CRONJOB_TemplateType &&
		templateType != testkube.SCRAPER_TemplateType && templateType != testkube.PVC_TemplateType &&
		templateType != testkube.WEBHOOK_TemplateType && templateType != testkube.POD_TemplateType {
		ui.Failf("invalid template type: %s. use one of job|container|cronjob|scraper|pvc|webhook|pod", templateType)
	}

	body := cmd.Flag("body").Value.String()
	bodyContent := ""
	if body != "" {
		b, err := os.ReadFile(body)
		ui.ExitOnError("reading template body", err)
		bodyContent = string(b)
	}

	labels, err := cmd.Flags().GetStringToString("label")
	if err != nil {
		return options, err
	}

	options = apiv1.CreateTemplateOptions{
		Name:      name,
		Namespace: namespace,
		Type_:     &templateType,
		Labels:    labels,
		Body:      bodyContent,
	}

	return options, nil
}

// NewUpdateTemplateOptionsFromFlags creates update template options from command flags
func NewUpdateTemplateOptionsFromFlags(cmd *cobra.Command) (options apiv1.UpdateTemplateOptions, err error) {
	var fields = []struct {
		name        string
		destination **string
	}{
		{
			"name",
			&options.Name,
		},
	}

	for _, field := range fields {
		if cmd.Flag(field.name).Changed {
			value := cmd.Flag(field.name).Value.String()
			*field.destination = &value
		}
	}

	if cmd.Flag("template-type").Changed {
		templateType := testkube.TemplateType(cmd.Flag("template-type").Value.String())
		if templateType != testkube.JOB_TemplateType && templateType != testkube.CRONJOB_TemplateType &&
			templateType != testkube.SCRAPER_TemplateType && templateType != testkube.PVC_TemplateType &&
			templateType != testkube.WEBHOOK_TemplateType && templateType != testkube.POD_TemplateType {
			ui.Failf("invalid template type: %s. use one of job|container|cronjob|scraper|pvc|webhook|pod", templateType)
		}
		options.Type_ = &templateType
	}

	if cmd.Flag("body").Changed {
		body := cmd.Flag("body").Value.String()
		b, err := os.ReadFile(body)
		if err != nil {
			return options, fmt.Errorf("reading template body %w", err)
		}

		value := string(b)
		options.Body = &value
	}

	if cmd.Flag("label").Changed {
		labels, err := cmd.Flags().GetStringToString("label")
		if err != nil {
			return options, err
		}

		options.Labels = &labels
	}

	return options, nil
}
