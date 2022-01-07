package tests

import (
	"encoding/json"
	"io"
	"text/template"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

type ExecutionsListRenderer interface {
	Render(list testkube.ExecutionsResult, writer io.Writer) error
}

type ExecutionsJSONListRenderer struct {
}

func (r ExecutionsJSONListRenderer) Render(list testkube.ExecutionsResult, writer io.Writer) error {
	return json.NewEncoder(writer).Encode(list)
}

type ExecutionsGoTemplateListRenderer struct {
	Template string
}

func (r ExecutionsGoTemplateListRenderer) Render(list testkube.ExecutionsResult, writer io.Writer) error {
	tmpl, err := template.New("result").Parse(r.Template)
	if err != nil {
		return err
	}

	for _, execution := range list.Results {
		err := tmpl.Execute(writer, execution)
		if err != nil {
			return err
		}

	}

	return nil
}

type ExecutionsRawListRenderer struct {
}

func (r ExecutionsRawListRenderer) Render(list testkube.TestExecutionsResult, writer io.Writer) error {
	ui.Table(list, writer)
	return nil
}

func GetExecutionsListRenderer(cmd *cobra.Command) ExecutionsListRenderer {
	output := cmd.Flag("output").Value.String()

	switch output {
	case OutputRAW:
		return ExecutionsRawListRenderer{}
	case OutputJSON:
		return ExecutionsJSONListRenderer{}
	case OutputGoTemplate:
		template := cmd.Flag("go-template").Value.String()
		return ExecutionsGoTemplateListRenderer{Template: template}
	default:
		return ExecutionsRawListRenderer{}
	}
}
