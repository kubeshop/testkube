package diagnostics

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/pkg/diagnostics"
	"github.com/kubeshop/testkube/pkg/diagnostics/renderer"
	"github.com/kubeshop/testkube/pkg/ui"
)

// New creates the diagnostics service with the renderer that the `--output` flag asks for.
// JSON results are the only thing written to stdout, so every other line of the command
// (headers, prompts, errors) goes to stderr and the output stays readable as JSON lines.
func New(cmd *cobra.Command) diagnostics.Diagnostics {
	d := diagnostics.New()
	output := render.OutputPretty
	if f := cmd.Flag("output"); f != nil {
		output = render.OutputType(f.Value.String())
	}
	switch output {
	case render.OutputPretty:
	case render.OutputJSON:
		d.Renderer = renderer.NewJSONRenderer()
		ui.UseStderr()
	default:
		ui.Failf("unsupported output %q, use one of: pretty|json", output)
	}
	return d
}
