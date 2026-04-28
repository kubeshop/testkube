package v1

import (
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/internal/common"
)

type WorkflowSecurityContext struct {
	Capabilities   *corev1.Capabilities                  `json:"capabilities,omitempty" expr:"force"`
	Privileged     *bool                                 `json:"privileged,omitempty" expr:"template"`
	SELinuxOptions *corev1.SELinuxOptions                `json:"seLinuxOptions,omitempty" expr:"force"`
	WindowsOptions *corev1.WindowsSecurityContextOptions `json:"windowsOptions,omitempty" expr:"force"`
	// +kubebuilder:validation:XIntOrString
	RunAsUser *WorkflowInt64OrString `json:"runAsUser,omitempty" expr:"template"`
	// +kubebuilder:validation:XIntOrString
	RunAsGroup               *WorkflowInt64OrString  `json:"runAsGroup,omitempty" expr:"template"`
	RunAsNonRoot             *bool                   `json:"runAsNonRoot,omitempty" expr:"template"`
	ReadOnlyRootFilesystem   *bool                   `json:"readOnlyRootFilesystem,omitempty" expr:"template"`
	AllowPrivilegeEscalation *bool                   `json:"allowPrivilegeEscalation,omitempty" expr:"template"`
	ProcMount                *corev1.ProcMountType   `json:"procMount,omitempty" expr:"template"`
	SeccompProfile           *corev1.SeccompProfile  `json:"seccompProfile,omitempty" expr:"force"`
	AppArmorProfile          *corev1.AppArmorProfile `json:"appArmorProfile,omitempty" expr:"force"`
}

type WorkflowPodSecurityContext struct {
	SELinuxOptions *corev1.SELinuxOptions                `json:"seLinuxOptions,omitempty" expr:"force"`
	WindowsOptions *corev1.WindowsSecurityContextOptions `json:"windowsOptions,omitempty" expr:"force"`
	// +kubebuilder:validation:XIntOrString
	RunAsUser *WorkflowInt64OrString `json:"runAsUser,omitempty" expr:"template"`
	// +kubebuilder:validation:XIntOrString
	RunAsGroup               *WorkflowInt64OrString           `json:"runAsGroup,omitempty" expr:"template"`
	RunAsNonRoot             *bool                            `json:"runAsNonRoot,omitempty" expr:"template"`
	SupplementalGroups       []int64                          `json:"supplementalGroups,omitempty"`
	SupplementalGroupsPolicy *corev1.SupplementalGroupsPolicy `json:"supplementalGroupsPolicy,omitempty" expr:"template"`
	// +kubebuilder:validation:XIntOrString
	FSGroup             *WorkflowInt64OrString         `json:"fsGroup,omitempty" expr:"template"`
	Sysctls             []corev1.Sysctl                `json:"sysctls,omitempty" expr:"force"`
	FSGroupChangePolicy *corev1.PodFSGroupChangePolicy `json:"fsGroupChangePolicy,omitempty" expr:"template"`
	SeccompProfile      *corev1.SeccompProfile         `json:"seccompProfile,omitempty" expr:"force"`
	AppArmorProfile     *corev1.AppArmorProfile        `json:"appArmorProfile,omitempty" expr:"force"`
	SELinuxChangePolicy *corev1.PodSELinuxChangePolicy `json:"seLinuxChangePolicy,omitempty" expr:"template"`
}

