// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflows

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
)

var (
	container = testworkflowsv1.ContainerConfig{
		WorkingDir:      common.Ptr("/wd"),
		Image:           "some-image",
		ImagePullPolicy: "IfNotPresent",
		Env: []corev1.EnvVar{
			{Name: "some-naaame", Value: "some-value"},
			{Name: "some-naaame", ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					APIVersion: "api.value.1",
					FieldPath:  "the.field.pa",
				},
				ResourceFieldRef: &corev1.ResourceFieldSelector{
					ContainerName: "con-name",
					Resource:      "anc",
				},
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "cfg-name"},
					Key:                  "cfg-key",
				},
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "some-sec"},
					Key:                  "sec-key",
				},
			}},
		},
		EnvFrom: []corev1.EnvFromSource{
			{
				Prefix: "some-prefix",
				ConfigMapRef: &corev1.ConfigMapEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "some-name",
					},
				},
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "some-sec",
					},
					Optional: common.Ptr(true),
				},
			},
		},
		Command: common.Ptr([]string{"c", "d"}),
		Args:    common.Ptr([]string{"ar", "gs"}),
		Resources: &testworkflowsv1.Resources{
			Limits: map[corev1.ResourceName]intstr.IntOrString{
				corev1.ResourceCPU:    {Type: intstr.String, StrVal: "300m"},
				corev1.ResourceMemory: {Type: intstr.Int, IntVal: 1024},
			},
			Requests: map[corev1.ResourceName]intstr.IntOrString{
				corev1.ResourceCPU:    {Type: intstr.String, StrVal: "3800m"},
				corev1.ResourceMemory: {Type: intstr.Int, IntVal: 10204},
			},
		},
		SecurityContext: &corev1.SecurityContext{
			RunAsUser:                common.Ptr(int64(334)),
			RunAsGroup:               common.Ptr(int64(11)),
			RunAsNonRoot:             common.Ptr(true),
			ReadOnlyRootFilesystem:   common.Ptr(false),
			AllowPrivilegeEscalation: nil,
		},
	}
	content = testworkflowsv1.Content{
		Git: &testworkflowsv1.ContentGit{
			Uri:      "some-uri",
			Revision: "some-revision",
			Username: "some-username",
			UsernameFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					APIVersion: "testworkflows.dummy.io/v1",
					FieldPath:  "the.field.path",
				},
				ResourceFieldRef: &corev1.ResourceFieldSelector{
					ContainerName: "container.name",
					Resource:      "the.resource",
					Divisor:       resource.MustParse("300"),
				},
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "the-name-config"},
					Key:                  "the-key",
					Optional:             common.Ptr(true),
				},
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "the-name-secret"},
					Key:                  "the-key-secret",
					Optional:             nil,
				},
			},
			Token: "the-token",
			TokenFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					APIVersion: "some.dummy.api/v1",
					FieldPath:  "some.field",
				},
				ResourceFieldRef: &corev1.ResourceFieldSelector{
					ContainerName: "some-container-name",
					Resource:      "some-resource",
					Divisor:       resource.MustParse("200"),
				},
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "the-name"},
					Key:                  "the-abc",
					Optional:             nil,
				},
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "xyz"},
					Key:                  "222",
					Optional:             nil,
				},
			},
			AuthType:  "basic",
			MountPath: "/some/output/path",
			Paths:     []string{"a", "b", "c"},
		},
		Files: []testworkflowsv1.ContentFile{
			{
				Path:    "some-path",
				Content: "some-content",
				ContentFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "api.version.abc",
						FieldPath:  "field.path",
					},
				},
				Mode: common.Ptr(int32(0777)),
			},
		},
	}
	stepBase = testworkflowsv1.StepBase{
		Name:      "some-name",
		Condition: "some-condition",
		Negative:  true,
		Optional:  false,
		Retry: &testworkflowsv1.RetryPolicy{
			Count: 444,
			Until: "abc",
		},
		Timeout: "3h15m",
		Delay:   "2m40s",
		Content: &testworkflowsv1.Content{
			Git: &testworkflowsv1.ContentGit{
				Uri:      "some-url",
				Revision: "another-rev",
				Username: "some-username",
				UsernameFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "dummy.api",
						FieldPath:  "field.path.there",
					},
					ResourceFieldRef: &corev1.ResourceFieldSelector{
						ContainerName: "con-name",
						Resource:      "abc1",
					},
				},
				Token: "",
				TokenFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "test.v1",
						FieldPath:  "abc.there",
					},
				},
				AuthType:  "basic",
				MountPath: "/a/b/c",
				Paths:     []string{"p", "a", "th"},
			},
			Files: []testworkflowsv1.ContentFile{
				{Path: "abc", Content: "some-content"},
			},
		},
		Shell: "shell-to-run",
		Run: &testworkflowsv1.StepRun{
			ContainerConfig: testworkflowsv1.ContainerConfig{
				WorkingDir:      common.Ptr("/abc"),
				Image:           "im-g",
				ImagePullPolicy: "IfNotPresent",
				Env: []corev1.EnvVar{
					{Name: "abc", Value: "230"},
				},
				EnvFrom: []corev1.EnvFromSource{
					{Prefix: "abc"},
				},
				Command: common.Ptr([]string{"c", "m", "d"}),
				Args:    common.Ptr([]string{"arg", "s", "d"}),
				Resources: &testworkflowsv1.Resources{
					Limits: map[corev1.ResourceName]intstr.IntOrString{
						corev1.ResourceCPU: {Type: intstr.Int, IntVal: 444},
					},
				},
				SecurityContext: &corev1.SecurityContext{
					RunAsUser:                common.Ptr(int64(444)),
					RunAsGroup:               nil,
					RunAsNonRoot:             common.Ptr(true),
					ReadOnlyRootFilesystem:   nil,
					AllowPrivilegeEscalation: nil,
				},
			},
		},
		WorkingDir: common.Ptr("/ssss"),
		Container: &testworkflowsv1.ContainerConfig{
			WorkingDir:      common.Ptr("/aaaa"),
			Image:           "ssss",
			ImagePullPolicy: "Never",
			Env:             []corev1.EnvVar{{Name: "xyz", Value: "bar"}},
			Command:         common.Ptr([]string{"ab"}),
			Args:            common.Ptr([]string{"abrgs"}),
			Resources: &testworkflowsv1.Resources{
				Requests: map[corev1.ResourceName]intstr.IntOrString{
					corev1.ResourceMemory: {Type: intstr.String, StrVal: "300m"},
				},
			},
			SecurityContext: &corev1.SecurityContext{
				Privileged: common.Ptr(true),
				RunAsUser:  common.Ptr(int64(33)),
			},
		},
		Execute: &testworkflowsv1.StepExecute{
			Parallelism: 880,
			Async:       false,
			Tests:       []testworkflowsv1.StepExecuteTest{{Name: "some-name-test"}},
			Workflows: []testworkflowsv1.StepExecuteWorkflow{{Name: "some-workflow", Config: map[string]intstr.IntOrString{
				"id": {Type: intstr.String, StrVal: "xyzz"},
			}}},
		},
		Artifacts: &testworkflowsv1.StepArtifacts{
			Compress: &testworkflowsv1.ArtifactCompression{
				Name: "some-artifact.tar.gz",
			},
			Paths: []string{"/get", "/from/there"},
		},
	}
	step = testworkflowsv1.Step{
		StepBase: stepBase,
		Use: []testworkflowsv1.TemplateRef{
			{Name: "/abc", Config: map[string]intstr.IntOrString{
				"xxx": {Type: intstr.Int, IntVal: 322},
			}},
		},
		Template: &testworkflowsv1.TemplateRef{
			Name: "other-one",
			Config: map[string]intstr.IntOrString{
				"foo": {Type: intstr.String, StrVal: "bar"},
			},
		},
		Steps: []testworkflowsv1.Step{
			{StepBase: testworkflowsv1.StepBase{Name: "xyz"}},
		},
	}
	independentStep = testworkflowsv1.IndependentStep{
		StepBase: stepBase,
		Steps: []testworkflowsv1.IndependentStep{
			{StepBase: testworkflowsv1.StepBase{Name: "xyz"}},
		},
	}
	workflowSpecBase = testworkflowsv1.TestWorkflowSpecBase{
		Config: map[string]testworkflowsv1.ParameterSchema{
			"some-key": {
				Description: "some-description",
				Type:        "integer",
				Enum:        []string{"en", "um"},
				Example: &intstr.IntOrString{
					Type:   intstr.String,
					StrVal: "some-vale",
				},
				Default: &intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 233,
				},
				ParameterStringSchema: testworkflowsv1.ParameterStringSchema{
					Format:    "url",
					Pattern:   "^abc$",
					MinLength: common.Ptr(int64(1)),
					MaxLength: common.Ptr(int64(2)),
				},
				ParameterNumberSchema: testworkflowsv1.ParameterNumberSchema{
					Minimum:          common.Ptr(int64(3)),
					Maximum:          common.Ptr(int64(4)),
					ExclusiveMinimum: common.Ptr(int64(5)),
					ExclusiveMaximum: common.Ptr(int64(7)),
					MultipleOf:       common.Ptr(int64(8)),
				},
			},
		},
		Content:   &content,
		Container: &container,
		Job: &testworkflowsv1.JobConfig{
			Labels:      map[string]string{"some-key": "some-value"},
			Annotations: map[string]string{"some-key=2": "some-value-2"},
		},
		Pod: &testworkflowsv1.PodConfig{
			ServiceAccountName: "some-name",
			ImagePullSecrets:   []corev1.LocalObjectReference{{Name: "v1"}, {Name: "v2"}},
			NodeSelector:       map[string]string{"some-key-3": "some-value"},
			Labels:             map[string]string{"some-key-4": "some-value"},
			Annotations:        map[string]string{"some-key=5": "some-value-2"},
		},
	}
)

