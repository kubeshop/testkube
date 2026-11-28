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

// EnvironmentID returns the Environment of the entry named namespace. It returns empty, the default
// Environment, for a namespace without a mapping, such as one that only GitOpsAllNamespaces watches,
// and for a namespace that maps to tokenEnvID. One Environment so has one value, and the name
// collision check of the sync controllers sees two namespaces of the token Environment as one.
func (n GitOpsNamespaces) EnvironmentID(namespace, tokenEnvID string) string {
	for _, ns := range n {
		if ns.Name == namespace && ns.EnvID != tokenEnvID {
			return ns.EnvID
		}
	}
	return ""
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
// A namespace without a mapping, or mapped to it, syncs into the Environment of the agent token
// (ProContext.TokenEnvID), not TESTKUBE_PRO_ENV_ID. A namespace mapped to another Environment syncs
// there through the environment-id header, so that Environment must be one of the agent's, and the
// Control Plane must have the sync-multi-namespace capability to read the header.
//
// The capability is also needed for any namespace other than defaultNamespace
// (TESTKUBE_NAMESPACE), and so for several namespaces or *. Without it the Control Plane ignores
// the namespace of a request: it keeps same-name resources of different namespaces together, and
// it deletes in defaultNamespace while an update takes the namespace of its payload, so the
// ownership check may refuse the delete of a resource synced from another namespace. Only
// defaultNamespace in the token Environment passes without it, as before GITOPS_NAMESPACES.
func ValidateGitOpsNamespaces(namespaces GitOpsNamespaces, defaultNamespace string, proContext ProContext) error {
	capable := proContext.MultiNamespaceSyncSupportedInControlPlane
	for _, ns := range namespaces {
		if ns.EnvID == "" || ns.EnvID == proContext.TokenEnvID {
			continue
		}
		if !agentHasEnvironment(proContext, ns.EnvID) {
			return fmt.Errorf("GITOPS_NAMESPACES: namespace %q maps to Environment %q, which is not an Environment of the agent (known: %s)",
				ns.Name, ns.EnvID, strings.Join(knownEnvironmentIDs(proContext), ", "))
		}
		if !capable {
			return fmt.Errorf("GITOPS_NAMESPACES: namespace %q maps to Environment %q, but the Control Plane lacks the sync-multi-namespace capability",
				ns.Name, ns.EnvID)
		}
	}
	if capable {
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

// knownEnvironmentIDs lists the token Environment first, then the other Environments of the agent.
func knownEnvironmentIDs(proContext ProContext) []string {
	known := make([]string, 0, len(proContext.Agent.Environments)+1)
	if proContext.TokenEnvID != "" {
		known = append(known, proContext.TokenEnvID)
	}
	for _, env := range proContext.Agent.Environments {
		if env.ID != proContext.TokenEnvID {
			known = append(known, env.ID)
		}
	}
	return known
}

func agentHasEnvironment(proContext ProContext, envID string) bool {
	for _, env := range proContext.Agent.Environments {
		if env.ID == envID {
			return true
		}
	}
	return false
}
