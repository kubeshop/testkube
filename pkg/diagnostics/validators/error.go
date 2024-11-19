package validators

type ErrorWithSuggesstion struct {
	Error       error
	Details     string
	Suggestions []string
	DocsURI     string
}
