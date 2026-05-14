package commonmapper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestMapEnvVarSourceKubeToAPI_PreservesFileKeyRef(t *testing.T) {
	optional := true
	source := &corev1.EnvVarSource{
		FileKeyRef: &corev1.FileKeySelector{
			VolumeName: "env-files",
			Path:       "secrets.env",
			Key:        "GIT_TOKEN",
			Optional:   &optional,
		},
	}

	mapped := MapEnvVarSourceKubeToAPI(source)
	if assert.NotNil(t, mapped) && assert.NotNil(t, mapped.FileKeyRef) {
		assert.Equal(t, "env-files", mapped.FileKeyRef.VolumeName)
		assert.Equal(t, "secrets.env", mapped.FileKeyRef.Path)
		assert.Equal(t, "GIT_TOKEN", mapped.FileKeyRef.Key)
		assert.Equal(t, &optional, mapped.FileKeyRef.Optional)
	}
}

func TestMapEnvVarSourceAPIToKube_PreservesFileKeyRef(t *testing.T) {
	optional := true
	source := &corev1.EnvVarSource{
		FileKeyRef: &corev1.FileKeySelector{
			VolumeName: "env-files",
			Path:       "secrets.env",
			Key:        "GIT_TOKEN",
			Optional:   &optional,
		},
	}

	roundTripped := MapEnvVarSourceAPIToKube(MapEnvVarSourceKubeToAPI(source))
	if assert.NotNil(t, roundTripped) && assert.NotNil(t, roundTripped.FileKeyRef) {
		assert.Equal(t, source.FileKeyRef.VolumeName, roundTripped.FileKeyRef.VolumeName)
		assert.Equal(t, source.FileKeyRef.Path, roundTripped.FileKeyRef.Path)
		assert.Equal(t, source.FileKeyRef.Key, roundTripped.FileKeyRef.Key)
		assert.Equal(t, source.FileKeyRef.Optional, roundTripped.FileKeyRef.Optional)
	}
}
