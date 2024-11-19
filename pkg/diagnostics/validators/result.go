package validators

type Status string

const (
	StatusValid   Status = "valid"
	StatusInvalid Status = "invalid"
)

func NewResult() ValidationResult {
	return ValidationResult{
		Status: StatusInvalid,
	}
}

func NewErrorResponse(err error) ValidationResult {
	return ValidationResult{
		Status:  StatusInvalid,
		Message: err.Error(),
		Errors: []Error{
			{
				Message: err.Error(),
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

// ValidationResult represents the result of a validation operation
type ValidationResult struct {
	BreakValidationChain bool

	Validator string
	Status    Status
	Message   string
	// Errors
	Errors []Error

	// Logs
	Logs map[string]string
}

func (r ValidationResult) WithValidStatus() ValidationResult {
	r.Status = StatusValid
	return r
}

func (r ValidationResult) WithInvalidStatus() ValidationResult {
	r.Status = StatusValid
	return r
}

func (r ValidationResult) WithError(err Error) ValidationResult {
	r.Status = StatusInvalid
	r.Errors = append(r.Errors, err)
	return r
}
