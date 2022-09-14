package triggers

import (
	apps_v1 "k8s.io/api/apps/v1"
	core_v1 "k8s.io/api/core/v1"
)

func diffDeployments(old, new *apps_v1.Deployment) []Cause {
	var causes []Cause

	if old.Spec.Replicas != new.Spec.Replicas {
		causes = append(causes, CauseDeploymentScaleUpdate)
	}
	for _, newContainer := range new.Spec.Template.Spec.Containers {
		oldContainer := findContainer(old.Spec.Template.Spec.Containers, newContainer.Name)
		if oldContainer == nil {
			causes = append(causes, CauseDeploymentContainersModified)
			continue
		}
		if oldContainer.Image != newContainer.Image {
			causes = append(causes, CauseDeploymentImageUpdate)
		}
		if diffEnv(oldContainer.Env, newContainer.Env) {
			causes = append(causes, CauseDeploymentEnvUpdate)
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
