package deps

import (
	"errors"

	"github.com/kubeshop/testkube/pkg/diagnostics/validators"
)

type HelmDependencyValidator struct{}

func (v HelmDependencyValidator) Name() string {
	return "helm check"
}

func (v HelmDependencyValidator) Validate(subject any) (r validators.ValidationResult) {

	if !checkFileExists("helm") {
		return r.WithStdError(errors.New("helm not found"))
	}

	return r.WithValidStatus()
}

func NewHelmDependencyValidator() HelmDependencyValidator {
	return HelmDependencyValidator{}
}
