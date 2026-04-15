package testworkflowresolver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestMergeCapabilities_NilDst(t *testing.T) {
	include := &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}}
	result := MergeCapabilities(nil, include)
	assert.Equal(t, include, result)
}

func TestMergeCapabilities_NilInclude(t *testing.T) {
	dst := &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}}
	result := MergeCapabilities(dst, nil)
	assert.Equal(t, dst, result)
}

func TestMergeCapabilities_BothNil(t *testing.T) {
	result := MergeCapabilities(nil, nil)
	assert.Nil(t, result)
}

func TestMergeCapabilities_NoDuplicates(t *testing.T) {
	dst := &corev1.Capabilities{
		Add:  []corev1.Capability{"NET_ADMIN"},
		Drop: []corev1.Capability{"ALL"},
	}
	include := &corev1.Capabilities{
		Add:  []corev1.Capability{"SYS_TIME"},
		Drop: []corev1.Capability{"NET_RAW"},
	}
	result := MergeCapabilities(dst, include)
	assert.Equal(t, []corev1.Capability{"NET_ADMIN", "SYS_TIME"}, result.Add)
	assert.Equal(t, []corev1.Capability{"ALL", "NET_RAW"}, result.Drop)
}

func TestMergeCapabilities_DeduplicatesDrop(t *testing.T) {
	dst := &corev1.Capabilities{
		Drop: []corev1.Capability{"ALL"},
	}
	include := &corev1.Capabilities{
		Drop: []corev1.Capability{"ALL"},
	}
	result := MergeCapabilities(dst, include)
	assert.Equal(t, []corev1.Capability{"ALL"}, result.Drop)
}

func TestMergeCapabilities_DeduplicatesAdd(t *testing.T) {
	dst := &corev1.Capabilities{
		Add: []corev1.Capability{"NET_ADMIN", "SYS_TIME"},
	}
	include := &corev1.Capabilities{
		Add: []corev1.Capability{"NET_ADMIN", "NET_RAW"},
	}
	result := MergeCapabilities(dst, include)
	assert.Equal(t, []corev1.Capability{"NET_ADMIN", "SYS_TIME", "NET_RAW"}, result.Add)
}

func TestMergeCapabilities_DeduplicatesMultiple(t *testing.T) {
	dst := &corev1.Capabilities{
		Drop: []corev1.Capability{"ALL", "NET_RAW"},
	}
	include := &corev1.Capabilities{
		Drop: []corev1.Capability{"ALL", "NET_RAW", "SYS_TIME"},
	}
	result := MergeCapabilities(dst, include)
	assert.Equal(t, []corev1.Capability{"ALL", "NET_RAW", "SYS_TIME"}, result.Drop)
}
