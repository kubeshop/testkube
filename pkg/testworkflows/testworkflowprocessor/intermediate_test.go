package testworkflowprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestAddEmptyDirVolume_NoSizeLimit(t *testing.T) {
	layer := NewIntermediate("")
	mount := layer.AddEmptyDirVolume(nil, "/data")

	volumes := layer.Volumes()
	assert.Len(t, volumes, 1)
	assert.Equal(t, mount.Name, volumes[0].Name)
	assert.Equal(t, "/data", mount.MountPath)
	assert.NotNil(t, volumes[0].EmptyDir)
	assert.Nil(t, volumes[0].EmptyDir.SizeLimit)
}

func TestAddEmptyDirVolume_WithDefaultSizeLimit(t *testing.T) {
	layer := NewIntermediate("256Mi")
	mount := layer.AddEmptyDirVolume(nil, "/data")

	volumes := layer.Volumes()
	assert.Len(t, volumes, 1)
	assert.Equal(t, mount.Name, volumes[0].Name)
	assert.Equal(t, "/data", mount.MountPath)
	assert.NotNil(t, volumes[0].EmptyDir)
	assert.NotNil(t, volumes[0].EmptyDir.SizeLimit)

	expectedQty := resource.MustParse("256Mi")
	assert.True(t, volumes[0].EmptyDir.SizeLimit.Equal(expectedQty))
}

func TestAddEmptyDirVolume_WithExplicitSizeLimit(t *testing.T) {
	layer := NewIntermediate("256Mi")
	explicitQty := resource.MustParse("512Mi")
	source := &corev1.EmptyDirVolumeSource{SizeLimit: &explicitQty}
	mount := layer.AddEmptyDirVolume(source, "/data")

	volumes := layer.Volumes()
	assert.Len(t, volumes, 1)
	assert.Equal(t, mount.Name, volumes[0].Name)
	assert.NotNil(t, volumes[0].EmptyDir.SizeLimit)

	// Explicit sizeLimit should not be overridden by default
	assert.True(t, volumes[0].EmptyDir.SizeLimit.Equal(explicitQty))
}

func TestAddEmptyDirVolume_WithExplicitNilSizeLimit(t *testing.T) {
	layer := NewIntermediate("1Gi")
	source := &corev1.EmptyDirVolumeSource{}
	mount := layer.AddEmptyDirVolume(source, "/tmp")

	volumes := layer.Volumes()
	assert.Len(t, volumes, 1)
	assert.Equal(t, mount.Name, volumes[0].Name)
	assert.NotNil(t, volumes[0].EmptyDir.SizeLimit)

	expectedQty := resource.MustParse("1Gi")
	assert.True(t, volumes[0].EmptyDir.SizeLimit.Equal(expectedQty))
}

func TestAddEmptyDirVolume_InvalidDefaultSizeLimit_DoesNotPanic(t *testing.T) {
	layer := NewIntermediate("not-a-quantity")

	assert.NotPanics(t, func() {
		layer.AddEmptyDirVolume(nil, "/data")
	})

	volumes := layer.Volumes()
	assert.Len(t, volumes, 1)
	assert.NotNil(t, volumes[0].EmptyDir)
	assert.Nil(t, volumes[0].EmptyDir.SizeLimit)
}
