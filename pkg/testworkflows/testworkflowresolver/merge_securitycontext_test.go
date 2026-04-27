package testworkflowresolver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
)

func TestMergeSecurityContext_NilInputs(t *testing.T) {
	t.Run("both nil", func(t *testing.T) {
		result := MergeSecurityContext(nil, nil)
		assert.Nil(t, result)
	})
	t.Run("dst nil", func(t *testing.T) {
		include := &corev1.SecurityContext{RunAsUser: common.Ptr(int64(1000))}
		result := MergeSecurityContext(nil, include)
		assert.Equal(t, include, result)
	})
	t.Run("include nil", func(t *testing.T) {
		dst := &corev1.SecurityContext{RunAsUser: common.Ptr(int64(1000))}
		result := MergeSecurityContext(dst, nil)
		assert.Equal(t, dst, result)
	})
}

func TestMergeSecurityContext_OverrideScalarFields(t *testing.T) {
	dst := &corev1.SecurityContext{
		RunAsUser:                common.Ptr(int64(1000)),
		RunAsGroup:               common.Ptr(int64(1000)),
		RunAsNonRoot:             common.Ptr(false),
		Privileged:               common.Ptr(false),
		ReadOnlyRootFilesystem:   common.Ptr(false),
		AllowPrivilegeEscalation: common.Ptr(true),
	}
	include := &corev1.SecurityContext{
		RunAsUser:                common.Ptr(int64(2000)),
		RunAsGroup:               common.Ptr(int64(2000)),
		RunAsNonRoot:             common.Ptr(true),
		ReadOnlyRootFilesystem:   common.Ptr(true),
		AllowPrivilegeEscalation: common.Ptr(false),
	}

	result := MergeSecurityContext(dst, include)

	assert.Equal(t, common.Ptr(int64(2000)), result.RunAsUser, "RunAsUser should be overridden")
	assert.Equal(t, common.Ptr(int64(2000)), result.RunAsGroup, "RunAsGroup should be overridden")
	assert.Equal(t, common.Ptr(true), result.RunAsNonRoot, "RunAsNonRoot should be overridden")
	assert.Equal(t, common.Ptr(false), result.Privileged, "Privileged should be preserved from dst")
	assert.Equal(t, common.Ptr(true), result.ReadOnlyRootFilesystem, "ReadOnlyRootFilesystem should be overridden")
	assert.Equal(t, common.Ptr(false), result.AllowPrivilegeEscalation, "AllowPrivilegeEscalation should be overridden")
}

func TestMergeSecurityContext_PreserveNonOverriddenFields(t *testing.T) {
	dst := &corev1.SecurityContext{
		RunAsUser:              common.Ptr(int64(1000)),
		RunAsNonRoot:           common.Ptr(true),
		ReadOnlyRootFilesystem: common.Ptr(true),
	}
	include := &corev1.SecurityContext{
		RunAsUser: common.Ptr(int64(2000)),
	}

	result := MergeSecurityContext(dst, include)

	assert.Equal(t, common.Ptr(int64(2000)), result.RunAsUser, "RunAsUser should be overridden")
	assert.Equal(t, common.Ptr(true), result.RunAsNonRoot, "RunAsNonRoot should be preserved")
	assert.Equal(t, common.Ptr(true), result.ReadOnlyRootFilesystem, "ReadOnlyRootFilesystem should be preserved")
}

func TestMergeSecurityContext_Capabilities(t *testing.T) {
	dst := &corev1.SecurityContext{
		Capabilities: &corev1.Capabilities{
			Add:  []corev1.Capability{"NET_ADMIN"},
			Drop: []corev1.Capability{"MKNOD"},
		},
	}
	include := &corev1.SecurityContext{
		Capabilities: &corev1.Capabilities{
			Add:  []corev1.Capability{"SYS_TIME"},
			Drop: []corev1.Capability{"ALL"},
		},
	}

	result := MergeSecurityContext(dst, include)

	assert.NotNil(t, result.Capabilities)
	assert.Contains(t, result.Capabilities.Add, corev1.Capability("NET_ADMIN"))
	assert.Contains(t, result.Capabilities.Add, corev1.Capability("SYS_TIME"))
	assert.Contains(t, result.Capabilities.Drop, corev1.Capability("MKNOD"))
	assert.Contains(t, result.Capabilities.Drop, corev1.Capability("ALL"))
}

func TestMergeSecurityContext_ProcMount(t *testing.T) {
	unmaskedProcMount := corev1.UnmaskedProcMount
	defaultProcMount := corev1.DefaultProcMount

	dst := &corev1.SecurityContext{
		ProcMount: &defaultProcMount,
	}
	include := &corev1.SecurityContext{
		ProcMount: &unmaskedProcMount,
	}

	result := MergeSecurityContext(dst, include)

	assert.Equal(t, &unmaskedProcMount, result.ProcMount, "ProcMount should be overridden")
}

func TestMergeSecurityContext_SeccompProfile(t *testing.T) {
	dst := &corev1.SecurityContext{
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeUnconfined,
		},
	}
	include := &corev1.SecurityContext{
		SeccompProfile: &corev1.SeccompProfile{
			Type:             corev1.SeccompProfileTypeLocalhost,
			LocalhostProfile: common.Ptr("my-profile.json"),
		},
	}

	result := MergeSecurityContext(dst, include)

	assert.NotNil(t, result.SeccompProfile)
	assert.Equal(t, corev1.SeccompProfileTypeLocalhost, result.SeccompProfile.Type)
	assert.Equal(t, common.Ptr("my-profile.json"), result.SeccompProfile.LocalhostProfile)
}

