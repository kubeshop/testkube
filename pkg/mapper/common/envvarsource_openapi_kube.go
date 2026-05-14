package commonmapper

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func MapEnvVarSourceKubeToAPI(v *corev1.EnvVarSource) *testkube.EnvVarSource {
	if v == nil {
		return nil
	}
	return &testkube.EnvVarSource{
		FieldRef:         MapFieldRefKubeToAPI(v.FieldRef),
		ResourceFieldRef: MapResourceFieldRefKubeToAPI(v.ResourceFieldRef),
		ConfigMapKeyRef:  MapConfigMapKeyRefKubeToAPI(v.ConfigMapKeyRef),
		SecretKeyRef:     MapSecretKeyRefKubeToAPI(v.SecretKeyRef),
		FileKeyRef:       MapFileKeyRefKubeToAPI(v.FileKeyRef),
	}
}

func MapFieldRefKubeToAPI(v *corev1.ObjectFieldSelector) *testkube.FieldRef {
	if v == nil {
		return nil
	}
	return &testkube.FieldRef{
		ApiVersion: v.APIVersion,
		FieldPath:  v.FieldPath,
	}
}

func MapResourceFieldRefKubeToAPI(v *corev1.ResourceFieldSelector) *testkube.ResourceFieldRef {
	if v == nil {
		return nil
	}
	divisor := ""
	if !v.Divisor.IsZero() {
		divisor = v.Divisor.String()
	}
	return &testkube.ResourceFieldRef{
		ContainerName: v.ContainerName,
		Resource:      v.Resource,
		Divisor:       divisor,
	}
}

func MapConfigMapKeyRefKubeToAPI(v *corev1.ConfigMapKeySelector) *testkube.EnvVarSourceConfigMapKeyRef {
	if v == nil {
		return nil
	}
	return &testkube.EnvVarSourceConfigMapKeyRef{Name: v.Name, Key: v.Key, Optional: v.Optional}
}

func MapSecretKeyRefKubeToAPI(v *corev1.SecretKeySelector) *testkube.EnvVarSourceSecretKeyRef {
	if v == nil {
		return nil
	}
	return &testkube.EnvVarSourceSecretKeyRef{Name: v.Name, Key: v.Key, Optional: v.Optional}
}

func MapFileKeyRefKubeToAPI(v *corev1.FileKeySelector) *testkube.EnvVarSourceFileKeyRef {
	if v == nil {
		return nil
	}
	return &testkube.EnvVarSourceFileKeyRef{
		VolumeName: v.VolumeName,
		Path:       v.Path,
		Key:        v.Key,
		Optional:   v.Optional,
	}
}

func MapEnvVarSourceAPIToKube(v *testkube.EnvVarSource) *corev1.EnvVarSource {
	if v == nil {
		return nil
	}
	return &corev1.EnvVarSource{
		FieldRef:         MapFieldRefAPIToKube(v.FieldRef),
		ResourceFieldRef: MapResourceFieldRefAPIToKube(v.ResourceFieldRef),
		ConfigMapKeyRef:  MapConfigMapKeyRefAPIToKube(v.ConfigMapKeyRef),
		SecretKeyRef:     MapSecretKeyRefAPIToKube(v.SecretKeyRef),
		FileKeyRef:       MapFileKeyRefAPIToKube(v.FileKeyRef),
	}
}

func MapFieldRefAPIToKube(v *testkube.FieldRef) *corev1.ObjectFieldSelector {
	if v == nil {
		return nil
	}
	return &corev1.ObjectFieldSelector{
		APIVersion: v.ApiVersion,
		FieldPath:  v.FieldPath,
	}
}

func MapResourceFieldRefAPIToKube(v *testkube.ResourceFieldRef) *corev1.ResourceFieldSelector {
	if v == nil {
		return nil
	}
	divisor := resource.Quantity{}
	if v.Divisor != "" {
		if parsedDivisor, err := resource.ParseQuantity(v.Divisor); err == nil {
			divisor = parsedDivisor
		} else {
			divisor = resource.MustParse("1")
		}
	}
	return &corev1.ResourceFieldSelector{
		ContainerName: v.ContainerName,
		Resource:      v.Resource,
		Divisor:       divisor,
	}
}

func MapConfigMapKeyRefAPIToKube(v *testkube.EnvVarSourceConfigMapKeyRef) *corev1.ConfigMapKeySelector {
	if v == nil {
		return nil
	}
	return &corev1.ConfigMapKeySelector{
		Key:                  v.Key,
		LocalObjectReference: corev1.LocalObjectReference{Name: v.Name},
		Optional:             v.Optional,
	}
}

func MapSecretKeyRefAPIToKube(v *testkube.EnvVarSourceSecretKeyRef) *corev1.SecretKeySelector {
	if v == nil {
		return nil
	}
	return &corev1.SecretKeySelector{
		Key:                  v.Key,
		LocalObjectReference: corev1.LocalObjectReference{Name: v.Name},
		Optional:             v.Optional,
	}
}

func MapFileKeyRefAPIToKube(v *testkube.EnvVarSourceFileKeyRef) *corev1.FileKeySelector {
	if v == nil {
		return nil
	}
	return &corev1.FileKeySelector{
		VolumeName: v.VolumeName,
		Path:       v.Path,
		Key:        v.Key,
		Optional:   v.Optional,
	}
}
