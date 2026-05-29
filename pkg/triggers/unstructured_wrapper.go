package triggers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// unstructuredTemplateObject wraps an unstructured Kubernetes object so that Go
// templates can access it with the same field names as typed API objects.
// For example, {{ .ObjectMeta.Labels }} works the same way for a CRD resource
// as it does for a native Deployment.
type unstructuredTemplateObject struct {
	metav1.TypeMeta `json:",inline"`
	ObjectMeta      metav1.ObjectMeta      `json:"metadata"`
	Spec            map[string]interface{} `json:"spec,omitempty"`
	Status          map[string]interface{} `json:"status,omitempty"`
}

// newUnstructuredTemplateObject creates a template-friendly wrapper from an
// *unstructured.Unstructured object. The wrapper exposes ObjectMeta (with
// Labels, Annotations, Name, Namespace, etc.) and Spec/Status as maps,
// making them accessible with the same template syntax used for typed objects.
func newUnstructuredTemplateObject(u *unstructured.Unstructured) *unstructuredTemplateObject {
	obj := &unstructuredTemplateObject{
		TypeMeta: metav1.TypeMeta{
			Kind:       u.GetKind(),
			APIVersion: u.GetAPIVersion(),
		},
	}

	// Populate ObjectMeta from the unstructured accessors
	obj.ObjectMeta = metav1.ObjectMeta{
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

	// Extract spec and status as raw maps for template access
	if spec, ok := u.Object["spec"].(map[string]interface{}); ok {
		obj.Spec = spec
	}
	if status, ok := u.Object["status"].(map[string]interface{}); ok {
		obj.Status = status
	}

	return obj
}
