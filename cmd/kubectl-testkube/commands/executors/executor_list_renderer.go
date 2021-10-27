package executors

import (
	"encoding/json"
	"io"
	"text/template"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

type ExecutorListRenderer interface {
	Render(list testkube.ExecutorsDetails, writer io.Writer) error
}

type ExecutorJSONListRenderer struct {
}

func (r ExecutorJSONListRenderer) Render(list testkube.ExecutorsDetails, writer io.Writer) error {
	return json.NewEncoder(writer).Encode(list)
}

type ExecutorGoTemplateListRenderer struct {
	Template string
}

func (r ExecutorGoTemplateListRenderer) Render(list testkube.ExecutorsDetails, writer io.Writer) error {
	tmpl, err := template.New("result").Parse(r.Template)
	if err != nil {
		return err
	}

	for _, executorDetails := range list {
		err := tmpl.Execute(writer, executorDetails)
		if err != nil {
			return err
		}

	}

	return nil
}

type ExecutorRawListRenderer struct {
}

func (r ExecutorRawListRenderer) Render(list testkube.ExecutorsDetails, writer io.Writer) error {
	ui.Table(list, writer)
	return nil
}

func GetExecutorListRenderer(cmd *cobra.Command) ExecutorListRenderer {
	output := cmd.Flag("output").Value.String()

	switch output {
	case OutputRAW:
		return ExecutorRawListRenderer{}
	case OutputJSON:
		return ExecutorJSONListRenderer{}
	case OutputGoTemplate:
		template := cmd.Flag("go-template").Value.String()
		return ExecutorGoTemplateListRenderer{Template: template}
	default:
		return ExecutorRawListRenderer{}
	}
}
