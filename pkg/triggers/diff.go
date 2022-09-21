package triggers

import (
	"github.com/kubeshop/testkube-operator/pkg/validation/tests/v1/testtrigger"
	apps_v1 "k8s.io/api/apps/v1"
	core_v1 "k8s.io/api/core/v1"
)

func diffDeployments(old, new *apps_v1.Deployment) []testtrigger.Cause {
	var causes []testtrigger.Cause

	if old.Spec.Replicas != new.Spec.Replicas {
		causes = append(causes, testtrigger.CauseDeploymentScaleUpdate)
	}
	for _, newContainer := range new.Spec.Template.Spec.Containers {
		oldContainer := findContainer(old.Spec.Template.Spec.Containers, newContainer.Name)
		if oldContainer == nil {
			causes = append(causes, testtrigger.CauseDeploymentContainersModified)
			continue
		}
		if oldContainer.Image != newContainer.Image {
			causes = append(causes, testtrigger.CauseDeploymentImageUpdate)
		}
		if diffEnv(oldContainer.Env, newContainer.Env) {
			causes = append(causes, testtrigger.CauseDeploymentEnvUpdate)
		}
		break
	}
	return causes
}

func diffEnv(old, new []core_v1.EnvVar) bool {
	if len(old) != len(new) {
		return true
	}
	for i := range new {
		nameUpdated := old[i].Name != new[i].Name
		valueUpdated := old[i].Value != new[i].Value
		valueFromUpdated := old[i].ValueFrom != new[i].ValueFrom
		if nameUpdated || valueUpdated || valueFromUpdated {
			return true
		}
	}
	return false
}
