package deps

import (
	"errors"

	"github.com/kubeshop/testkube/pkg/diagnostics/validators"
)

type KubectlDependencyValidator struct{}

func (v KubectlDependencyValidator) Validate(subject any) (r validators.ValidationResult) {
	r = r.WithValidator("kubectl check")

	if !checkFileExists("kubectl") {
		return r.WithStdError(errors.New("kubectl not found"))
	}

	return r.WithValidStatus()
}

func NewKubectlDependencyValidator() KubectlDependencyValidator {
	return KubectlDependencyValidator{}
}
