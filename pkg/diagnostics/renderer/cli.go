package renderer

import (
	"strings"

	"github.com/kubeshop/testkube/pkg/diagnostics/validators"
	"github.com/kubeshop/testkube/pkg/ui"
)

var _ Renderer = CLIRenderer{}

func NewCLIRenderer() CLIRenderer {
	return CLIRenderer{}
}

type CLIRenderer struct {
}

func (r CLIRenderer) RenderGroupStart(message string) {
	message = strings.Replace(message, ".", " ", 1)
	ui.H2(message)
}

func (r CLIRenderer) RenderProgress(message string) {
	ui.Printf("%s", message)
}

func (r CLIRenderer) RenderResult(res validators.ValidationResult) {
	if res.Message != "" {
		ui.Warn(res.Validator + " validator status: " + res.Message)
	}

	if len(res.Errors) > 0 {
		for _, err := range res.Errors {
			ui.NL()
			ui.Errf(err.Message)
			ui.NL()
			ui.Info("Consider following suggestions before proceeding: ")
			for _, s := range err.Suggestions {
				ui.Printf("* %s", ui.LightBlue(s))
			}
			ui.NL()
			if err.DocsURI != "" {
				ui.Printf("For more details follow docs: [%s]", ui.Yellow(err.DocsURI))
			}
		}
	} else {
		ui.Success("ok")
	}

}
