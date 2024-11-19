package license

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/diagnostics/validators"
)

var (

	// Errors
	ErrInvalidLicenseFormat = validators.ValidationResult{
		Status:  validators.StatusInvalid,
		Message: "Invalid license format",
		Errors: []validators.Error{
			ErrLicenseKeyInvalidFormat,
		},
	}
)

func NewKeygenShValidator() KeygenShValidator {
	return KeygenShValidator{
		Client: NewClient(),
	}
}

type KeygenShValidator struct {
	Client *Client
}

func (v KeygenShValidator) Validate(subject any) validators.ValidationResult {
	// get key
	key, ok := subject.(string)
	if !ok {
		return ErrInvalidLicenseFormat
	}

	// validate
	resp, err := v.Client.ValidateLicense(LicenseRequest{License: key})
	if err != nil {
		return validators.NewErrorResponse(err)
	}

	if resp.Valid {
		return validators.NewValidResponse()
	}

	return validators.ValidationResult{
		Status:  validators.StatusInvalid,
		Message: fmt.Sprintf("License key is not valid: '%s'", key),
		Errors: []validators.Error{
			{
				Message: resp.Message,
				DocsURI: "https://docs.testkube.io/articles/migrate-from-oss#license",
			},
		},
	}
}
