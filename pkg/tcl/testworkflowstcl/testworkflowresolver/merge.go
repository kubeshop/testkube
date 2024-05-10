// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowresolver

import (
	"maps"

	corev1 "k8s.io/api/core/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
)

func MergePodConfig(dst, include *testworkflowsv1.PodConfig) *testworkflowsv1.PodConfig {
	if dst == nil {
		return include
	} else if include == nil {
		return dst
	}
	dst.Labels = MergeMap(dst.Labels, include.Labels)
	dst.Annotations = MergeMap(dst.Annotations, include.Annotations)
	dst.NodeSelector = MergeMap(dst.NodeSelector, include.NodeSelector)
	dst.Volumes = append(dst.Volumes, include.Volumes...)
	dst.ImagePullSecrets = append(dst.ImagePullSecrets, include.ImagePullSecrets...)
	if include.ServiceAccountName != "" {
		dst.ServiceAccountName = include.ServiceAccountName
	}
	if include.ActiveDeadlineSeconds != nil {
		dst.ActiveDeadlineSeconds = include.ActiveDeadlineSeconds
	}
	if include.DNSPolicy != "" {
		dst.DNSPolicy = include.DNSPolicy
	}
	if include.NodeName != "" {
		dst.NodeName = include.NodeName
	}
	dst.SecurityContext = MergePodSecurityContext(dst.SecurityContext, include.SecurityContext)
	if include.Hostname != "" {
		dst.Hostname = include.Hostname
	}
	if include.Subdomain != "" {
		dst.Subdomain = include.Subdomain
	}
	dst.Affinity = MergeAffinity(dst.Affinity, include.Affinity)
	dst.Tolerations = append(dst.Tolerations, include.Tolerations...)
	dst.HostAliases = append(dst.HostAliases, include.HostAliases...)
	if include.PriorityClassName != "" {
		dst.PriorityClassName = include.PriorityClassName
	}
	if include.Priority != nil {
		dst.Priority = include.Priority
	}
	dst.DNSConfig = MergePodDNSConfig(dst.DNSConfig, include.DNSConfig)
	if include.PreemptionPolicy != nil {
		dst.PreemptionPolicy = include.PreemptionPolicy
	}
	dst.TopologySpreadConstraints = append(dst.TopologySpreadConstraints, include.TopologySpreadConstraints...)
	dst.SchedulingGates = append(dst.SchedulingGates, include.SchedulingGates...)
	dst.ResourceClaims = append(include.ResourceClaims, dst.ResourceClaims...)
	return dst
}

func MergeJobConfig(dst, include *testworkflowsv1.JobConfig) *testworkflowsv1.JobConfig {
	if dst == nil {
		return include
	} else if include == nil {
		return dst
	}
	dst.Labels = MergeMap(dst.Labels, include.Labels)
	dst.Annotations = MergeMap(dst.Annotations, include.Annotations)
	if include.Namespace != "" {
		dst.Namespace = include.Namespace
	}
	if include.ActiveDeadlineSeconds != nil {
		dst.ActiveDeadlineSeconds = include.ActiveDeadlineSeconds
	}
	return dst
}

func MergePodDNSConfig(dst, include *corev1.PodDNSConfig) *corev1.PodDNSConfig {
	if dst == nil {
		return include
	} else if include == nil {
		return dst
	}
	dst.Nameservers = append(dst.Nameservers, include.Nameservers...)
	dst.Searches = append(dst.Searches, include.Searches...)
	dst.Options = append(dst.Options, include.Options...)
	return dst
}

func MergePodSecurityContext(dst, include *corev1.PodSecurityContext) *corev1.PodSecurityContext {
	if dst == nil {
		return include
	} else if include == nil {
		return dst
	}
	dst.SELinuxOptions = MergeSELinuxOptions(dst.SELinuxOptions, include.SELinuxOptions)
	dst.WindowsOptions = MergeWindowsSecurityContextOptions(dst.WindowsOptions, include.WindowsOptions)
	if include.RunAsUser != nil {
		dst.RunAsUser = include.RunAsUser
	}
	if include.RunAsGroup != nil {
		dst.RunAsGroup = include.RunAsGroup
	}
	if include.RunAsNonRoot != nil {
		dst.RunAsNonRoot = include.RunAsNonRoot
	}
	dst.SupplementalGroups = append(dst.SupplementalGroups, include.SupplementalGroups...)
	if include.FSGroup != nil {
		dst.FSGroup = include.FSGroup
	}
	if include.FSGroupChangePolicy != nil {
		dst.FSGroupChangePolicy = include.FSGroupChangePolicy
	}
	if include.SeccompProfile != nil {
		dst.SeccompProfile = include.SeccompProfile
	}
	return dst
}

