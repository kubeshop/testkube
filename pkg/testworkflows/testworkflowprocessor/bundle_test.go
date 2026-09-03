package testworkflowprocessor

import (
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestBundleAddLabelsLabelsEveryGeneratedResource(t *testing.T) {
	bundle := &Bundle{
		Job:        batchv1.Job{},
		Secrets:    []corev1.Secret{{}},
		ConfigMaps: []corev1.ConfigMap{{}},
		Pvcs:       []corev1.PersistentVolumeClaim{{}},
	}
	bundle.AddLabels(map[string]string{"testkube.io/local": "true", "testkube.io/local-run-id": "run-1"})
	objects := []corev1.ObjectReference{
		{Kind: "Job", Name: bundle.Job.Labels["testkube.io/local-run-id"]},
		{Kind: "PodTemplate", Name: bundle.Job.Spec.Template.Labels["testkube.io/local-run-id"]},
		{Kind: "Secret", Name: bundle.Secrets[0].Labels["testkube.io/local-run-id"]},
		{Kind: "ConfigMap", Name: bundle.ConfigMaps[0].Labels["testkube.io/local-run-id"]},
		{Kind: "PersistentVolumeClaim", Name: bundle.Pvcs[0].Labels["testkube.io/local-run-id"]},
	}
	for _, object := range objects {
		if object.Name != "run-1" {
			t.Fatalf("%s did not receive the local run label: %#v", object.Kind, object)
		}
	}
}
