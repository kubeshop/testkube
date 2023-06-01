package ui

// PrintConfigError prints error message suggestion and documentation link
func PrintConfigError(err error) {
	ui.PrintOnError("    Can't access config file", err)
	ui.Info(IconSuggestion+"  Suggestion:", "Do you have enough rights to handle the config file?")
	ui.Info(IconDocumentation+"  Documentation:", "https://docs.testkube.io")
}

// PrintConfigApiError prints error message suggestion and documentation link
func PrintConfigApiError(err error) {
	ui.PrintOnError("    Can't access the API", err)
	ui.Info(IconSuggestion+"  Suggestion:", "Is the API running?")
	ui.Info(IconDocumentation+"  Documentation:", "https://docs.testkube.io")
}
