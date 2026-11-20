// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowresolver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
)

func TestMergePodConfig_Resources(t *testing.T) {
	dst := &testworkflowsv1.PodConfig{
		Resources: &testworkflowsv1.Resources{
			Requests: map[corev1.ResourceName]testworkflowsv1.ConfigValue{corev1.ResourceCPU: "1", corev1.ResourceMemory: "1Gi"},
		},
	}
	include := &testworkflowsv1.PodConfig{
		Resources: &testworkflowsv1.Resources{
			Requests: map[corev1.ResourceName]testworkflowsv1.ConfigValue{corev1.ResourceCPU: "2"},
			Limits:   map[corev1.ResourceName]testworkflowsv1.ConfigValue{corev1.ResourceMemory: "2Gi"},
		},
	}

	got := MergePodConfig(dst, include)

	assert.Equal(t, &testworkflowsv1.Resources{
		Requests: map[corev1.ResourceName]testworkflowsv1.ConfigValue{corev1.ResourceCPU: "2", corev1.ResourceMemory: "1Gi"},
		Limits:   map[corev1.ResourceName]testworkflowsv1.ConfigValue{corev1.ResourceMemory: "2Gi"},
	}, got.Resources)
}

func TestMergePodConfig_ResourcesFromTemplateOnly(t *testing.T) {
	include := &testworkflowsv1.PodConfig{
		Resources: &testworkflowsv1.Resources{
			Limits: map[corev1.ResourceName]testworkflowsv1.ConfigValue{corev1.ResourceCPU: "4"},
		},
	}

	got := MergePodConfig(&testworkflowsv1.PodConfig{}, include)

	assert.Equal(t, include.Resources, got.Resources)
}
