package validators

type Status string

const (
	StatusValid   Status = "valid"
	StatusInvalid Status = "invalid"
)

func NewErrorResponse(err error) ValidationResult {
	return ValidationResult{
		Status:  StatusInvalid,
		Message: err.Error(),
		Errors: []ErrorWithSuggesstion{
			{
				Error: err,
				Suggestions: []string{
					"got unexpected error, please contact Testkube team",
				},
			},
		},
	}
}

func NewValidResponse() ValidationResult {
	return ValidationResult{
		Status: StatusValid,
	}
}

type ValidationResult struct {
	Validator string
	Status    Status
	Message   string
	// Errors
	Errors []ErrorWithSuggesstion

	// Logs
	Logs map[string]string
}
