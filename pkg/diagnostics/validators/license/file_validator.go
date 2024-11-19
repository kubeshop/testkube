package license

import (
	"github.com/kubeshop/testkube/pkg/diagnostics/validators"
)

func NewFileValidator() FileValidator {
	return FileValidator{}
}

type FileValidator struct {
}

func (v FileValidator) Requireds() bool {
	return true
}

// Validate validates a given license file for format / length correctness without calling external services
func (v FileValidator) Validate(subject any) validators.ValidationResult {
	// get file
	file, ok := subject.(string)
	if !ok {
		return ErrInvalidLicenseFormat
	}

	if file == "" {
		return validators.ValidationResult{
			Status: validators.StatusInvalid,
			Errors: []validators.Error{
				ErrLicenseFileNotFound,
			},
		}

	}

	return validators.NewValidResponse()
}
