package deps

import v "github.com/kubeshop/testkube/pkg/diagnostics/validators"

var (
	ErrKubectlFileNotFound = v.Err("license key not found", v.ErrorKindFileNotFound).
		WithSuggestion("Make sure kubectl is correctly installed and provided in system PATH").
		WithDocsURI("https://kubernetes.io/docs/tasks/tools")
)
