package validators

type Validator interface {
	Validate(subject any) ValidationResult
}
