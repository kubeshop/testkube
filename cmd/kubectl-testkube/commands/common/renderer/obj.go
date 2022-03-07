package renderer

import (
	"encoding/json"
	"io"
	"text/template"

	"github.com/kubeshop/testkube/pkg/ui"
	"gopkg.in/yaml.v2"
)

func RenderJSON(obj interface{}, writer io.Writer) error {
	return json.NewEncoder(writer).Encode(obj)
}

func RenderYaml(obj interface{}, writer io.Writer) error {
	return yaml.NewEncoder(writer).Encode(obj)
}

func RenderGoTemplate(list []interface{}, writer io.Writer, tpl string) error {
	tmpl, err := template.New("result").Parse(tpl)
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

func RenderTable(obj ui.TableData, writer io.Writer) error {
	ui.Table(obj, writer)
	return nil
}
