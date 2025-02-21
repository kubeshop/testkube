package license

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/diagnostics/validators"
)

var (

	// Errors
	ErrInvalidLicenseFormat = validators.ValidationResult{
		Status: validators.StatusInvalid,
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

func (v KeygenShValidator) Name() string {
	return "License key correctness online check"
}

func (v KeygenShValidator) Validate(subject any) (r validators.ValidationResult) {
	// get key
	key, ok := subject.(string)
	if !ok {
		return r.WithError(ErrLicenseKeyInvalidFormat)
	}

	// validate
	resp, err := v.Client.ValidateLicense(LicenseRequest{License: key})
	if err != nil {
		return r.WithStdError(err)
	}

	return mapResponseToValidatonResult(r, resp)

}

func mapResponseToValidatonResult(r validators.ValidationResult, resp *LicenseResponse) validators.ValidationResult {
	if resp.Valid {
		return r.WithValidStatus()
	}

	switch resp.Code {
	case "EXPIRED":
		return r.WithError(ErrKeygenShValidationExpired.
			WithDetails(fmt.Sprintf("Looks like your license '%s' has expired at '%s'", resp.License.Name, resp.License.Expiry)))
	}

	return r.WithError(ErrKeygenShValidation.WithDetails(resp.Message + resp.Code))
}