func MergeNodeSelector(dst, include *corev1.NodeSelector) *corev1.NodeSelector {
	if dst == nil {
		return include
	} else if include == nil {
		return dst
	}
	dst.NodeSelectorTerms = append(dst.NodeSelectorTerms, include.NodeSelectorTerms...)
	return dst
}

func MergeNodeAffinity(dst, include *corev1.NodeAffinity) *corev1.NodeAffinity {
	if dst == nil {
		return include
	} else if include == nil {
		return dst
	}
	dst.RequiredDuringSchedulingIgnoredDuringExecution = MergeNodeSelector(dst.RequiredDuringSchedulingIgnoredDuringExecution, include.RequiredDuringSchedulingIgnoredDuringExecution)
	dst.PreferredDuringSchedulingIgnoredDuringExecution = append(dst.PreferredDuringSchedulingIgnoredDuringExecution, include.PreferredDuringSchedulingIgnoredDuringExecution...)
	return dst
}

func MergePodAffinity(dst, include *corev1.PodAffinity) *corev1.PodAffinity {
	if dst == nil {
		return include
	} else if include == nil {
		return dst
	}
	dst.RequiredDuringSchedulingIgnoredDuringExecution = append(dst.RequiredDuringSchedulingIgnoredDuringExecution, include.RequiredDuringSchedulingIgnoredDuringExecution...)
	dst.PreferredDuringSchedulingIgnoredDuringExecution = append(dst.PreferredDuringSchedulingIgnoredDuringExecution, include.PreferredDuringSchedulingIgnoredDuringExecution...)
	return dst
}

func MergePodAntiAffinity(dst, include *corev1.PodAntiAffinity) *corev1.PodAntiAffinity {
	if dst == nil {
		return include
	} else if include == nil {
		return dst
	}
	dst.RequiredDuringSchedulingIgnoredDuringExecution = append(dst.RequiredDuringSchedulingIgnoredDuringExecution, include.RequiredDuringSchedulingIgnoredDuringExecution...)
	dst.PreferredDuringSchedulingIgnoredDuringExecution = append(dst.PreferredDuringSchedulingIgnoredDuringExecution, include.PreferredDuringSchedulingIgnoredDuringExecution...)
	return dst
}

func MergeAffinity(dst, include *corev1.Affinity) *corev1.Affinity {
	if dst == nil {
		return include
	} else if include == nil {
		return dst
	}
	dst.NodeAffinity = MergeNodeAffinity(dst.NodeAffinity, include.NodeAffinity)
	dst.PodAffinity = MergePodAffinity(dst.PodAffinity, include.PodAffinity)
	dst.PodAntiAffinity = MergePodAntiAffinity(dst.PodAntiAffinity, include.PodAntiAffinity)
	return dst
}

func MergeContentGit(dst, include *testworkflowsv1.ContentGit) *testworkflowsv1.ContentGit {
	if dst == nil {
		return include
	} else if include == nil {
		return dst
	}
	return include
}

func MergeCapabilities(dst, include *corev1.Capabilities) *corev1.Capabilities {
	if dst == nil {
		return include
	} else if include == nil {
		return dst
	}
	dst.Add = append(dst.Add, include.Add...)
	dst.Drop = append(dst.Drop, include.Drop...)
	return dst
}

func MergeSELinuxOptions(dst, include *corev1.SELinuxOptions) *corev1.SELinuxOptions {
	if dst == nil {
		return include
	} else if include == nil {
		return dst
	}
	if include.User != "" {
		dst.User = include.User
	}
	if include.Role != "" {
		dst.Role = include.Role
	}
	if include.Type != "" {
		dst.Type = include.Type
	}
	if include.Level != "" {
		dst.Level = include.Level
	}
	return dst
}

