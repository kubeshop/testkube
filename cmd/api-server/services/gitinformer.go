package services

import intconfig "github.com/kubeshop/testkube/internal/config"

func ShouldRunGitInformer(
	useTestTriggerControlPlane bool,
	useCloudTestTriggers bool,
	proContext intconfig.ProContext,
) bool {
	if useCloudTestTriggers {
		// Cloud test trigger client requires the trigger control plane and
		// environment ID for list/get/update calls.
		if !useTestTriggerControlPlane || proContext.EnvID == "" {
			return false
		}
	}

	// OSS mode (Kubernetes trigger client) can run without environment ID
	// and does not require the trigger control plane.
	return true
}
