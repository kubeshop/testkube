package commonmapper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestMapEnvVarSourceKubeToAPI_PreservesSecretAndConfigMapRefs(t *testing.T) {
	optional := true
	source := &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: "git-secret"},
			Key:                  "token",
			Optional:             &optional,
		},
		ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: "git-config"},
			Key:                  "username",
			Optional:             &optional,
		},
	}

	mapped := MapEnvVarSourceKubeToAPI(source)
	if assert.NotNil(t, mapped) {
		if assert.NotNil(t, mapped.SecretKeyRef) {
			assert.Equal(t, "git-secret", mapped.SecretKeyRef.Name)
			assert.Equal(t, "token", mapped.SecretKeyRef.Key)
			assert.Equal(t, &optional, mapped.SecretKeyRef.Optional)
		}
		if assert.NotNil(t, mapped.ConfigMapKeyRef) {
			assert.Equal(t, "git-config", mapped.ConfigMapKeyRef.Name)
			assert.Equal(t, "username", mapped.ConfigMapKeyRef.Key)
			assert.Equal(t, &optional, mapped.ConfigMapKeyRef.Optional)
		}
	}
}

func TestMapEnvVarSourceAPIToKube_RoundTripSecretAndConfigMapRefs(t *testing.T) {
	optional := true
	source := &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: "git-secret"},
			Key:                  "token",
			Optional:             &optional,
		},
		ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: "git-config"},
			Key:                  "username",
			Optional:             &optional,
		},
	}

	roundTripped := MapEnvVarSourceAPIToKube(MapEnvVarSourceKubeToAPI(source))
	if assert.NotNil(t, roundTripped) {
		if assert.NotNil(t, roundTripped.SecretKeyRef) {
			assert.Equal(t, source.SecretKeyRef.LocalObjectReference.Name, roundTripped.SecretKeyRef.LocalObjectReference.Name)
			assert.Equal(t, source.SecretKeyRef.Key, roundTripped.SecretKeyRef.Key)
			assert.Equal(t, source.SecretKeyRef.Optional, roundTripped.SecretKeyRef.Optional)
		}
		if assert.NotNil(t, roundTripped.ConfigMapKeyRef) {
			assert.Equal(t, source.ConfigMapKeyRef.LocalObjectReference.Name, roundTripped.ConfigMapKeyRef.LocalObjectReference.Name)
			assert.Equal(t, source.ConfigMapKeyRef.Key, roundTripped.ConfigMapKeyRef.Key)
			assert.Equal(t, source.ConfigMapKeyRef.Optional, roundTripped.ConfigMapKeyRef.Optional)
		}
	}
}
