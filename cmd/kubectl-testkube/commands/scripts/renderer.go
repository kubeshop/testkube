package scripts

import (
	"encoding/json"
	"fmt"
	"io"
	"text/template"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/spf13/cobra"
)

const (
	OutputGoTemplate = "go"
	OutputJSON       = "json"
	OutputRAW        = "raw"
)

type Renderer interface {
	Render(result testkube.Execution, writer io.Writer) error
	Watch(result testkube.Execution, writer io.Writer) error
}

type JSONRenderer struct {
}

func (r JSONRenderer) Render(result testkube.Execution, writer io.Writer) error {
	return json.NewEncoder(writer).Encode(result)
}
func (r JSONRenderer) Watch(result testkube.Execution, writer io.Writer) error {
	return r.Render(result, writer)
}

type GoTemplateRenderer struct {
	Template string
}

func (r GoTemplateRenderer) Render(result testkube.Execution, writer io.Writer) error {
	tmpl, err := template.New("result").Parse(r.Template)
	if err != nil {
		return err
	}

	return tmpl.Execute(writer, result)
}
func (r GoTemplateRenderer) Watch(result testkube.Execution, writer io.Writer) error {
	return r.Render(result, writer)
}

type RawRenderer struct {
}

func (r RawRenderer) Render(execution testkube.Execution, writer io.Writer) error {
	err := r.renderDetails(execution, writer)
	if err != nil {
		return err
	}

	if execution.ExecutionResult == nil {
		return fmt.Errorf("invalid script execution, want struct but got nil, please ensure executor returns valid Execution object")
	}

	result := execution.ExecutionResult

	if result.ErrorMessage != "" {
		_, err := writer.Write([]byte(result.ErrorMessage + "\n\n"))
		if err != nil {
			return err
		}
	}

	// TODO handle outputTypes
	_, err = writer.Write([]byte(result.Output))
	return err
}

func (r RawRenderer) Watch(execution testkube.Execution, writer io.Writer) error {
	_, err := fmt.Fprintf(writer, "Status: %s, Duration: %s\n",
		*execution.ExecutionResult.Status,
		execution.ExecutionResult.Duration(),
	)

	return err
}

func (r RawRenderer) renderDetails(execution testkube.Execution, writer io.Writer) error {
	_, err := fmt.Fprintf(writer, "Name: %s, Status: %s, Duration: %s\n",
		execution.Name,
		*execution.ExecutionResult.Status,
		execution.ExecutionResult.Duration(),
	)

	return err
}

func GetRenderer(cmd *cobra.Command) Renderer {
	output := cmd.Flag("output").Value.String()

	switch output {
	case OutputRAW:
		return RawRenderer{}
	case OutputJSON:
		return JSONRenderer{}
	case OutputGoTemplate:
		template := cmd.Flag("go-template").Value.String()
		return GoTemplateRenderer{Template: template}
	testkube:
		return RawRenderer{}
	}
}
