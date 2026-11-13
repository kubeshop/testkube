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

// ValidateGitOpsNamespaces checks the Environments of GITOPS_NAMESPACES against the Control Plane.
// An entry mapped to an Environment other than the one of the agent token must name an
// Environment of the agent, and needs the sync-multi-namespace capability. The token Environment
// is exempt from the capability check, because the Control Plane already syncs into it without
// the environment-id header.
func ValidateGitOpsNamespaces(namespaces GitOpsNamespaces, proContext ProContext) error {
	for _, ns := range namespaces {
		if ns.EnvID == "" || ns.EnvID == proContext.EnvID {
			continue
		}
		if !agentHasEnvironment(proContext, ns.EnvID) {
			known := make([]string, 0, len(proContext.Agent.Environments)+1)
			if proContext.EnvID != "" {
				known = append(known, proContext.EnvID)
			}
			for _, env := range proContext.Agent.Environments {
				if env.ID != proContext.EnvID {
					known = append(known, env.ID)
				}
			}
			return fmt.Errorf("GITOPS_NAMESPACES: namespace %q maps to Environment %q, which is not an Environment of the agent (known: %s)",
				ns.Name, ns.EnvID, strings.Join(known, ", "))
		}
		if !proContext.MultiNamespaceSyncSupportedInControlPlane {
			return fmt.Errorf("GITOPS_NAMESPACES: namespace %q maps to Environment %q, but the Control Plane lacks the sync-multi-namespace capability",
				ns.Name, ns.EnvID)
		}
	}
	return nil
}

func agentHasEnvironment(proContext ProContext, envID string) bool {
	for _, env := range proContext.Agent.Environments {
		if env.ID == envID {
			return true
		}
	}
	return false
}
