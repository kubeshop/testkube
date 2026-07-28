package config

import "slices"

type ProContextMode string

const (
	ProContextModeUnknown ProContextMode = ""

	ProContextModeEnterprise ProContextMode = "enterprise"
	// TODO: Use "pro" in the future when refactoring TK Pro API server to use "pro" instead of "cloud"
	ProContextModePro ProContextMode = "cloud"
)

// Ref: #/components/schemas/PlanStatus
type ProContextStatus string

const (
	ProContextStatusUnknown           ProContextStatus = ""
	ProContextStatusActive            ProContextStatus = "Active"
	ProContextStatusCanceled          ProContextStatus = "Canceled"
	ProContextStatusIncomplete        ProContextStatus = "Incomplete"
	ProContextStatusIncompleteExpired ProContextStatus = "IncompleteExpired"
	ProContextStatusPastDue           ProContextStatus = "PastDue"
	ProContextStatusTrailing          ProContextStatus = "Trailing"
	ProContextStatusUnpaid            ProContextStatus = "Unpaid"
	ProContextStatusDeleted           ProContextStatus = "Deleted"
	ProContextStatusLocked            ProContextStatus = "Locked"
	ProContextStatusBlocked           ProContextStatus = "Blocked"
)

type ProContext struct {
	APIKey                              string
	URL                                 string
	TLSInsecure                         bool
	WorkerCount                         int
	SkipVerify                          bool
	EnvID                               string
	EnvSlug                             string
	EnvName                             string
	OrgID                               string
	OrgSlug                             string
	OrgName                             string
	Migrate                             string
	ConnectionTimeout                   int
	DashboardURI                        string
	CloudStorage                        bool
	CloudStorageSupportedInControlPlane bool
	HasSourceOfTruthCapability          bool
	Agent                               ProContextAgent
}

func (p *ProContext) GetEnvSlug(id string) string {
	for i := range p.Agent.Environments {
		if p.Agent.Environments[i].ID == id && p.Agent.Environments[i].Slug != "" {
			return p.Agent.Environments[i].Slug
		}
	}
	if p.EnvID == id && p.EnvSlug != "" {
		return p.EnvSlug
	}
	return id
}

type ProContextAgentEnvironment struct {
	ID   string
	Slug string
	Name string
}

// AgentCapability names a capability the Control Plane assigns to an agent. The
// values are the Control Plane's own, carried verbatim over the wire, so they
// must not be renamed independently of it.
type AgentCapability string

const (
	AgentCapabilityRunner        AgentCapability = "runner"
	AgentCapabilityListener      AgentCapability = "listener"
	AgentCapabilityGitOps        AgentCapability = "gitops"
	AgentCapabilityWebhooks      AgentCapability = "webhooks"
	AgentCapabilityCloudWebhooks AgentCapability = "cloud-webhooks"
)

type ProContextAgent struct {
	ID           string
	Name         string
	Disabled     bool
	Labels       map[string]string
	IsSuperAgent bool
	Environments []ProContextAgentEnvironment
	// Capabilities is empty when the Control Plane predates capability
	// reporting, which is indistinguishable from an agent with none.
	Capabilities []string
}

func (a *ProContextAgent) HasCapability(capability AgentCapability) bool {
	return slices.Contains(a.Capabilities, string(capability))
}

// ShouldPushClusterInventory reports whether this agent should run the CRD
// watcher and push the cluster-resources inventory. Only listener-capable
// agents should: the Control Plane rejects everyone else's push, and a
// runner-only deployment has no CRD RBAC to watch with. Standalone serves
// discovery from its own API, so it never pushes. Against a Control Plane too
// old to report capabilities, the listener-oriented deployment flags are the
// closest local approximation.
func ShouldPushClusterInventory(cfg *Config, proContext ProContext) bool {
	if proContext.APIKey == "" {
		return false
	}
	if len(proContext.Agent.Capabilities) > 0 {
		return proContext.Agent.HasCapability(AgentCapabilityListener)
	}
	return cfg.EnableK8sControllers || cfg.GitOpsSyncKubernetesToCloudEnabled
}
