package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"text/template"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
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
		execution.CalculateDuration(),
	)

	return err
}

// TODO fix this - introduce some common data interface for rendering such objects
// renderers need to be simplified and render Execution should be in one place (not many as now)
// - move all logic from execution, start, watch here to show final execution
func (r ExecutionRawRenderer) renderDetails(execution testkube.Execution, writer io.Writer) error {
	ui.Writer = writer
	uiPrintStatus(execution)
	uiShellGetExecution(execution.Id)
	return nil
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
