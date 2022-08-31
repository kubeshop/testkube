package ui

// PrintConfigError prints error message suggestion and documentation link
func PrintConfigError(err error) {
	ui.PrintOnError("    Can't access config file", err)
	ui.Info(IconSuggestion+"  Suggestion:", "Do you have enough rights to handle the config file?")
	ui.Info(IconDocumentation+"  Documentation:", "https://kubeshop.github.io/testkube/")
}