func TestMapTestWorkflowBackAndForth(t *testing.T) {
	want := testworkflowsv1.TestWorkflow{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TestWorkflow",
			APIVersion: "testworkflows.testkube.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dummy",
			Namespace: "dummy-namespace",
		},
		Spec: testworkflowsv1.TestWorkflowSpec{
			Use: []testworkflowsv1.TemplateRef{
				{
					Name: "some-name",
					Config: map[string]intstr.IntOrString{
						"some-key":   {Type: intstr.String, StrVal: "some-value"},
						"some-key-2": {Type: intstr.Int, IntVal: 444},
					},
				},
			},
			TestWorkflowSpecBase: workflowSpecBase,
			Setup:                []testworkflowsv1.Step{step},
			Steps:                []testworkflowsv1.Step{step, step},
			After:                []testworkflowsv1.Step{step, step, step, step},
		},
	}
	got := MapTestWorkflowAPIToKube(MapTestWorkflowKubeToAPI(*want.DeepCopy()))
	assert.Equal(t, want, got)
}

func TestMapEmptyTestWorkflowBackAndForth(t *testing.T) {
	want := testworkflowsv1.TestWorkflow{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TestWorkflow",
			APIVersion: "testworkflows.testkube.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dummy",
			Namespace: "dummy-namespace",
		},
		Spec: testworkflowsv1.TestWorkflowSpec{},
	}
	got := MapTestWorkflowAPIToKube(MapTestWorkflowKubeToAPI(*want.DeepCopy()))
	assert.Equal(t, want, got)
}

