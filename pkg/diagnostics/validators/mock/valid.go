package mock

import v "github.com/kubeshop/testkube/pkg/diagnostics/validators"

type AlwaysValidValidator struct {
	Name string
}

func (val AlwaysValidValidator) Validate(subject any) v.ValidationResult {
	return v.NewResult().WithValidStatus().WithValidator("Always valid " + val.Name)
}
