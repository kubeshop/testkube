package scripts

import (
	"encoding/json"
	"fmt"
	"io"
	"text/template"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/spf13/cobra"
)

type ExecutionRenderer interface {
	Render(result testkube.Execution, writer io.Writer) error
	Watch(result testkube.Execution, writer io.Writer) error
}

type ExecutionJSONRenderer struct {
}

func (r ExecutionJSONRenderer) Render(result testkube.Execution, writer io.Writer) error {
	return json.NewEncoder(writer).Encode(result)
}
func (r ExecutionJSONRenderer) Watch(result testkube.Execution, writer io.Writer) error {
	return r.Render(result, writer)
}

type ExecutionGoTemplateRenderer struct {
	Template string
}

func (r ExecutionGoTemplateRenderer) Render(result testkube.Execution, writer io.Writer) error {
	tmpl, err := template.New("result").Parse(r.Template)
	if err != nil {
		return err
	}

	return tmpl.Execute(writer, result)
}
func (r ExecutionGoTemplateRenderer) Watch(result testkube.Execution, writer io.Writer) error {
	return r.Render(result, writer)
}

type ExecutionRawRenderer struct {
}

func (r ExecutionRawRenderer) Render(execution testkube.Execution, writer io.Writer) error {
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

func (r ExecutionRawRenderer) Watch(execution testkube.Execution, writer io.Writer) error {
	_, err := fmt.Fprintf(writer, "Status: %s, Duration: %s\n",
		*execution.ExecutionResult.Status,
		execution.Duration(),
	)

	return err
}

func (r ExecutionRawRenderer) renderDetails(execution testkube.Execution, writer io.Writer) error {
	_, err := fmt.Fprintf(writer, "Name: %s, Status: %s, Duration: %s\n",
		execution.Name,
		*execution.ExecutionResult.Status,
		execution.Duration(),
	)

	return err
}

func GetExecutionRenderer(cmd *cobra.Command) ExecutionRenderer {
	output := cmd.Flag("output").Value.String()

	switch output {
	case OutputRAW:
		return ExecutionRawRenderer{}
	case OutputJSON:
		return ExecutionJSONRenderer{}
	case OutputGoTemplate:
		template := cmd.Flag("go-template").Value.String()
		return ExecutionGoTemplateRenderer{Template: template}
	default:
		return ExecutionRawRenderer{}
	}
}
