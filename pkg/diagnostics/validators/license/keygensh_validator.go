package license

import (
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

func (v KeygenShValidator) Validate(subject any) (r validators.ValidationResult) {
	r = r.WithValidator("License key correctness online check")
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

	return validators.ValidationResult{
		Status: validators.StatusInvalid,
		Errors: []validators.Error{
			ErrKeygenShValidation.WithDetails(resp.Message),
		},
	}
}
