package ui

import "github.com/pterm/pterm"

var (
	checkInfoPrinter = pterm.Info.
				WithMessageStyle(pterm.NewStyle(pterm.FgWhite, pterm.BgDefault)).
				WithPrefix(pterm.Prefix{Text: " ️", Style: pterm.NewStyle(pterm.FgDefault, pterm.BgDefault)})

	checkOkPrinter = pterm.Info.
			WithMessageStyle(pterm.NewStyle(pterm.FgWhite, pterm.BgDefault)).
			WithPrefix(pterm.Prefix{Text: "✅", Style: pterm.NewStyle(pterm.FgDefault, pterm.BgDefault)})

	checkFailPrinter = pterm.Info.
				WithMessageStyle(pterm.NewStyle(pterm.FgRed, pterm.BgDefault)).
				WithPrefix(pterm.Prefix{Text: "❗", Style: pterm.NewStyle(pterm.FgDefault, pterm.BgDefault)})
)

func NewSpinner(t string) *pterm.SpinnerPrinter {
	s := pterm.DefaultSpinner.
		WithSequence(` ⠋ `, ` ⠹ `, ` ⠼ `, ` ⠦ `, ` ⠇ `)
	s.SuccessPrinter = checkOkPrinter
	s.InfoPrinter = checkInfoPrinter
	s.FailPrinter = checkFailPrinter
	sp, _ := s.Start(t)
	return sp
}
