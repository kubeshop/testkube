package validators

type Validator interface {
	Validate() ValidationResult
	// DocsURI returns URI for related documentation
	DocsURI() string
}
