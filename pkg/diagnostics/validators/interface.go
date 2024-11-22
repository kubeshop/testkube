package validators

// Validator interface defines the Validate method for validation logic
type Validator interface {
	// Validate runs validation logic against subject
	Validate(subject any) ValidationResult
	Name() string
}
