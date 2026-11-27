package config

import (
	"fmt"
	"strings"
)

// GitOpsAllNamespaces is the GITOPS_NAMESPACES entry that watches every namespace.
const GitOpsAllNamespaces = "*"

// GitOpsNamespace is one entry of GITOPS_NAMESPACES.
type GitOpsNamespace struct {
	Name string
	// EnvID is the Environment the namespace syncs into; empty is the default Environment.
	EnvID string
}

// GitOpsNamespaces is the parsed GITOPS_NAMESPACES value.
type GitOpsNamespaces []GitOpsNamespace

// WatchesAll reports whether an entry is GitOpsAllNamespaces.
func (n GitOpsNamespaces) WatchesAll() bool {
	for _, ns := range n {
		if ns.Name == GitOpsAllNamespaces {
			return true
		}
	}
	return false
}

// Names returns the named namespaces, without GitOpsAllNamespaces.
func (n GitOpsNamespaces) Names() []string {
	names := make([]string, 0, len(n))
	for _, ns := range n {
		if ns.Name != GitOpsAllNamespaces {
			names = append(names, ns.Name)
		}
	}
	return names
}

// ParseGitOpsNamespaces parses GITOPS_NAMESPACES. Entries are comma separated and trimmed, and
// empty entries are ignored. An entry is ns, ns=<envId> or *. An empty value means
// defaultNamespace in the default Environment.
func ParseGitOpsNamespaces(raw, defaultNamespace string) (GitOpsNamespaces, error) {
	var result GitOpsNamespaces
	seen := make(map[string]struct{})
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		name, envID, mapped := strings.Cut(entry, "=")
		name = strings.TrimSpace(name)
		envID = strings.TrimSpace(envID)
		switch {
		case name == "":
			return nil, fmt.Errorf("GITOPS_NAMESPACES: entry %q has no namespace", entry)
		case mapped && envID == "":
			return nil, fmt.Errorf("GITOPS_NAMESPACES: entry %q has no Environment id after '='", entry)
		case mapped && name == GitOpsAllNamespaces:
			return nil, fmt.Errorf("GITOPS_NAMESPACES: entry %q cannot map %q to an Environment", entry, GitOpsAllNamespaces)
		}
		if _, ok := seen[name]; ok {
			return nil, fmt.Errorf("GITOPS_NAMESPACES: namespace %q appears more than once", name)
		}
		seen[name] = struct{}{}
		result = append(result, GitOpsNamespace{Name: name, EnvID: envID})
	}
	if len(result) == 0 {
		return GitOpsNamespaces{{Name: defaultNamespace}}, nil
	}
	return result, nil
}

// ValidateGitOpsNamespaces checks GITOPS_NAMESPACES against the Control Plane.
//
// Every namespace syncs into the Environment the Control Plane reports for the agent token
// (ProContext.ControlPlaneEnvID), not TESTKUBE_PRO_ENV_ID: the sync client sends neither updates
// nor deletes with an Environment. So an entry mapped to another Environment fails rather than
// syncing there silently, and so does any mapping when the Control Plane reports no Environment.
//
// The sync-multi-namespace capability is needed for any namespace other than defaultNamespace
// (TESTKUBE_NAMESPACE), and so for several namespaces or *. Without it the Control Plane ignores
// the namespace of a request: it keeps same-name resources of different namespaces together, and
// it deletes in defaultNamespace while an update takes the namespace of its payload, so the
// ownership check may refuse the delete of a resource synced from another namespace. Only
// defaultNamespace passes without it, as before GITOPS_NAMESPACES.
func ValidateGitOpsNamespaces(namespaces GitOpsNamespaces, defaultNamespace string, proContext ProContext) error {
	syncEnvID := proContext.ControlPlaneEnvID
	for _, ns := range namespaces {
		if ns.EnvID == "" || ns.EnvID == syncEnvID {
			continue
		}
		if syncEnvID == "" {
			return fmt.Errorf("GITOPS_NAMESPACES: namespace %q maps to Environment %q, but the Control Plane reported no Environment for the agent token to check it against",
				ns.Name, ns.EnvID)
		}
		return fmt.Errorf("GITOPS_NAMESPACES: namespace %q maps to Environment %q, but every namespace syncs into the Environment of the agent token, %q",
			ns.Name, ns.EnvID, syncEnvID)
	}
	if proContext.MultiNamespaceSyncSupportedInControlPlane {
		return nil
	}
	for _, ns := range namespaces {
		switch ns.Name {
		case defaultNamespace:
		case GitOpsAllNamespaces:
			return fmt.Errorf("GITOPS_NAMESPACES: %q watches every namespace, but the Control Plane lacks the sync-multi-namespace capability",
				GitOpsAllNamespaces)
		default:
			return fmt.Errorf("GITOPS_NAMESPACES: namespace %q is not the agent namespace %q, but the Control Plane lacks the sync-multi-namespace capability",
				ns.Name, defaultNamespace)
		}
	}
	return nil
}
