package validators

type Status string

const (
	StatusValid   Status = "valid"
	StatusInvalid Status = "invalid"
)

type ValidationResult struct {
	Validator string
	Status    Status
	Message   string
	// Errors
	Errors []ErrorWithSuggesstion

	// Logs
	Logs map[string]string

	DocsURI string
}
