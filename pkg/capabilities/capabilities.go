package capabilities

import "github.com/kubeshop/testkube/pkg/cloud"

type Capability string

const CapabilityJUnitReports Capability = "junit-reports"

// Deprecated: NewArchitecture is always enabled since November 2025.
// This is kept for backwards compatibility with older agents.
// Feel free to permanently delete this after 2026Q1.
const CapabilityNewArchitecture Capability = "exec"
const CapabilityCloudStorage Capability = "tw-storage"

// CapabilityArtifactRead is whether the control plane can grant read access to the
// artifacts of another execution, which is what read_artifact() and the `fetch`
// instruction of the `execute` step rely on.
const CapabilityArtifactRead Capability = "tw-artifact-read"

// CapabilityDependencyCache is whether the control plane can store and serve the
// dependency cache a step's `cache` block asks for.
//
// A control plane without it makes every restore a miss and every save a no-op, rather
// than failing the step: a cache is an optimization, so a workflow that ran before the
// capability existed has to keep running after.
const CapabilityDependencyCache Capability = "tw-cache"

// CapabilitySourceOfTruth is whether the control plane is ready to act as source of truth.
// When this capability is present, newer versions of the agent MUST migrate which entails
// pushing data to the control plane and handing over control to let the control plane be the new source of truth.
const CapabilitySourceOfTruth Capability = "source-of-truth"

// CapabilityControlPlaneResources is whether the control plane accepts Control Plane resources,
// such as Insights boards, through the SyncService (its control_plane_resource arm). A GitOps
// runner syncs those custom resources only when the control plane advertises it, so an older
// control plane never receives an arm it cannot handle.
const CapabilityControlPlaneResources Capability = "controlplane-resources"

// CapabilityControlPlaneResourcesAI is whether the control plane takes the AI kinds (AI Agents,
// MCP servers and LLM providers) through the control_plane_resource arm. It is advertised apart
// from CapabilityControlPlaneResources because a control plane serves those kinds only with AI
// features on PostgreSQL, and one that predates their sync refuses them.
const CapabilityControlPlaneResourcesAI Capability = "controlplane-resources-ai"

// CapabilityControlPlaneResourcesTestBundles is whether the control plane takes Test Bundles
// through the control_plane_resource arm, for the same reasons as CapabilityControlPlaneResourcesAI.
const CapabilityControlPlaneResourcesTestBundles Capability = "controlplane-resources-testbundles"

func Enabled(capabilities []*cloud.Capability, capability Capability) bool {
	for _, c := range capabilities {
		if c.Name == string(capability) {
			return c.Enabled
		}
	}
	return false
}
