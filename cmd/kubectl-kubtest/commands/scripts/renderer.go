package scripts

import (
	"encoding/json"
	"fmt"
	"io"
	"text/template"

	"github.com/kubeshop/kubtest/pkg/api/kubtest"
	"github.com/spf13/cobra"
)

const (
	OutputGoTemplate = "go"
	OutputJSON       = "json"
	OutputRAW        = "raw"
)

type Renderer interface {
	Render(result kubtest.Execution, writer io.Writer) error
	Watch(result kubtest.Execution, writer io.Writer) error
}

type JSONRenderer struct {
}

func (r JSONRenderer) Render(result kubtest.Execution, writer io.Writer) error {
	return json.NewEncoder(writer).Encode(result)
}
func (r JSONRenderer) Watch(result kubtest.Execution, writer io.Writer) error {
	return r.Render(result, writer)
}

type GoTemplateRenderer struct {
	Template string
}

func (r GoTemplateRenderer) Render(result kubtest.Execution, writer io.Writer) error {
	tmpl, err := template.New("result").Parse(r.Template)
	if err != nil {
		return err
	}

	return tmpl.Execute(writer, result)
}
func (r GoTemplateRenderer) Watch(result kubtest.Execution, writer io.Writer) error {
	return r.Render(result, writer)
}

type RawRenderer struct {
}

func (r RawRenderer) Render(scriptExecution kubtest.Execution, writer io.Writer) error {
	err := r.renderDetails(scriptExecution, writer)
	if err != nil {
		return err
	}

	if scriptExecution.Result == nil {
		return fmt.Errorf("invalid script execution, want struct but got nil, please ensure executor returns valid Execution object")
	}

	if scriptExecution.Result.Result == nil {
		return fmt.Errorf("invalid execution result, want struct but got nil, please ensure executor returns valid ExecutionResult object")
	}

	result := scriptExecution.Result.Result

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

func (r RawRenderer) Watch(scriptExecution kubtest.Execution, writer io.Writer) error {
	_, err := fmt.Fprintf(writer, "Status: %s, Duration: %s\n",
		scriptExecution.Result.Status,
		scriptExecution.Result.Duration(),
	)

	return err
}

func (r RawRenderer) renderDetails(scriptExecution kubtest.Execution, writer io.Writer) error {
	_, err := fmt.Fprintf(writer, "Name: %s, Status: %s, Duration: %s\n",
		scriptExecution.Name,
		scriptExecution.Result.Status,
		scriptExecution.Result.Duration(),
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
	default:
		return RawRenderer{}
	}
}
