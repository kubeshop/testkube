package license

import "errors"

var (
	ErrLicenseFileNotFound     = Err("license file not found")
	ErrLicenseKeyNotFound      = Err("license key not found")
	ErrLicenseKeyInvalid       = Err("license key invalid")
	ErrLicenseKeyInvalidFormat = Err("license key invalid format")
	ErrLicenseKeyInvalidLength = Err("license key invalid length")
	ErrWhitespacesAdded        = Err("license key contains additional whitespaces")
)

func IsLicenseError(err error) bool {
	return errors.Is(err, &LicenseError{})
}

func Err(e string) error {
	return &LicenseError{msg: e}
}

type LicenseError struct {
	msg string
}

func (e *LicenseError) Error() string {
	return e.msg
}
