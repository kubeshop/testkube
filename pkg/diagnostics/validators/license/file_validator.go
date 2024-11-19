package license

import (
	"strings"

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

	// check if file doesn't contain invalid spaces
	cleaned := strings.TrimSpace(file)
	if file != cleaned {
		return r.WithError(ErrWhitespacesAdded)
	}

	// TODO use checks for file format validation

	return validators.NewValidResponse()
}
