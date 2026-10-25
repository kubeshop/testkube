package agent

import v "github.com/kubeshop/testkube/pkg/diagnostics/validators"

var (
	ErrClusterUnreachable = v.Err("Kubernetes cluster is not reachable", v.ErrorKindCustom).
				WithSuggestion("Make sure your kubeconfig points to the cluster Testkube is installed in").
				WithSuggestion("Check the current context with `kubectl config current-context`")

	ErrAgentNotFound = v.Err("Testkube agent not found", v.ErrorKindCustom).
				WithSuggestion("Pass the namespace Testkube is installed in with `--namespace`").
				WithSuggestion("Find the installation with `helm list -A`").
				WithSuggestion("An installation with a custom `nameOverride` is not detected").
				WithDocsURI("https://docs.testkube.io/articles/install/overview")

	ErrAgentNotReady = v.Err("Testkube agent is not ready", v.ErrorKindCustom).
				WithSuggestion("Inspect the pods named above with `kubectl describe pod <name> -n <namespace>`").
				WithSuggestion("Collect the logs with `testkube debug agent`")
)
