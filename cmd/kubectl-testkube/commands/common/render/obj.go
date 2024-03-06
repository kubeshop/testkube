package render

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
)

func Obj(cmd *cobra.Command, obj interface{}, w io.Writer, renderer ...CliObjRenderer) error {
	outputFlag := cmd.Flag("output")
	outputType := OutputPretty
	if outputFlag != nil {
		outputType = OutputType(outputFlag.Value.String())
	}

	switch outputType {
	case OutputPretty:
		if len(renderer) > 0 { // if custom renderer is set render using custom pretty renderer
			client, _, err := common.GetClient(cmd)
			if err != nil {
				return err
			}

			return renderer[0](client, ui.NewUI(ui.Verbose, w), obj)
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
