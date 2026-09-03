package localrunner

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// KubeTarget is the immutable Kubernetes selection for one local command. The
// command never writes kubeconfig or changes its current context.
type KubeTarget struct {
	ContextName string
	RESTConfig  *rest.Config
}

// ResolveKubeTarget loads kubeconfig using kubectl-compatible precedence and
// applies --context only in memory. Local mode accepts Kind and k3d contexts by
// default; the explicit override protects an accidentally selected shared cluster.
func ResolveKubeTarget(kubeconfig, contextName string, allowNonLocal bool) (KubeTarget, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfig != "" {
		rules.ExplicitPath = kubeconfig
	}
	overrides := &clientcmd.ConfigOverrides{CurrentContext: contextName}
	loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides)
	raw, err := loader.RawConfig()
	if err != nil {
		return KubeTarget{}, UsageError("loading kubeconfig: %v", err)
	}
	selected := contextName
	if selected == "" {
		selected = raw.CurrentContext
	}
	if selected == "" {
		return KubeTarget{}, UsageError("no Kubernetes context is selected; pass --context or configure a current context")
	}
	if !allowNonLocal && !isLocalContext(selected) {
		return KubeTarget{}, UsageError("Kubernetes context %q is not a Kind or k3d context; pass --allow-non-local-context only after confirming it is safe", selected)
	}
	cfg, err := loader.ClientConfig()
	if err != nil {
		return KubeTarget{}, UsageError("building Kubernetes client configuration for context %q: %v", selected, err)
	}
	if cfg.Host == "" {
		return KubeTarget{}, UsageError("Kubernetes context %q has no API server", selected)
	}
	return KubeTarget{ContextName: selected, RESTConfig: cfg}, nil
}

func isLocalContext(name string) bool {
	return strings.HasPrefix(name, "kind-") || strings.HasPrefix(name, "k3d-")
}

// ValidateNamespace rejects empty or accidentally broad namespace selection.
// Kubernetes validates the actual object name as well; this guards flag parsing
// before a request is made.
func ValidateNamespace(namespace string) error {
	if strings.TrimSpace(namespace) == "" {
		return UsageError("--namespace must not be empty")
	}
	if messages := validation.IsDNS1123Label(namespace); len(messages) > 0 {
		return UsageError("--namespace %q is not a Kubernetes namespace: %s", namespace, strings.Join(messages, "; "))
	}
	return nil
}

// shellQuote returns a POSIX-shell-safe representation for user-facing
// copy-paste hints. Kubernetes context names are allowed to contain spaces or
// shell metacharacters, so they must never be interpolated bare into a command.
func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

// KubeTargetDescription is kept separate for predictable terminal messages and tests.
func KubeTargetDescription(target KubeTarget) string {
	return fmt.Sprintf("context %s", target.ContextName)
}