func TestMergeSecurityContext_AppArmorProfile(t *testing.T) {
	dst := &corev1.SecurityContext{
		AppArmorProfile: &corev1.AppArmorProfile{
			Type: corev1.AppArmorProfileTypeUnconfined,
		},
	}
	include := &corev1.SecurityContext{
		AppArmorProfile: &corev1.AppArmorProfile{
			Type:             corev1.AppArmorProfileTypeLocalhost,
			LocalhostProfile: common.Ptr("my-apparmor-profile"),
		},
	}

	result := MergeSecurityContext(dst, include)

	assert.NotNil(t, result.AppArmorProfile)
	assert.Equal(t, corev1.AppArmorProfileTypeLocalhost, result.AppArmorProfile.Type)
	assert.Equal(t, common.Ptr("my-apparmor-profile"), result.AppArmorProfile.LocalhostProfile)
}

func TestMergeSecurityContext_SELinuxOptions(t *testing.T) {
	dst := &corev1.SecurityContext{
		SELinuxOptions: &corev1.SELinuxOptions{
			User: "system_u",
			Role: "system_r",
		},
	}
	include := &corev1.SecurityContext{
		SELinuxOptions: &corev1.SELinuxOptions{
			Type:  "container_t",
			Level: "s0:c1,c2",
		},
	}

	result := MergeSecurityContext(dst, include)

	assert.NotNil(t, result.SELinuxOptions)
	assert.Equal(t, "system_u", result.SELinuxOptions.User)
	assert.Equal(t, "system_r", result.SELinuxOptions.Role)
	assert.Equal(t, "container_t", result.SELinuxOptions.Type)
	assert.Equal(t, "s0:c1,c2", result.SELinuxOptions.Level)
}

func TestMergeSecurityContext_AllFieldsCombined(t *testing.T) {
	unmaskedProcMount := corev1.UnmaskedProcMount

	dst := &corev1.SecurityContext{
		RunAsUser:    common.Ptr(int64(1000)),
		RunAsNonRoot: common.Ptr(true),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
	}
	include := &corev1.SecurityContext{
		RunAsGroup:               common.Ptr(int64(2000)),
		AllowPrivilegeEscalation: common.Ptr(false),
		ReadOnlyRootFilesystem:   common.Ptr(true),
		ProcMount:                &unmaskedProcMount,
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
		AppArmorProfile: &corev1.AppArmorProfile{
			Type: corev1.AppArmorProfileTypeRuntimeDefault,
		},
	}

	result := MergeSecurityContext(dst, include)

	// From dst
	assert.Equal(t, common.Ptr(int64(1000)), result.RunAsUser)
	assert.Equal(t, common.Ptr(true), result.RunAsNonRoot)
	assert.Contains(t, result.Capabilities.Drop, corev1.Capability("ALL"))

	// From include
	assert.Equal(t, common.Ptr(int64(2000)), result.RunAsGroup)
	assert.Equal(t, common.Ptr(false), result.AllowPrivilegeEscalation)
	assert.Equal(t, common.Ptr(true), result.ReadOnlyRootFilesystem)
	assert.Equal(t, &unmaskedProcMount, result.ProcMount)
	assert.Equal(t, corev1.SeccompProfileTypeRuntimeDefault, result.SeccompProfile.Type)
	assert.Equal(t, corev1.AppArmorProfileTypeRuntimeDefault, result.AppArmorProfile.Type)
}

func TestMergeContainerConfig_SecurityContext(t *testing.T) {
	t.Run("include securityContext when dst has none", func(t *testing.T) {
		dst := &testworkflowsv1.ContainerConfig{Image: "base:1.0"}
		include := &testworkflowsv1.ContainerConfig{
			SecurityContext: &corev1.SecurityContext{
				RunAsNonRoot: common.Ptr(true),
			},
		}

		result := MergeContainerConfig(dst, include)

		assert.NotNil(t, result.SecurityContext)
		assert.Equal(t, common.Ptr(true), result.SecurityContext.RunAsNonRoot)
		assert.Equal(t, "base:1.0", result.Image)
	})

	t.Run("merge securityContext when both have values", func(t *testing.T) {
		dst := &testworkflowsv1.ContainerConfig{
			SecurityContext: &corev1.SecurityContext{
				RunAsUser:    common.Ptr(int64(1000)),
				RunAsNonRoot: common.Ptr(true),
			},
		}
		include := &testworkflowsv1.ContainerConfig{
			SecurityContext: &corev1.SecurityContext{
				RunAsUser:              common.Ptr(int64(2000)),
				ReadOnlyRootFilesystem: common.Ptr(true),
			},
		}

		result := MergeContainerConfig(dst, include)

		assert.NotNil(t, result.SecurityContext)
		assert.Equal(t, common.Ptr(int64(2000)), result.SecurityContext.RunAsUser, "should be overridden")
		assert.Equal(t, common.Ptr(true), result.SecurityContext.RunAsNonRoot, "should be preserved")
		assert.Equal(t, common.Ptr(true), result.SecurityContext.ReadOnlyRootFilesystem, "should be added")
	})

	t.Run("preserve securityContext when include has none", func(t *testing.T) {
		dst := &testworkflowsv1.ContainerConfig{
			SecurityContext: &corev1.SecurityContext{
				RunAsNonRoot: common.Ptr(true),
			},
		}
		include := &testworkflowsv1.ContainerConfig{
			Image: "new:image",
		}

		result := MergeContainerConfig(dst, include)

		assert.NotNil(t, result.SecurityContext)
		assert.Equal(t, common.Ptr(true), result.SecurityContext.RunAsNonRoot)
	})
}
