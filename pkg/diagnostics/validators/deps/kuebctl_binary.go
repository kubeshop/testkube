package deps

import (
	"fmt"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/diagnostics/validators"
	"github.com/kubeshop/testkube/pkg/semver"
)

func NewKubectlDependencyValidator() KubectlDependencyValidator {
	return KubectlDependencyValidator{
		RequiredKubectlVersion:    validators.RequiredKubectlVersion,
		RequiredKubernetesVersion: validators.RequiredKubernetesVersion,
	}
}

type KubectlDependencyValidator struct {
	RequiredKubectlVersion    string
	RequiredKubernetesVersion string
}

func (v KubectlDependencyValidator) Name() string {
	return "Kubernetes check"
}

func (v KubectlDependencyValidator) Validate(subject any) (r validators.ValidationResult) {

	if !checkFileExists("kubectl") {
		return r.WithError(ErrKubectlFileNotFound)
	}

	clientVersion, kubernetesVersion, err := common.KubectlVersion()
	if err != nil {
		return r.WithStdError(err)
	}

	ok, err := semver.Lte(v.RequiredKubectlVersion, clientVersion)
	if err != nil {
		return r.WithStdError(err)
	}
	if !ok {
		return r.WithError(ErrKubectlInvalidVersion.WithDetails(fmt.Sprintf("We need at least version %s, but your is %s, please consider upgrading", v.RequiredKubectlVersion, clientVersion)))
	}

	ok, err = semver.Lte(v.RequiredKubernetesVersion, kubernetesVersion)
	if err != nil {
		return r.WithStdError(err)
	}
	if !ok {
		return r.WithError(ErrKubernetesInvalidVersion.WithDetails(fmt.Sprintf("We need at least version %s, but your is %s, please consider upgrading", v.RequiredKubectlVersion, kubernetesVersion)))
	}

	return r.WithValidStatus()
}
