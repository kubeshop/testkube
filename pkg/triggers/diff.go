package triggers

import (
	"github.com/google/go-cmp/cmp"
	apps_v1 "k8s.io/api/apps/v1"

	"github.com/kubeshop/testkube-operator/pkg/validation/tests/v1/testtrigger"
)

func diffDeployments(old, new *apps_v1.Deployment) []testtrigger.Cause {
	var causes []testtrigger.Cause

	if *old.Spec.Replicas != *new.Spec.Replicas {
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
		if !cmp.Equal(oldContainer.Env, newContainer.Env) {
			causes = append(causes, testtrigger.CauseDeploymentEnvUpdate)
		}
		break
	}
	return causes
}