func CloneWorkflowSecurityContext(v *WorkflowSecurityContext) *WorkflowSecurityContext {
	if v == nil {
		return nil
	}
	result := &WorkflowSecurityContext{
		Privileged:               v.Privileged,
		RunAsUser:                common.MapPtr(v.RunAsUser, func(i WorkflowInt64OrString) WorkflowInt64OrString { return i }),
		RunAsGroup:               common.MapPtr(v.RunAsGroup, func(i WorkflowInt64OrString) WorkflowInt64OrString { return i }),
		RunAsNonRoot:             v.RunAsNonRoot,
		ReadOnlyRootFilesystem:   v.ReadOnlyRootFilesystem,
		AllowPrivilegeEscalation: v.AllowPrivilegeEscalation,
		ProcMount:                v.ProcMount,
	}
	if v.Capabilities != nil {
		result.Capabilities = v.Capabilities.DeepCopy()
	}
	if v.SELinuxOptions != nil {
		result.SELinuxOptions = v.SELinuxOptions.DeepCopy()
	}
	if v.WindowsOptions != nil {
		result.WindowsOptions = v.WindowsOptions.DeepCopy()
	}
	if v.SeccompProfile != nil {
		result.SeccompProfile = v.SeccompProfile.DeepCopy()
	}
	if v.AppArmorProfile != nil {
		result.AppArmorProfile = v.AppArmorProfile.DeepCopy()
	}
	return result
}

func CloneWorkflowPodSecurityContext(v *WorkflowPodSecurityContext) *WorkflowPodSecurityContext {
	if v == nil {
		return nil
	}
	result := &WorkflowPodSecurityContext{
		RunAsUser:                common.MapPtr(v.RunAsUser, func(i WorkflowInt64OrString) WorkflowInt64OrString { return i }),
		RunAsGroup:               common.MapPtr(v.RunAsGroup, func(i WorkflowInt64OrString) WorkflowInt64OrString { return i }),
		RunAsNonRoot:             v.RunAsNonRoot,
		SupplementalGroups:       append([]int64(nil), v.SupplementalGroups...),
		SupplementalGroupsPolicy: v.SupplementalGroupsPolicy,
		FSGroup:                  common.MapPtr(v.FSGroup, func(i WorkflowInt64OrString) WorkflowInt64OrString { return i }),
		Sysctls:                  append([]corev1.Sysctl(nil), v.Sysctls...),
		FSGroupChangePolicy:      v.FSGroupChangePolicy,
		SELinuxChangePolicy:      v.SELinuxChangePolicy,
	}
	if v.SELinuxOptions != nil {
		result.SELinuxOptions = v.SELinuxOptions.DeepCopy()
	}
	if v.WindowsOptions != nil {
		result.WindowsOptions = v.WindowsOptions.DeepCopy()
	}
	if v.SeccompProfile != nil {
		result.SeccompProfile = v.SeccompProfile.DeepCopy()
	}
	if v.AppArmorProfile != nil {
		result.AppArmorProfile = v.AppArmorProfile.DeepCopy()
	}
	return result
}

func ResolveWorkflowInt64(fieldPath string, value *WorkflowInt64OrString) (*int64, error) {
	if value == nil {
		return nil, nil
	}
	parsed, err := strconv.ParseInt(value.String(), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%s must resolve to an integer: %w", fieldPath, err)
	}
	return &parsed, nil
}

func Int64ToWorkflowIntOrString(value *int64) *WorkflowInt64OrString {
	if value == nil {
		return nil
	}
	return NewWorkflowInt64OrString(strconv.FormatInt(*value, 10))
}

func WorkflowSecurityContextFromKube(v *corev1.SecurityContext) *WorkflowSecurityContext {
	if v == nil {
		return nil
	}
	return &WorkflowSecurityContext{
		Capabilities:             v.Capabilities,
		Privileged:               v.Privileged,
		SELinuxOptions:           v.SELinuxOptions,
		WindowsOptions:           v.WindowsOptions,
		RunAsUser:                Int64ToWorkflowIntOrString(v.RunAsUser),
		RunAsGroup:               Int64ToWorkflowIntOrString(v.RunAsGroup),
		RunAsNonRoot:             v.RunAsNonRoot,
		ReadOnlyRootFilesystem:   v.ReadOnlyRootFilesystem,
		AllowPrivilegeEscalation: v.AllowPrivilegeEscalation,
		ProcMount:                v.ProcMount,
		SeccompProfile:           v.SeccompProfile,
		AppArmorProfile:          v.AppArmorProfile,
	}
}

