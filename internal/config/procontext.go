package config

import (
	"slices"

	"github.com/kubeshop/testkube/pkg/cloud"
)

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

type ProContextAgent struct {
	ID           string
	Name         string
	Disabled     bool
	Labels       map[string]string
	IsSuperAgent bool
	Environments []ProContextAgentEnvironment
	// Capabilities is the effective set after the startup capability update:
	// the set the Control Plane stored, or the agent's own flag-derived set when
	// the Control Plane could not be asked. Empty in standalone mode.
	Capabilities []cloud.AgentCapability
}

func (a *ProContextAgent) HasCapability(capability cloud.AgentCapability) bool {
	return slices.Contains(a.Capabilities, capability)
}

// ShouldPushClusterInventory reports whether this agent should run the CRD
// watcher and push the cluster-resources inventory. Only listener-capable
// agents should: the Control Plane rejects everyone else's push, and a
// runner-only deployment has no CRD RBAC to watch with. Standalone serves
// discovery from its own API, so it never pushes.
func ShouldPushClusterInventory(proContext ProContext) bool {
	return proContext.APIKey != "" && proContext.Agent.HasCapability(cloud.AgentCapability_AGENT_CAPABILITY_LISTENER)
}