func TestMapTestWorkflowTemplateBackAndForth(t *testing.T) {
	want := testworkflowsv1.TestWorkflowTemplate{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TestWorkflowTemplate",
			APIVersion: "testworkflows.testkube.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dummy",
			Namespace: "dummy-namespace",
		},
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
			TestWorkflowSpecBase: workflowSpecBase,
			Setup:                []testworkflowsv1.IndependentStep{independentStep},
			Steps:                []testworkflowsv1.IndependentStep{independentStep, independentStep},
			After:                []testworkflowsv1.IndependentStep{independentStep, independentStep, independentStep, independentStep},
		},
	}
	got := MapTestWorkflowTemplateAPIToKube(MapTestWorkflowTemplateKubeToAPI(*want.DeepCopy()))
	assert.Equal(t, want, got)
}

func TestMapEmptyTestWorkflowTemplateBackAndForth(t *testing.T) {
	want := testworkflowsv1.TestWorkflowTemplate{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TestWorkflowTemplate",
			APIVersion: "testworkflows.testkube.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dummy",
			Namespace: "dummy-namespace",
		},
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{},
	}
	got := MapTestWorkflowTemplateAPIToKube(MapTestWorkflowTemplateKubeToAPI(*want.DeepCopy()))
	assert.Equal(t, want, got)
}
