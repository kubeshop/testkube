package localinstall

import (
	"strings"

	"k8s.io/client-go/tools/clientcmd"
)

var localContextPrefixes = []string{"kind-", "minikube", "docker-desktop", "k3d-", "rancher-desktop", "orbstack", "colima"}

// Files only: a dead cluster can't stall the installer.
func CurrentKubeContext() (name string, local bool) {
	cfg, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	if err != nil || cfg.Contexts[cfg.CurrentContext] == nil {
		return "", false
	}
	return cfg.CurrentContext, isLocalContext(cfg.CurrentContext)
}

func isLocalContext(name string) bool {
	for _, prefix := range localContextPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}
