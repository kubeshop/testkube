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
	message = strings.ToUpper(strings.Replace(message, ".", " ", -1))
	lines := strings.Repeat("=", len(message))
	ui.Printf("\n%s\n%s\n\n", ui.Green(message), ui.Yellow(lines))
}

func (r CLIRenderer) RenderProgress(message string) {
	ui.Printf("%s", message)
}

func (r CLIRenderer) RenderResult(res validators.ValidationResult) {

	ui.Printf("  %s %s: ", ui.Green(">"), res.Validator)

	if len(res.Errors) > 0 {
		ui.Printf("%s\n", ui.IconCross)

		for _, err := range res.Errors {
			ui.NL()
			ui.Printf("    %s %s\n", ui.IconError, err.Message)
			if err.Details != "" {
				ui.Printf("      %s\n", ui.LightCyan(err.Details))
			}
			if len(err.Suggestions) > 0 {
				ui.Info(ui.LightGray("      Consider following suggestions/fixes before proceeding: "))
				for _, s := range err.Suggestions {
					ui.Printf("        * %s\n", ui.LightBlue(s))
				}
			}
			if err.DocsURI != "" {
				ui.Printf("      For more details follow docs: [%s]\n", ui.Yellow(err.DocsURI))
			}
		}
	} else {
		ui.Printf("%s", ui.IconCheckMark)
	}
	ui.NL()

}
