package validators

const (
	ErrorKindCustom       ErrorKind = "custom"
	ErrorKindFileNotFound ErrorKind = "file not found"
	ErrorKindKeyNotFound  ErrorKind = "key not found"

	ErrorKindInvalidFileContent ErrorKind = "invalid file content"
	ErrorKindInvalidKeyContent  ErrorKind = "invalid key content"
	ErrorKindBadWhitespaces     ErrorKind = "bad whitespaces"

	ErrorKindLicenseInvalid ErrorKind = "license invalid"
	ErrorKindLicenseExpired ErrorKind = "license expired"
)

var (
	// Suggestions map
	Suggestions = map[ErrorKind][]string{
		ErrorKindKeyNotFound: {
			"please provide valid file for your control plane installation",
			"you can pass file as environment variable details here https://docs.testkube.io/installation",
			"make sure valid environment variables are set in pods",
		},
		ErrorKindFileNotFound: {
			"please provide valid file for your control plane installation",
			"you can pass file as environment variable details here https://docs.testkube.io/blabalbalabl",
			"make sure valid environment variables are set in pods, you can use `kubectl describe pods ....`",
		},
		ErrorKindInvalidKeyContent: {
			"please make sure your key is in valid format",
			"please make sure given key was not modified in any editor",
			"check if provided value was not changed by accident",
			"check if additional whitespases were not added on the beggining and the end of the key",
		},
		ErrorKindInvalidFileContent: {
			"please make sure your key is in valid format",
			"please make sure given key was not modified in any editor",
			"check if provided value was not changed by accident",
			"check if additional whitespases were not added on the beggining and the end of the key",
		},
		ErrorKindBadWhitespaces: {
			"please make sure given key was not modified in any editor",
			"check if additional whitespases were not added on the beggining and the end of the key",
		},
	}
)
