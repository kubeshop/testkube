package license

import (
	"fmt"
	"regexp"

	"github.com/kubeshop/testkube/pkg/diagnostics/validators"
)

func NewOnlineLicenseKeyValidator() OnlineLicenseKeyValidator {
	return OnlineLicenseKeyValidator{}
}

type OnlineLicenseKeyValidator struct {
}

func (v OnlineLicenseKeyValidator) Name() string {
	return "License key format check"
}

// Validate validates a given license key for format / length correctness without calling external services
func (v OnlineLicenseKeyValidator) Validate(subject any) validators.ValidationResult {
	r := validators.NewResult()

	// get key
	key, ok := subject.(string)
	if !ok {
		return r.WithError(ErrLicenseKeyInvalidFormat)
	}

	if key == "" {
		return r.WithError(ErrLicenseKeyNotFound)
	}

	// Check if the license key is the correct length and validate
	if len(key) != 37 {
		return r.WithError(ErrOnlineLicenseKeyInvalidLength.WithDetails(fmt.Sprintf("Passed license key length is %d and should be 37", len(key))))
	}

	// Check if the license key matches the expected format
	match, _ := regexp.MatchString(`^([A-Z0-9_]{6}-){5}[^-]{2}$`, key)
	if !match {
		return r.WithError(ErrOnlineLicenseKeyInvalidLength)
	}

	return r.WithValidStatus()
}
