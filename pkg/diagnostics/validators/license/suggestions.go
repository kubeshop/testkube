package license

var (
	// Suggestions map
	Suggestions = map[error][]string{

		ErrLicenseFileNotFound: {
			"please provide valid license file for your control plane installation",
			"you can pass file as environment variable details here https://docs.testkube.io/blabalbalabl",
			"license file can be obtained through https://tesktube.io website, follow XXX to grab it",
			"make sure valid environment variables are set in pods",
		},

		ErrLicenseKeyInvalid: {
			"please make sure your license key is in valid format: KEY or encrypted key",
		},

		ErrLicenseKeyInvalidFormat: {
			"please make sure given key was not modified in any editor",
			"check if additional whitespases were not added on the beggining and the end of the key",
		},

		ErrLicenseKeyInvalidLength: {
			"please make sure given key was not modified in any editor",
			"check if additional whitespases were not added on the beggining and the end of the key",
		},

		ErrWhitespacesAdded: {
			"please make sure given key was not modified in any editor",
			"check if additional whitespases were not added on the beggining and the end of the key",
		},
	}
)
