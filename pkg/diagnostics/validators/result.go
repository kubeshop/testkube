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
		Status: StatusInvalid,
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

	// Errors
	Errors []Error

	// Logs
	Logs map[string]string

	AdditionalInfo string
}

func (r ValidationResult) WithValidator(v string) ValidationResult {
	r.Validator = v
	return r
}

func (r ValidationResult) WithBreak() ValidationResult {
	r.BreakValidationChain = true
	return r
}

func (r ValidationResult) WithValidStatus() ValidationResult {
	r.Status = StatusValid
	return r
}

func (r ValidationResult) WithInvalidStatus() ValidationResult {
	r.Status = StatusValid
	return r
}

func (r ValidationResult) WithAdditionalInfo(i string) ValidationResult {
	r.AdditionalInfo = i
	return r
}

func (r ValidationResult) WithError(err Error) ValidationResult {
	r.Status = StatusInvalid
	r.Errors = append(r.Errors, err)
	return r
}

func (r ValidationResult) WithStdError(err error) ValidationResult {
	r.Status = StatusInvalid
	r.Errors = append(r.Errors, Error{Kind: ErrorKindCustom, Message: err.Error()})
	return r
}
