package triggers

import (
	"maps"

	"github.com/google/go-cmp/cmp"
	apps_v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube-operator/pkg/validation/tests/v1/testtrigger"
)

func diffDeployments(old, new *apps_v1.Deployment) []testtrigger.Cause {
	var causes []testtrigger.Cause

	if *old.Spec.Replicas != *new.Spec.Replicas {
		causes = append(causes, testtrigger.CauseDeploymentScaleUpdate)
	}

	containerCauses := append(diffContainers(old.Spec.Template.Spec.InitContainers, new.Spec.Template.Spec.InitContainers),
		diffContainers(old.Spec.Template.Spec.Containers, new.Spec.Template.Spec.Containers)...)

	unique := make(map[testtrigger.Cause]struct{})
	for _, containerCause := range containerCauses {
		if _, ok := unique[containerCause]; !ok {
			unique[containerCause] = struct{}{}
			causes = append(causes, containerCause)
		}
	}

	if old.Generation != new.Generation {
		causes = append(causes, testtrigger.CauseDeploymentGenerationModified)
	}

	if old.ResourceVersion != new.ResourceVersion {
		causes = append(causes, testtrigger.CauseDeploymentResourceModified)
	}

	return causes
}

func diffContainers(old, new []corev1.Container) []testtrigger.Cause {
	var causes []testtrigger.Cause
	oldNames := make(map[string]struct{})
	oldContainers := make(map[string]corev1.Container)
	for _, o := range old {
		oldNames[o.Name] = struct{}{}
		oldContainers[o.Name] = o
	}

	newNames := make(map[string]struct{})
	newContainers := make(map[string]corev1.Container)
	for _, n := range new {
		newNames[n.Name] = struct{}{}
		newContainers[n.Name] = n
	}

	if !maps.Equal(oldNames, newNames) {
		causes = append(causes, testtrigger.CauseDeploymentContainersModified)
	}

	for name, newContainer := range newContainers {
		if oldContainer, ok := oldContainers[name]; ok {
			if !cmp.Equal(oldContainer, newContainer) {
				causes = append(causes, testtrigger.CauseDeploymentContainersModified)
			}

			if oldContainer.Image != newContainer.Image {
				causes = append(causes, testtrigger.CauseDeploymentImageUpdate)
			}

			if !cmp.Equal(oldContainer.Env, newContainer.Env) {
				causes = append(causes, testtrigger.CauseDeploymentEnvUpdate)
			}
		}
	}

	return causes
}
