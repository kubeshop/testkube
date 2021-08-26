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
	Render(result kubtest.ScriptExecution, writer io.Writer) error
}

type JSONRenderer struct {
}

func (r JSONRenderer) Render(result kubtest.ScriptExecution, writer io.Writer) error {
	return json.NewEncoder(writer).Encode(result)
}

type GoTemplateRenderer struct {
	Template string
}

func (r GoTemplateRenderer) Render(result kubtest.ScriptExecution, writer io.Writer) error {
	tmpl, err := template.New("result").Parse(r.Template)
	if err != nil {
		return err
	}

	return tmpl.Execute(writer, result)
}

type RawRenderer struct {
}

func (r RawRenderer) Render(execution kubtest.ScriptExecution, writer io.Writer) error {
	err := r.renderDetails(execution, writer)
	if err != nil {
		return err
	}

	if execution.Execution.Result.ErrorMessage != "" {
		_, err := writer.Write([]byte(execution.Execution.Result.ErrorMessage + "\n\n"))
		if err != nil {
			return err
		}
	}

	// TODO handle output-types
	_, err = writer.Write([]byte(execution.Execution.Result.RawOutput))
	return err
}

func (r RawRenderer) renderDetails(execution kubtest.ScriptExecution, writer io.Writer) error {

	_, err := fmt.Fprintf(writer, "Name: %s,Status: %s,Duration: %s\n\n",
		execution.Name,
		execution.Execution.Status,
		execution.Execution.Duration(),
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
