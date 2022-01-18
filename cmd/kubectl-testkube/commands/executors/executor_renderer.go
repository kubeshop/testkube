package executors

import (
	"encoding/json"
	"fmt"
	"io"
	"text/template"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/spf13/cobra"
)

type ExecutorRenderer interface {
	Render(result testkube.ExecutorDetails, writer io.Writer) error
}

type ExecutorJSONRenderer struct {
}

func (r ExecutorJSONRenderer) Render(result testkube.ExecutorDetails, writer io.Writer) error {
	return json.NewEncoder(writer).Encode(result)
}

type ExecutorGoTemplateRenderer struct {
	Template string
}

func (r ExecutorGoTemplateRenderer) Render(result testkube.ExecutorDetails, writer io.Writer) error {
	tmpl, err := template.New("result").Parse(r.Template)
	if err != nil {
		return err
	}

	return tmpl.Execute(writer, result)
}

type ExecutorRawRenderer struct {
}

func (r ExecutorRawRenderer) Render(executor testkube.ExecutorDetails, writer io.Writer) error {
	_, err := fmt.Fprintf(writer, "Name: %s, Image: %s\n",
		executor.Name,
		executor.Executor.Image,
	)

	return err
}

func GetExecutorRenderer(cmd *cobra.Command) ExecutorRenderer {
	output := cmd.Flag("output").Value.String()

	switch output {
	case OutputRAW:
		return ExecutorRawRenderer{}
	case OutputJSON:
		return ExecutorJSONRenderer{}
	case OutputGoTemplate:
		template := cmd.Flag("go-template").Value.String()
		return ExecutorGoTemplateRenderer{Template: template}
	default:
		return ExecutorRawRenderer{}
	}
}
