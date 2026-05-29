package triggers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// unstructuredTemplateObject wraps an unstructured Kubernetes object so that Go
// templates can access it with the same field names as typed API objects.
// For example, {{ .ObjectMeta.Labels }} works the same way for a CRD resource
// as it does for a native Deployment.
type unstructuredTemplateObject map[string]interface{}

// newUnstructuredTemplateObject creates a template-friendly wrapper from an
// *unstructured.Unstructured object. The wrapper exposes ObjectMeta (with
// Labels, Annotations, Name, Namespace, etc.) and Spec/Status as maps,
// making them accessible with the same template syntax used for typed objects.
func newUnstructuredTemplateObject(u *unstructured.Unstructured) *unstructuredTemplateObject {
	objectMeta := metav1.ObjectMeta{
		Name:                       u.GetName(),
		GenerateName:               u.GetGenerateName(),
		Namespace:                  u.GetNamespace(),
		UID:                        u.GetUID(),
		ResourceVersion:            u.GetResourceVersion(),
		Generation:                 u.GetGeneration(),
		CreationTimestamp:          u.GetCreationTimestamp(),
		DeletionTimestamp:          u.GetDeletionTimestamp(),
		DeletionGracePeriodSeconds: u.GetDeletionGracePeriodSeconds(),
		Labels:                     u.GetLabels(),
		Annotations:                u.GetAnnotations(),
		OwnerReferences:            u.GetOwnerReferences(),
		Finalizers:                 u.GetFinalizers(),
	}

	metadata, ok := u.Object["metadata"].(map[string]interface{})
	if !ok {
		if converted, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&objectMeta); err == nil {
			metadata = converted
		} else {
			metadata = map[string]interface{}{
				"name":      objectMeta.Name,
				"namespace": objectMeta.Namespace,
				"labels":    objectMeta.Labels,
			}
		}
	}

	spec, _ := u.Object["spec"].(map[string]interface{})
	status, _ := u.Object["status"].(map[string]interface{})

	obj := unstructuredTemplateObject{
		// Legacy unstructured map access (existing v1 templates)
		"metadata":   metadata,
		"spec":       spec,
		"status":     status,
		"kind":       u.GetKind(),
		"apiVersion": u.GetAPIVersion(),

		// Typed object parity for Go templates
		"ObjectMeta": objectMeta,
		"Spec":       spec,
		"Status":     status,
		"TypeMeta": metav1.TypeMeta{
			Kind:       u.GetKind(),
			APIVersion: u.GetAPIVersion(),
		},
		"Kind":       u.GetKind(),
		"APIVersion": u.GetAPIVersion(),
	}

	return &obj
}
