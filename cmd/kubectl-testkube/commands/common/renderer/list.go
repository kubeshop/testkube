package renderer

import (
	"encoding/json"
	"fmt"
	"io"
	"text/template"

	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func Render(cmd *cobra.Command, obj interface{}, w io.Writer) error {
	outputType := OutputType(cmd.Flag("output").Value.String())

	switch outputType {
	case OutputPretty:
		return RenderYaml(obj, w)
	case OutputJSON:
		return RenderJSON(obj, w)
	case OutputGoTemplate:
		tpl := cmd.Flag("go-template").Value.String()
		list, ok := obj.([]interface{})
		if !ok {
			return fmt.Errorf("can't render, need list type but got: %+v", obj)
		}
		return RenderGoTemplate(list, w, tpl)
	default:
		return RenderYaml(obj, w)
	}

}

func RenderJSON(obj interface{}, w io.Writer) error {
	return json.NewEncoder(w).Encode(obj)
}

func RenderYaml(obj interface{}, w io.Writer) error {
	return yaml.NewEncoder(w).Encode(obj)
}

func RenderGoTemplate(list []interface{}, w io.Writer, tpl string) error {
	tmpl, err := template.New("result").Parse(tpl)
	if err != nil {
		return err
	}

	for _, execution := range list {
		err := tmpl.Execute(w, execution)
		if err != nil {
			return err
		}

	}

	return nil
}

func RenderTable(obj ui.TableData, w io.Writer) error {
	ui.Table(obj, w)
	return nil
}