func WorkflowPodSecurityContextFromKube(v *corev1.PodSecurityContext) *WorkflowPodSecurityContext {
	if v == nil {
		return nil
	}
	return &WorkflowPodSecurityContext{
		SELinuxOptions:           v.SELinuxOptions,
		WindowsOptions:           v.WindowsOptions,
		RunAsUser:                Int64ToWorkflowIntOrString(v.RunAsUser),
		RunAsGroup:               Int64ToWorkflowIntOrString(v.RunAsGroup),
		RunAsNonRoot:             v.RunAsNonRoot,
		SupplementalGroups:       v.SupplementalGroups,
		SupplementalGroupsPolicy: v.SupplementalGroupsPolicy,
		FSGroup:                  Int64ToWorkflowIntOrString(v.FSGroup),
		Sysctls:                  v.Sysctls,
		FSGroupChangePolicy:      v.FSGroupChangePolicy,
		SeccompProfile:           v.SeccompProfile,
		AppArmorProfile:          v.AppArmorProfile,
		SELinuxChangePolicy:      v.SELinuxChangePolicy,
	}
}

func (v *WorkflowSecurityContext) ToKube() (*corev1.SecurityContext, error) {
	if v == nil {
		return nil, nil
	}

	runAsUser, err := ResolveWorkflowInt64("container.securityContext.runAsUser", v.RunAsUser)
	if err != nil {
		return nil, err
	}
	runAsGroup, err := ResolveWorkflowInt64("container.securityContext.runAsGroup", v.RunAsGroup)
	if err != nil {
		return nil, err
	}

	return &corev1.SecurityContext{
		Capabilities:             v.Capabilities,
		Privileged:               v.Privileged,
		SELinuxOptions:           v.SELinuxOptions,
		WindowsOptions:           v.WindowsOptions,
		RunAsUser:                runAsUser,
		RunAsGroup:               runAsGroup,
		RunAsNonRoot:             v.RunAsNonRoot,
		ReadOnlyRootFilesystem:   v.ReadOnlyRootFilesystem,
		AllowPrivilegeEscalation: v.AllowPrivilegeEscalation,
		ProcMount:                v.ProcMount,
		SeccompProfile:           v.SeccompProfile,
		AppArmorProfile:          v.AppArmorProfile,
	}, nil
}

func (v *WorkflowPodSecurityContext) ToKube() (*corev1.PodSecurityContext, error) {
	if v == nil {
		return nil, nil
	}

	runAsUser, err := ResolveWorkflowInt64("pod.securityContext.runAsUser", v.RunAsUser)
	if err != nil {
		return nil, err
	}
	runAsGroup, err := ResolveWorkflowInt64("pod.securityContext.runAsGroup", v.RunAsGroup)
	if err != nil {
		return nil, err
	}
	fsGroup, err := ResolveWorkflowInt64("pod.securityContext.fsGroup", v.FSGroup)
	if err != nil {
		return nil, err
	}

	return &corev1.PodSecurityContext{
		SELinuxOptions:           v.SELinuxOptions,
		WindowsOptions:           v.WindowsOptions,
		RunAsUser:                runAsUser,
		RunAsGroup:               runAsGroup,
		RunAsNonRoot:             v.RunAsNonRoot,
		SupplementalGroups:       v.SupplementalGroups,
		SupplementalGroupsPolicy: v.SupplementalGroupsPolicy,
		FSGroup:                  fsGroup,
		Sysctls:                  v.Sysctls,
		FSGroupChangePolicy:      v.FSGroupChangePolicy,
		SeccompProfile:           v.SeccompProfile,
		AppArmorProfile:          v.AppArmorProfile,
		SELinuxChangePolicy:      v.SELinuxChangePolicy,
	}, nil
}
