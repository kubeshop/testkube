package mock

import v "github.com/kubeshop/testkube/pkg/diagnostics/validators"

type AlwaysInvalidMultiValidator struct {
	Name string
}

func (val AlwaysInvalidMultiValidator) Validate(subject any) v.ValidationResult {
	return v.NewResult().
		WithValidator("Always invalid " + val.Name).
		WithError(v.Err("err1", v.ErrorKindCustom).WithDetails("some error occured")).
		WithError(v.Err("err2", v.ErrorKindCustom).WithDetails("some error occured")).
		WithError(v.Err("err3", v.ErrorKindCustom).WithDocsURI("https://docs.testkube.io/"))
}
