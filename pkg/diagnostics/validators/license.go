package validators

import "errors"

const (
	ErrLicenseFileNotFound = "license file not found"
)

var SuggestionsMap = map[string][]string{
	ErrLicenseFileNotFound: {
		"please provide valid license file for your control plane installation",
		"you can pass file as environment variable details here https://docs.testkube.io/blabalbalabl",
		"license file can be obtained through https://tesktube.io website, follow XXX to grab it",
	},
}

func NewLicenseValidator() LicenseValidator {
	return LicenseValidator{}
}

type LicenseValidator struct {
}

func (v LicenseValidator) DocsURI() string {
	return "https://docs.testkube.io/articles/migrate-from-oss#license"
}
func (v LicenseValidator) Validate() ValidationResult {
	// get license

	// validate

	// example mocked result
	return ValidationResult{
		Status: StatusInvalid,

		Message: "License file not valid",

		Errors: []ErrorWithSuggesstion{
			{
				Error:       errors.New(ErrLicenseFileNotFound),
				Suggestions: SuggestionsMap[ErrLicenseFileNotFound],
				DocsURI:     "https://docs.testkube.io/articles/migrate-from-oss#license",
			},
		},
	}
}
