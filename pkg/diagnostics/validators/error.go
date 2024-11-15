package validators

type ErrorWithSuggesstion struct {
	Error       error
	Suggestions []string
	DocsURI     string
}
