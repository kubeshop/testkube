package mock

import v "github.com/kubeshop/testkube/pkg/diagnostics/validators"

type AlwaysInvalidValidator struct {
	Name string
}

func (val AlwaysInvalidValidator) Validate(subject any) v.ValidationResult {
	return v.NewResult().WithError(v.Err("Some error", v.ErrorKindCustom)).WithValidator("Always invalid " + val.Name)
}
