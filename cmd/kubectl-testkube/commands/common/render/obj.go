package render

import (
	"io"

	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func Obj(cmd *cobra.Command, obj interface{}, w io.Writer, renderer ...CliObjRenderer) error {
	outputType := OutputType(cmd.Flag("output").Value.String())

	switch outputType {
	case OutputPretty:
		if len(renderer) > 0 { // if custom renderer is set render using custom pretty renderer
			return renderer[0](ui.NewUI(ui.Verbose, w), obj)
		}
		return RenderYaml(obj, w) // fallback to yaml
	case OutputYAML:
		return RenderYaml(obj, w)
	case OutputJSON:
		return RenderJSON(obj, w)
	case OutputGoTemplate:
		tpl := cmd.Flag("go-template").Value.String()
		return RenderGoTemplate(obj, w, tpl)
	default:
		return RenderYaml(obj, w)
	}
}
