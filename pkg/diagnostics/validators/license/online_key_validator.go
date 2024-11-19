package license

import (
	"regexp"

	"github.com/kubeshop/testkube/pkg/diagnostics/validators"
)

const exampleLicense = "AB24F3-405E39-C3F657-94D113-F06C13-V3"

func NewOnlineLicenseKeyValidator() LocalLicenseKeyValidator {
	return LocalLicenseKeyValidator{}
}

type LocalLicenseKeyValidator struct {
}

// Validate validates a given license key for format / length correctness without calling external services
func (v LocalLicenseKeyValidator) Validate(subject any) validators.ValidationResult {
	r := validators.NewResult().WithValidator("License key")

	// get key
	key, ok := subject.(string)
	if !ok {
		return r.WithError(ErrLicenseKeyInvalidFormat)
	}

	if key == "" {
		return r.WithError(ErrLicenseKeyNotFound)
	}

	// Check if the license key is the correct length and validate
	if len(key) != 29 {
		return r.WithError(ErrOnlineLicenseKeyInvalidLength)
	}

	// Check if the license key matches the expected format
	match, _ := regexp.MatchString(`^[A-Z0-9]{6}-[A-Z0-9]{6}-[A-Z0-9]{6}-[A-Z0-9]{6}-[A-Z0-9]{6}-[A-Z0-9]{1-2}$`, key)
	if !match {
		println(match)
		return r.WithError(ErrLicenseKeyInvalidFormat)
	}

	return validators.NewValidResponse()
}
