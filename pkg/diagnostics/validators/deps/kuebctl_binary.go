package deps

import (
	"fmt"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/diagnostics/validators"
	"github.com/kubeshop/testkube/pkg/semver"
)

const MinimalKubectlVersion = "v1."

type KubectlDependencyValidator struct {
	MinimalKubectlVersion string
}

func (v KubectlDependencyValidator) Validate(subject any) (r validators.ValidationResult) {
	r = r.WithValidator("kubectl check")

	if !checkFileExists("kubectl") {
		return r.WithError(ErrKubectlFileNotFound)
	}

	clientVersion, _, err := common.KubectlVersion()
	if err != nil {
		return r.WithStdError(err)
	}

	fmt.Printf("%+v\n", clientVersion)

	ok, err := semver.Lt(clientVersion, v.MinimalKubectlVersion)
	if err != nil {
		return r.WithStdError(err)
	}

	if !ok {
		return r.WithError(ErrKubectlInvalidVersion.WithDetails(fmt.Sprintf("We need version %s but your is %s, consider upgrading", clientVersion, v.MinimalKubectlVersion)))
	}

	return r.WithValidStatus()
}

func NewKubectlDependencyValidator() KubectlDependencyValidator {
	return KubectlDependencyValidator{
		MinimalKubectlVersion: "1.31.3",
	}
}
