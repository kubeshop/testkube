package capabilities

import "github.com/kubeshop/testkube/pkg/cloud"

type Capability string

const CapabilityJUnitReports Capability = "junit-reports"

// Deprecated: NewArchitecture is always enabled since November 2025.
// This is kept for backwards compatibility with older agents.
// Feel free to permanently delete this after 2026Q1.
const CapabilityNewArchitecture Capability = "exec"
const CapabilityCloudStorage Capability = "tw-storage"

func Enabled(capabilities []*cloud.Capability, capability Capability) bool {
	for _, c := range capabilities {
		if c.Name == string(capability) {
			return c.Enabled
		}
	}
	return false
}
