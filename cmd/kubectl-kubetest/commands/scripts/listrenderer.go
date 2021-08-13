package scripts

import (
	"encoding/json"
	"io"
	"text/template"

	"github.com/kubeshop/kubetest/pkg/api/kubetest"
	"github.com/kubeshop/kubetest/pkg/ui"
	"github.com/spf13/cobra"
)

type ListRenderer interface {
	Render(list kubetest.ScriptExecutions, writer io.Writer) error
}

type JSONListRenderer struct {
}

func (r JSONListRenderer) Render(list kubetest.ScriptExecutions, writer io.Writer) error {
	return json.NewEncoder(writer).Encode(list)
}

type GoTemplateListRenderer struct {
	Template string
}

func (r GoTemplateListRenderer) Render(list kubetest.ScriptExecutions, writer io.Writer) error {
	tmpl, err := template.New("result").Parse(r.Template)
	if err != nil {
		return err
	}

	for _, execution := range list {
		err := tmpl.Execute(writer, execution)
		if err != nil {
			return err
		}

	}

	return nil
}

type RawListRenderer struct {
}

func (r RawListRenderer) Render(list kubetest.ScriptExecutions, writer io.Writer) error {
	ui.Table(list, writer)
	return nil
}

func GetListRenderer(cmd *cobra.Command) ListRenderer {
	output := cmd.Flag("output").Value.String()

	switch output {
	case OutputRAW:
		return RawListRenderer{}
	case OutputJSON:
		return JSONListRenderer{}
	case OutputGoTemplate:
		template := cmd.Flag("go-template").Value.String()
		return GoTemplateListRenderer{Template: template}
	default:
		return RawListRenderer{}
	}
}