func MergeWindowsSecurityContextOptions(dst, include *corev1.WindowsSecurityContextOptions) *corev1.WindowsSecurityContextOptions {
	if dst == nil {
		return include
	} else if include == nil {
		return dst
	}
	if include.GMSACredentialSpecName != nil {
		dst.GMSACredentialSpecName = include.GMSACredentialSpecName
	}
	if include.GMSACredentialSpec != nil {
		dst.GMSACredentialSpec = include.GMSACredentialSpec
	}
	if include.RunAsUserName != nil {
		dst.RunAsUserName = include.RunAsUserName
	}
	if include.HostProcess != nil {
		dst.HostProcess = include.HostProcess
	}
	return dst
}

func MergeSecurityContext(dst, include *corev1.SecurityContext) *corev1.SecurityContext {
	if dst == nil {
		return include
	} else if include == nil {
		return dst
	}
	dst.Capabilities = MergeCapabilities(dst.Capabilities, include.Capabilities)
	if include.Privileged != nil {
		dst.Privileged = include.Privileged
	}
	dst.SELinuxOptions = MergeSELinuxOptions(dst.SELinuxOptions, include.SELinuxOptions)
	dst.WindowsOptions = MergeWindowsSecurityContextOptions(dst.WindowsOptions, include.WindowsOptions)
	if include.RunAsUser != nil {
		dst.RunAsUser = include.RunAsUser
	}
	if include.RunAsGroup != nil {
		dst.RunAsGroup = include.RunAsGroup
	}
	if include.RunAsNonRoot != nil {
		dst.RunAsNonRoot = include.RunAsNonRoot
	}
	if include.ReadOnlyRootFilesystem != nil {
		dst.ReadOnlyRootFilesystem = include.ReadOnlyRootFilesystem
	}
	if include.AllowPrivilegeEscalation != nil {
		dst.AllowPrivilegeEscalation = include.AllowPrivilegeEscalation
	}
	if include.ProcMount != nil {
		dst.ProcMount = include.ProcMount
	}
	if include.SeccompProfile != nil {
		dst.SeccompProfile = include.SeccompProfile
	}
	return dst
}

func MergeContent(dst, include *testworkflowsv1.Content) *testworkflowsv1.Content {
	if dst == nil {
		return include
	} else if include == nil {
		return dst
	}
	dst.Files = append(dst.Files, include.Files...)
	dst.Git = MergeContentGit(dst.Git, include.Git)
	dst.Tarball = append(dst.Tarball, include.Tarball...)
	return dst
}

func MergeResources(dst, include *testworkflowsv1.Resources) *testworkflowsv1.Resources {
	if dst == nil {
		return include
	} else if include == nil {
		return dst
	}
	dst.Requests = MergeMap(dst.Requests, include.Requests)
	dst.Limits = MergeMap(dst.Limits, include.Limits)
	return dst
}

func MergeContainerConfig(dst, include *testworkflowsv1.ContainerConfig) *testworkflowsv1.ContainerConfig {
	if dst == nil {
		return include
	} else if include == nil {
		return dst
	}
	if include.WorkingDir != nil {
		dst.WorkingDir = include.WorkingDir
	}
	if include.ImagePullPolicy != "" {
		dst.ImagePullPolicy = include.ImagePullPolicy
	}
	dst.Env = append(dst.Env, include.Env...)
	dst.EnvFrom = append(dst.EnvFrom, include.EnvFrom...)
	dst.VolumeMounts = append(dst.VolumeMounts, include.VolumeMounts...)
	if include.Image != "" {
		dst.Image = include.Image
		dst.Command = include.Command
		dst.Args = include.Args
	} else if include.Command != nil {
		dst.Command = include.Command
		dst.Args = include.Args
	} else if include.Args != nil {
		dst.Args = include.Args
	}
	dst.Resources = MergeResources(dst.Resources, include.Resources)
	dst.SecurityContext = MergeSecurityContext(dst.SecurityContext, include.SecurityContext)
	return dst
}

func MergeMap[T comparable, U any](dst, include map[T]U) map[T]U {
	if include == nil {
		return dst
	} else if dst == nil {
		return include
	}
	maps.Copy(dst, include)
	return dst
}

func ConvertIndependentStepToStep(step testworkflowsv1.IndependentStep) (res testworkflowsv1.Step) {
	res.StepBase = step.StepBase
	res.Setup = common.MapSlice(step.Setup, ConvertIndependentStepToStep)
	res.Steps = common.MapSlice(step.Steps, ConvertIndependentStepToStep)
	return res
}
