package license

import (
	"strings"

	"github.com/kubeshop/testkube/pkg/diagnostics/validators"
)

func NewOfflineLicenseKeyValidator() OfflineLicenseKeyValidator {
	return OfflineLicenseKeyValidator{}
}

type OfflineLicenseKeyValidator struct {
}

// Validate validates a given license key for format / length correctness without calling external services
func (v OfflineLicenseKeyValidator) Validate(subject any) (r validators.ValidationResult) {
	// get key
	key, ok := subject.(string)
	if !ok {
		return r.WithError(ErrLicenseKeyInvalidFormat)
	}

	if key == "" {
		return r.WithError(ErrLicenseKeyNotFound)
	}

	// key can be in enrypted format
	if !strings.HasPrefix(key, "key/") {
		return r.WithError(ErrLicenseKeyInvalidLength)

	}

	return validators.NewValidResponse()
}
