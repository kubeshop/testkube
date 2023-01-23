package render

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/pkg/ui"
)

func List(cmd *cobra.Command, obj interface{}, w io.Writer) error {
	outputType := OutputType(cmd.Flag("output").Value.String())

	switch outputType {
	case OutputPretty:
		list, ok := obj.(ui.TableData)
		if !ok {
			return fmt.Errorf("can't render, need list of type ui.TableData but got: %T (%+v)", obj, obj)
		}
		return RenderPrettyList(list, w)
	case OutputYAML:
		return RenderYaml(obj, w)
	case OutputJSON:
		return RenderJSON(obj, w)
	case OutputGoTemplate:
		tpl := cmd.Flag("go-template").Value.String()
		list, ok := obj.([]interface{})
		if !ok {
			return fmt.Errorf("can't render, need list type but got: %+v", obj)
		}
		return RenderGoTemplateList(list, w, tpl)
	default:
		return RenderYaml(obj, w)
	}

}
