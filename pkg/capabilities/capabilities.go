package capabilities

import "github.com/kubeshop/testkube/pkg/cloud"

type Capability string

const CapabilityJUnitReports Capability = "junit-reports"

// Deprecated: NewArchitecture is always enabled since November 2025.
// This is kept for backwards compatibility with older agents.
// Feel free to permanently delete this after 2026Q1.
const CapabilityNewArchitecture Capability = "exec"
const CapabilityCloudStorage Capability = "tw-storage"

// CapabilitySourceOfTruth is whether the control plane is ready to act as source of truth.
// When this capability is present, newer versions of the agent MUST migrate which entails
// pushing data to the control plane and handing over control to let the control plane be the new source of truth.
const CapabilitySourceOfTruth Capability = "source-of-truth"

func Enabled(capabilities []*cloud.Capability, capability Capability) bool {
	for _, c := range capabilities {
		if c.Name == string(capability) {
			return c.Enabled
		}
	}
	return false
}
