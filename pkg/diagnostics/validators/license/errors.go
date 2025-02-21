package license

import (
	v "github.com/kubeshop/testkube/pkg/diagnostics/validators"
)

// Errors definitions for license based logic
var (
	ErrLicenseKeyNotFound = v.Err("license key not found", v.ErrorKindKeyNotFound).
				WithSuggestion("Make sure license key was correctly provided in for the testkube-cloud-api deployment").
				WithSuggestion("You can grab deployment detail with kubectl command - `kubectl get deployment testkube-cloud-api -n testkube`, check for ENTERPRISE_LICENSE_KEY value").
				WithSuggestion("Check your Helm chart installation values")

	ErrLicenseKeyInvalidFormat = v.Err("license key invalid format", v.ErrorKindInvalidKeyContent)

	ErrOnlineLicenseKeyInvalidLength = v.Err("license key invalid length", v.ErrorKindInvalidKeyContent).
						WithSuggestion("License key should be in form XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX-XX - 37 chars in length").
						WithSuggestion("Make sure license key is in valid format").
						WithSuggestion("Make sure there is no whitespaces on the begining and the end of the key")

	ErrWhitespacesAdded = v.Err("license key contains additional whitespaces", v.ErrorKindBadWhitespaces).
				WithSuggestion("Make sure there is no whitespaces on the begining and the end of the key")

	ErrLicenseFileNotFound = v.Err("license file not found", v.ErrorKindFileNotFound).
				WithSuggestion("Make sure license key was correctly provided in for the testkube-cloud-api deployment").
				WithSuggestion("You can grab deployment detail with kubectl command - `kubectl get deployment testkube-cloud-api -n testkube`, check for ENTERPRISE_LICENSE_FILE value")

	ErrKeygenShValidation = v.Err("license is invalid", v.ErrorKindInvalidKeyContent)

	ErrKeygenShValidationExpired = v.Err("license is expired", v.ErrorKindLicenseExpired).
					WithDetails("Looks like your testkube license has expired").
					WithSuggestion("Please contact testkube team [https://testkube.io/contact] to check with your license")

	ErrOfflineLicenseKeyInvalidPrefix = v.Err("license key has invalid prefix", v.ErrorKindInvalidKeyContent).
						WithDetails("License key should start with 'key/' string").
						WithSuggestion("Make sure license key is in valid format").
						WithSuggestion("Make sure you're NOT using 'online' keys for air-gapped ('offline') installations").
						WithSuggestion("Make sure there is no whitespaces on the begining and the end of the key")

	ErrOfflineLicensePublicKeyMissing        = v.Err("public key is missing", v.ErrorKindLicenseInvalid)
	ErrOfflineLicenseInvalid                 = v.Err("offline license is invalid", v.ErrorKindLicenseInvalid)
	ErrOfflineLicenseVerificationInvalid     = v.Err("offline license verification error", v.ErrorKindLicenseInvalid)
	ErrOfflineLicenseCertificateInvalid      = v.Err("offline license certificate error", v.ErrorKindLicenseInvalid)
	ErrOfflineLicenseDecodingError           = v.Err("offline license decoding error", v.ErrorKindLicenseInvalid)
	ErrOfflineLicenseLicenseFileIsNotGenuine = v.Err("license file is not genuine", v.ErrorKindLicenseInvalid)
	ErrOfflineLicenseClockTamperingDetected  = v.Err("system clock tampering detected", v.ErrorKindLicenseInvalid)
	ErrOfflineLicenseFileExpired             = v.Err("license file is expired", v.ErrorKindLicenseExpired).
							WithDetails("Looks like your testkube license has expired").
							WithSuggestion("Please contact testkube team [https://testkube.io/contact] to check with your license")

	ErrOfflineLicenseDatasetIsMissing = v.Err("license dataset missing", v.ErrorKindLicenseInvalid)

	ErrOfflineLicenseExpired = v.Err("license is expired", v.ErrorKindLicenseExpired).
					WithDetails("Looks like your testkube license has expired").
					WithSuggestion("Please contact testkube team [https://testkube.io/contact] to check with your license")
)
