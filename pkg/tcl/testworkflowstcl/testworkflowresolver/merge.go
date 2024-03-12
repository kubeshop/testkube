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
	"k8s.io/apimachinery/pkg/util/intstr"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
)

func MergePodConfig(dst, include *testworkflowsv1.PodConfig) *testworkflowsv1.PodConfig {
	if dst == nil {
		return include
	} else if include == nil {
		return dst
	}
	if len(include.Labels) > 0 && dst.Labels == nil {
		dst.Labels = map[string]string{}
	}
	maps.Copy(dst.Labels, include.Labels)
	if len(include.Annotations) > 0 && dst.Annotations == nil {
		dst.Annotations = map[string]string{}
	}
	maps.Copy(dst.Annotations, include.Annotations)
	if len(include.NodeSelector) > 0 && dst.NodeSelector == nil {
		dst.NodeSelector = map[string]string{}
	}
	maps.Copy(dst.NodeSelector, include.NodeSelector)
	dst.Volumes = append(dst.Volumes, include.Volumes...)
	dst.ImagePullSecrets = append(dst.ImagePullSecrets, include.ImagePullSecrets...)
	if include.ServiceAccountName != "" {
		dst.ServiceAccountName = include.ServiceAccountName
	}
	return dst
}

func MergeJobConfig(dst, include *testworkflowsv1.JobConfig) *testworkflowsv1.JobConfig {
	if dst == nil {
		return include
	} else if include == nil {
		return dst
	}
	if len(include.Labels) > 0 && dst.Labels == nil {
		dst.Labels = map[string]string{}
	}
	maps.Copy(dst.Labels, include.Labels)
	if len(include.Annotations) > 0 && dst.Annotations == nil {
		dst.Annotations = map[string]string{}
	}
	maps.Copy(dst.Annotations, include.Annotations)
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

func MergeSecurityContext(dst, include *corev1.SecurityContext) *corev1.SecurityContext {
	if dst == nil {
		return include
	} else if include == nil {
		return dst
	}
	return include
}

func MergeContent(dst, include *testworkflowsv1.Content) *testworkflowsv1.Content {
	if dst == nil {
		return include
	} else if include == nil {
		return dst
	}
	dst.Files = append(dst.Files, include.Files...)
	dst.Git = MergeContentGit(dst.Git, include.Git)
	return dst
}

func MergeResources(dst, include *testworkflowsv1.Resources) *testworkflowsv1.Resources {
	if dst == nil {
		return include
	} else if include == nil {
		return dst
	}
	if dst.Requests == nil && len(include.Requests) > 0 {
		dst.Requests = map[corev1.ResourceName]intstr.IntOrString{}
	}
	if dst.Limits == nil && len(include.Limits) > 0 {
		dst.Limits = map[corev1.ResourceName]intstr.IntOrString{}
	}
	maps.Copy(dst.Requests, include.Requests)
	maps.Copy(dst.Limits, include.Limits)
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

func ConvertIndependentStepToStep(step testworkflowsv1.IndependentStep) (res testworkflowsv1.Step) {
	res.StepBase = step.StepBase
	res.Setup = common.MapSlice(step.Setup, ConvertIndependentStepToStep)
	res.Steps = common.MapSlice(step.Steps, ConvertIndependentStepToStep)
	return res
}
