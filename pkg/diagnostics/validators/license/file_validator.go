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
func (v FileValidator) Validate(subject any) (r validators.ValidationResult) {
	r = r.WithValidator("License file")
	// get file
	file, ok := subject.(string)
	if !ok {
		return r.WithError(ErrLicenseKeyInvalidFormat)
	}

	if file == "" {
		return r.WithError(ErrLicenseFileNotFound)
	}

	// TODO use checks for file format validation

	return validators.NewValidResponse()
}
