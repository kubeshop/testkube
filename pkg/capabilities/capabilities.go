package capabilities

import "github.com/kubeshop/testkube/pkg/cloud"

type Capability string

const CapabilityJUnitReports Capability = "junit-reports"
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
