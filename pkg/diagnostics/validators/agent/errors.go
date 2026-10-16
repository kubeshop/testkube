package agent

import v "github.com/kubeshop/testkube/pkg/diagnostics/validators"

var (
	ErrClusterUnreachable = v.Err("Kubernetes cluster is not reachable", v.ErrorKindCustom).
				WithSuggestion("Make sure your kubeconfig points to the cluster Testkube is installed in").
				WithSuggestion("Check the current context with `kubectl config current-context`")

	ErrAPIServerNotFound = v.Err("Testkube API server not found", v.ErrorKindCustom).
				WithSuggestion("Pass the namespace Testkube is installed in with `--namespace`").
				WithSuggestion("Find the installation with `helm list -A`").
				WithDocsURI("https://docs.testkube.io/articles/install/overview")

	ErrAPIServerNotReady = v.Err("Testkube API server is not ready", v.ErrorKindCustom).
				WithSuggestion("Inspect the pods with `kubectl describe pods -l app.kubernetes.io/name=api-server`").
				WithSuggestion("Collect the logs with `testkube debug agent`")
)
