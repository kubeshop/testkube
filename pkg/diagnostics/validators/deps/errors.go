package deps

import v "github.com/kubeshop/testkube/pkg/diagnostics/validators"

var (
	ErrKubernetesInvalidVersion = v.Err("Your Kubernetes cluster has older version than Testkube require to run correctly", v.ErrorKindFileNotFound).
					WithSuggestion("Consider upgrading Kubernetes to recent version").
					WithSuggestion("Please follow your provider upgrading instructions")
	ErrKubectlInvalidVersion = v.Err("kubectl has older version than required", v.ErrorKindFileNotFound).
					WithSuggestion("Consider upgrading kubectl to recent version").
					WithDocsURI("https://kubernetes.io/docs/tasks/tools")

	ErrKubectlFileNotFound = v.Err("kubectl binary not found", v.ErrorKindFileNotFound).
				WithSuggestion("Make sure Kubectl is correctly installed and provided in system PATH").
				WithDocsURI("https://kubernetes.io/docs/tasks/tools")

	ErrHelmFileNotFound = v.Err("helm binary not found", v.ErrorKindFileNotFound).
				WithSuggestion("Make sure Helm is correctly installed and provided in system PATH").
				WithDocsURI("https://helm.sh/docs/intro/install/")

	ErrHelmInvalidVersion = v.Err("helm has older version than required", v.ErrorKindFileNotFound).
				WithSuggestion("Consider upgrading helm to recent version").
				WithDocsURI("https://helm.sh/docs/intro/install/")
)
