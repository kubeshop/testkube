package services

import intconfig "github.com/kubeshop/testkube/internal/config"

func ShouldRunGitInformer(
	useTestTriggerControlPlane bool,
	useCloudTestTriggers bool,
	proContext intconfig.ProContext,
) bool {
	if !useTestTriggerControlPlane {
		return false
	}

	// Cloud test trigger client requires environment ID for list/get/update calls.
	if useCloudTestTriggers && proContext.EnvID == "" {
		return false
	}

	// OSS mode (Kubernetes trigger client) can run without environment ID.
	return true
}
