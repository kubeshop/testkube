package license

import (
	v "github.com/kubeshop/testkube/pkg/diagnostics/validators"
)

// Errors definitions for license based logic
var (
	ErrLicenseFileNotFound = v.Err("license file not found", v.ErrorKindFileNotFound).
				WithSuggestion("Make sure license key was correctly provided in for the testkube-cloud-api deployment").
				WithSuggestion("You can grab deployment detail with kubectl command - `kubectl get deployment testkube-cloud-api -n testkube`, check for ENTERPRISE_LICENSE_FILE value")

	ErrLicenseKeyNotFound = v.Err("license key not found", v.ErrorKindKeyNotFound).
				WithSuggestion("Make sure license key was correctly provided in for the testkube-cloud-api deployment").
				WithSuggestion("You can grab deployment detail with kubectl command - `kubectl get deployment testkube-cloud-api -n testkube`, check for ENTERPRISE_LICENSE_KEY value").
				WithSuggestion("Check your Helm chart installation values")

	ErrLicenseKeyInvalidFormat = v.Err("license key invalid format", v.ErrorKindInvalidKeyContent)

	ErrLicenseKeyInvalidLength = v.Err("license key invalid length", v.ErrorKindInvalidKeyContent).
					WithDetails("License key should be in form XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX-XX - 29 chars in length").
					WithSuggestion("Make sure license key is in valid format").
					WithSuggestion("Make sure there is no whitespaces on the begining and the end of the key")

	ErrOfflineLicenseKeyInvalidPrefix = v.Err("license key has invalid prefix", v.ErrorKindInvalidKeyContent).
						WithDetails("License key should start with 'key/' string").
						WithSuggestion("Make sure license key is in valid format").
						WithSuggestion("Make sure there is no whitespaces on the begining and the end of the key")

	ErrWhitespacesAdded = v.Err("license key contains additional whitespaces", v.ErrorKindBadWhitespaces).
				WithSuggestion("Make sure there is no whitespaces on the begining and the end of the key")
)
