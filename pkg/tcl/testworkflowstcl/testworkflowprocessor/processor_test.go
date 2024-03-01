// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowprocessor

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/imageinspector"
)

type dummyInspector struct{}

func (*dummyInspector) Inspect(ctx context.Context, registry, image string, pullPolicy corev1.PullPolicy, pullSecretNames []string) (*imageinspector.Info, error) {
	return &imageinspector.Info{}, nil
}

func TestProcessEmpty(t *testing.T) {
	p := processor{}
	wf := &testworkflowsv1.TestWorkflow{}

	_, err := p.Process(wf)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no steps to run")
}

func TestProcessBasic(t *testing.T) {
	p := &processor{}
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{StepBase: testworkflowsv1.StepBase{Shell: "shell-test"}},
			},
		},
	}

	res, err := p.Process(wf)
	job, err2 := res.Job(&dummyInspector{})

	sig := res.Signature()
	sigSerialized, _ := json.Marshal(res.RootStage().Signature().Children())

	assert.NoError(t, err)
	assert.NoError(t, err2)
	assert.Equal(t, batchv1.Job{
		TypeMeta: metav1.TypeMeta{Kind: "Job", APIVersion: "batch/v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "{{execution.id}}",
			Labels: map[string]string(nil),
			Annotations: map[string]string{
				"testworkflows.testkube.io/signature": string(sigSerialized),
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: common.Ptr(int32(0)),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "{{execution.id}}-pod",
					Labels:      map[string]string(nil),
					Annotations: map[string]string(nil),
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Volumes:       res.Resources().Volumes(),
					InitContainers: []corev1.Container{
						{
							Name:            "copy-init",
							Image:           defaultInitImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command:         []string{"/bin/sh", "-c"},
							Args:            []string{fmt.Sprintf("cp /init %s && touch %s && chmod 777 %s", defaultInitPath, defaultStatePath, defaultStatePath)},
							VolumeMounts:    res.ContainerDefaults().VolumeMounts(),
						},
					},
					Containers: []corev1.Container{
						{
							Name:            sig[0].Ref(),
							ImagePullPolicy: "",
							Image:           defaultImage,
							Command: []string{
								"/.tktw/init",
								sig[0].Ref(),
								"-c", fmt.Sprintf("%s=passed", sig[0].Ref()),
								"-r", fmt.Sprintf("=%s", sig[0].Ref()),
								"--",
							},
							Args:            []string{defaultShell, "-c", "shell-test"},
							WorkingDir:      "",
							EnvFrom:         []corev1.EnvFromSource(nil),
							Env:             []corev1.EnvVar{{Name: "CI", Value: "1"}},
							Resources:       corev1.ResourceRequirements{},
							VolumeMounts:    res.ContainerDefaults().VolumeMounts(),
							SecurityContext: (*corev1.SecurityContext)(nil),
						},
					},
				},
			},
		},
	}, job)
}

func TestProcessBasicEnvReference(t *testing.T) {
	p := &processor{}
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{StepBase: testworkflowsv1.StepBase{
					Container: &testworkflowsv1.ContainerConfig{
						Env: []corev1.EnvVar{
							{Name: "ZERO", Value: "foo"},
							{Name: "UNDETERMINED", Value: "{{call(abc)}}xxx"},
							{Name: "INPUT", Value: "{{env.ZERO}}bar"},
							{Name: "NEXT", Value: "foo{{env.UNDETERMINED}}{{env.LAST}}"},
							{Name: "LAST", Value: "foo{{env.INPUT}}bar"},
						},
					},
					Shell: "shell-test",
				}},
			},
		},
	}

	res, err := p.Process(wf)
	job, err2 := res.Job(&dummyInspector{})
	sig := res.Signature()
	sigSerialized, _ := json.Marshal(res.RootStage().Signature().Children())

	assert.NoError(t, err)
	assert.NoError(t, err2)
	assert.Equal(t, batchv1.Job{
		TypeMeta: metav1.TypeMeta{Kind: "Job", APIVersion: "batch/v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "{{execution.id}}",
			Labels: map[string]string(nil),
			Annotations: map[string]string{
				"testworkflows.testkube.io/signature": string(sigSerialized),
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: common.Ptr(int32(0)),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "{{execution.id}}-pod",
					Labels:      map[string]string(nil),
					Annotations: map[string]string(nil),
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Volumes:       res.Resources().Volumes(),
					InitContainers: []corev1.Container{
						{
							Name:            "copy-init",
							Image:           defaultInitImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command:         []string{"/bin/sh", "-c"},
							Args:            []string{fmt.Sprintf("cp /init %s && touch %s && chmod 777 %s", defaultInitPath, defaultStatePath, defaultStatePath)},
							VolumeMounts:    res.ContainerDefaults().VolumeMounts(),
						},
					},
					Containers: []corev1.Container{
						{
							Name:            sig[0].Ref(),
							ImagePullPolicy: "",
							Image:           defaultImage,
							Command: []string{
								"/.tktw/init",
								sig[0].Ref(),
								"-e", "UNDETERMINED,NEXT",
								"-c", fmt.Sprintf("%s=passed", sig[0].Ref()),
								"-r", fmt.Sprintf("=%s", sig[0].Ref()),
								"--",
							},
							Args:       []string{defaultShell, "-c", "shell-test"},
							WorkingDir: "",
							EnvFrom:    []corev1.EnvFromSource(nil),
							Env: []corev1.EnvVar{
								{Name: "CI", Value: "1"},
								{Name: "ZERO", Value: "foo"},
								{Name: "UNDETERMINED", Value: "{{call(abc)}}xxx"},
								{Name: "INPUT", Value: "foobar"},
								{Name: "NEXT", Value: "foo{{env.UNDETERMINED}}foofoobarbar"},
								{Name: "LAST", Value: "foofoobarbar"},
							},
							Resources:       corev1.ResourceRequirements{},
							VolumeMounts:    res.ContainerDefaults().VolumeMounts(),
							SecurityContext: (*corev1.SecurityContext)(nil),
						},
					},
				},
			},
		},
	}, job)
}

func TestProcessMultipleSteps(t *testing.T) {
	p := &processor{}
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{StepBase: testworkflowsv1.StepBase{Shell: "shell-test"}},
				{StepBase: testworkflowsv1.StepBase{Shell: "shell-test-2"}},
			},
		},
	}

	res, err := p.Process(wf)
	job, err2 := res.Job(&dummyInspector{})
	sig := res.Signature()
	sigSerialized, _ := json.Marshal(res.RootStage().Signature().Children())

	assert.NoError(t, err)
	assert.NoError(t, err2)
	assert.Equal(t, batchv1.Job{
		TypeMeta: metav1.TypeMeta{Kind: "Job", APIVersion: "batch/v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "{{execution.id}}",
			Labels: map[string]string(nil),
			Annotations: map[string]string{
				"testworkflows.testkube.io/signature": string(sigSerialized),
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: common.Ptr(int32(0)),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "{{execution.id}}-pod",
					Labels:      map[string]string(nil),
					Annotations: map[string]string(nil),
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Volumes:       res.Resources().Volumes(),
					InitContainers: []corev1.Container{
						{
							Name:            "copy-init",
							Image:           defaultInitImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command:         []string{"/bin/sh", "-c"},
							Args:            []string{fmt.Sprintf("cp /init %s && touch %s && chmod 777 %s", defaultInitPath, defaultStatePath, defaultStatePath)},
							VolumeMounts:    res.ContainerDefaults().VolumeMounts(),
						},
						{
							Name:            sig[0].Ref(),
							ImagePullPolicy: "",
							Image:           defaultImage,
							Command: []string{
								"/.tktw/init",
								sig[0].Ref(),
								"-c", fmt.Sprintf("%s=passed", sig[0].Ref()),
								"-r", fmt.Sprintf("=%s&&%s", sig[0].Ref(), sig[1].Ref()),
								"--",
							},
							Args:            []string{defaultShell, "-c", "shell-test"},
							WorkingDir:      "",
							EnvFrom:         []corev1.EnvFromSource(nil),
							Env:             []corev1.EnvVar{{Name: "CI", Value: "1"}},
							Resources:       corev1.ResourceRequirements{},
							VolumeMounts:    res.ContainerDefaults().VolumeMounts(),
							SecurityContext: (*corev1.SecurityContext)(nil),
						},
					},
					Containers: []corev1.Container{
						{
							Name:            sig[1].Ref(),
							ImagePullPolicy: "",
							Image:           defaultImage,
							Command: []string{
								"/.tktw/init",
								sig[1].Ref(),
								"-c", fmt.Sprintf("%s=passed", sig[1].Ref()),
								"-r", fmt.Sprintf("=%s&&%s", sig[0].Ref(), sig[1].Ref()),
								"--",
							},
							Args:            []string{defaultShell, "-c", "shell-test-2"},
							WorkingDir:      "",
							EnvFrom:         []corev1.EnvFromSource(nil),
							Env:             []corev1.EnvVar{{Name: "CI", Value: "1"}},
							Resources:       corev1.ResourceRequirements{},
							VolumeMounts:    res.ContainerDefaults().VolumeMounts(),
							SecurityContext: (*corev1.SecurityContext)(nil),
						},
					},
				},
			},
		},
	}, job)
}

func TestProcessNestedSteps(t *testing.T) {
	p := &processor{}
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{StepBase: testworkflowsv1.StepBase{Name: "A", Shell: "shell-test"}},
				{
					StepBase: testworkflowsv1.StepBase{Name: "B"},
					Steps: []testworkflowsv1.Step{
						{StepBase: testworkflowsv1.StepBase{Name: "C", Shell: "shell-test-2"}},
						{StepBase: testworkflowsv1.StepBase{Name: "D", Shell: "shell-test-3"}},
					},
				},
				{StepBase: testworkflowsv1.StepBase{Name: "E", Shell: "shell-test-4"}},
			},
		},
	}

	res, err := p.Process(wf)
	job, err2 := res.Job(&dummyInspector{})
	sig := res.Signature()

	assert.NoError(t, err)
	assert.NoError(t, err2)
	assert.Equal(t, corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyNever,
		Volumes:       res.Resources().Volumes(),
		InitContainers: []corev1.Container{
			{
				Name:            "copy-init",
				Image:           defaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/bin/sh", "-c"},
				Args:            []string{fmt.Sprintf("cp /init %s && touch %s && chmod 777 %s", defaultInitPath, defaultStatePath, defaultStatePath)},
				VolumeMounts:    res.ContainerDefaults().VolumeMounts(),
			},
			{
				Name:            sig[0].Ref(),
				ImagePullPolicy: "",
				Image:           defaultImage,
				Command: []string{
					"/.tktw/init",
					sig[0].Ref(),
					"-c", fmt.Sprintf("%s=passed", sig[0].Ref()),
					"-r", fmt.Sprintf("=%s&&%s&&%s", sig[0].Ref(), sig[1].Ref(), sig[2].Ref()),
					"--",
				},
				Args:            []string{defaultShell, "-c", "shell-test"},
				WorkingDir:      "",
				EnvFrom:         []corev1.EnvFromSource(nil),
				Env:             []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:       corev1.ResourceRequirements{},
				VolumeMounts:    res.ContainerDefaults().VolumeMounts(),
				SecurityContext: (*corev1.SecurityContext)(nil),
			},
			{
				Name:            sig[1].Children()[0].Ref(),
				ImagePullPolicy: "",
				Image:           defaultImage,
				Command: []string{
					"/.tktw/init",
					sig[1].Children()[0].Ref(),
					"-i", fmt.Sprintf("%s", sig[1].Ref()),
					"-c", fmt.Sprintf("%s,%s,%s=passed", sig[1].Ref(), sig[1].Children()[0].Ref(), sig[1].Children()[1].Ref()),
					"-r", fmt.Sprintf("=%s&&%s&&%s", sig[0].Ref(), sig[1].Ref(), sig[2].Ref()),
					"-r", fmt.Sprintf("%s=%s&&%s", sig[1].Ref(), sig[1].Children()[0].Ref(), sig[1].Children()[1].Ref()),
					"--",
				},
				Args:            []string{defaultShell, "-c", "shell-test-2"},
				WorkingDir:      "",
				EnvFrom:         []corev1.EnvFromSource(nil),
				Env:             []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:       corev1.ResourceRequirements{},
				VolumeMounts:    res.ContainerDefaults().VolumeMounts(),
				SecurityContext: (*corev1.SecurityContext)(nil),
			},
			{
				Name:            sig[1].Children()[1].Ref(),
				ImagePullPolicy: "",
				Image:           defaultImage,
				Command: []string{
					"/.tktw/init",
					sig[1].Children()[1].Ref(),
					"-i", fmt.Sprintf("%s", sig[1].Ref()),
					"-c", fmt.Sprintf("%s=passed", sig[1].Children()[1].Ref()),
					"-r", fmt.Sprintf("=%s&&%s&&%s", sig[0].Ref(), sig[1].Ref(), sig[2].Ref()),
					"-r", fmt.Sprintf("%s=%s&&%s", sig[1].Ref(), sig[1].Children()[0].Ref(), sig[1].Children()[1].Ref()),
					"--",
				},
				Args:            []string{defaultShell, "-c", "shell-test-3"},
				WorkingDir:      "",
				EnvFrom:         []corev1.EnvFromSource(nil),
				Env:             []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:       corev1.ResourceRequirements{},
				VolumeMounts:    res.ContainerDefaults().VolumeMounts(),
				SecurityContext: (*corev1.SecurityContext)(nil),
			},
		},
		Containers: []corev1.Container{
			{
				Name:            sig[2].Ref(),
				ImagePullPolicy: "",
				Image:           defaultImage,
				Command: []string{
					"/.tktw/init",
					sig[2].Ref(),
					"-c", fmt.Sprintf("%s=passed", sig[2].Ref()),
					"-r", fmt.Sprintf("=%s&&%s&&%s", sig[0].Ref(), sig[1].Ref(), sig[2].Ref()),
					"--",
				},
				Args:            []string{defaultShell, "-c", "shell-test-4"},
				WorkingDir:      "",
				EnvFrom:         []corev1.EnvFromSource(nil),
				Env:             []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:       corev1.ResourceRequirements{},
				VolumeMounts:    res.ContainerDefaults().VolumeMounts(),
				SecurityContext: (*corev1.SecurityContext)(nil),
			},
		},
	}, job.Spec.Template.Spec)
}

func TestProcessOptionalSteps(t *testing.T) {
	p := &processor{}
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{StepBase: testworkflowsv1.StepBase{Name: "A", Shell: "shell-test"}},
				{
					StepBase: testworkflowsv1.StepBase{Name: "B", Optional: true},
					Steps: []testworkflowsv1.Step{
						{StepBase: testworkflowsv1.StepBase{Name: "C", Shell: "shell-test-2"}},
						{StepBase: testworkflowsv1.StepBase{Name: "D", Shell: "shell-test-3"}},
					},
				},
				{StepBase: testworkflowsv1.StepBase{Name: "E", Shell: "shell-test-4"}},
			},
		},
	}

	res, err := p.Process(wf)
	job, err2 := res.Job(&dummyInspector{})
	sig := res.Signature()

	assert.NoError(t, err)
	assert.NoError(t, err2)
	assert.Equal(t, corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyNever,
		Volumes:       res.Resources().Volumes(),
		InitContainers: []corev1.Container{
			{
				Name:            "copy-init",
				Image:           defaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/bin/sh", "-c"},
				Args:            []string{fmt.Sprintf("cp /init %s && touch %s && chmod 777 %s", defaultInitPath, defaultStatePath, defaultStatePath)},
				VolumeMounts:    res.ContainerDefaults().VolumeMounts(),
			},
			{
				Name:            sig[0].Ref(),
				ImagePullPolicy: "",
				Image:           defaultImage,
				Command: []string{
					"/.tktw/init",
					sig[0].Ref(),
					"-c", fmt.Sprintf("%s=passed", sig[0].Ref()),
					"-r", fmt.Sprintf("=%s&&%s", sig[0].Ref(), sig[2].Ref()),
					"--",
				},
				Args:            []string{defaultShell, "-c", "shell-test"},
				WorkingDir:      "",
				EnvFrom:         []corev1.EnvFromSource(nil),
				Env:             []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:       corev1.ResourceRequirements{},
				VolumeMounts:    res.ContainerDefaults().VolumeMounts(),
				SecurityContext: (*corev1.SecurityContext)(nil),
			},
			{
				Name:            sig[1].Children()[0].Ref(),
				ImagePullPolicy: "",
				Image:           defaultImage,
				Command: []string{
					"/.tktw/init",
					sig[1].Children()[0].Ref(),
					"-i", fmt.Sprintf("%s", sig[1].Ref()),
					"-c", fmt.Sprintf("%s,%s,%s=passed", sig[1].Ref(), sig[1].Children()[0].Ref(), sig[1].Children()[1].Ref()),
					"-r", fmt.Sprintf("%s=%s&&%s", sig[1].Ref(), sig[1].Children()[0].Ref(), sig[1].Children()[1].Ref()),
					"--",
				},
				Args:            []string{defaultShell, "-c", "shell-test-2"},
				WorkingDir:      "",
				EnvFrom:         []corev1.EnvFromSource(nil),
				Env:             []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:       corev1.ResourceRequirements{},
				VolumeMounts:    res.ContainerDefaults().VolumeMounts(),
				SecurityContext: (*corev1.SecurityContext)(nil),
			},
			{
				Name:            sig[1].Children()[1].Ref(),
				ImagePullPolicy: "",
				Image:           defaultImage,
				Command: []string{
					"/.tktw/init",
					sig[1].Children()[1].Ref(),
					"-i", fmt.Sprintf("%s", sig[1].Ref()),
					"-c", fmt.Sprintf("%s=passed", sig[1].Children()[1].Ref()),
					"-r", fmt.Sprintf("%s=%s&&%s", sig[1].Ref(), sig[1].Children()[0].Ref(), sig[1].Children()[1].Ref()),
					"--",
				},
				Args:            []string{defaultShell, "-c", "shell-test-3"},
				WorkingDir:      "",
				EnvFrom:         []corev1.EnvFromSource(nil),
				Env:             []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:       corev1.ResourceRequirements{},
				VolumeMounts:    res.ContainerDefaults().VolumeMounts(),
				SecurityContext: (*corev1.SecurityContext)(nil),
			},
		},
		Containers: []corev1.Container{
			{
				Name:            sig[2].Ref(),
				ImagePullPolicy: "",
				Image:           defaultImage,
				Command: []string{
					"/.tktw/init",
					sig[2].Ref(),
					"-c", fmt.Sprintf("%s=passed", sig[2].Ref()),
					"-r", fmt.Sprintf("=%s&&%s", sig[0].Ref(), sig[2].Ref()),
					"--",
				},
				Args:            []string{defaultShell, "-c", "shell-test-4"},
				WorkingDir:      "",
				EnvFrom:         []corev1.EnvFromSource(nil),
				Env:             []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:       corev1.ResourceRequirements{},
				VolumeMounts:    res.ContainerDefaults().VolumeMounts(),
				SecurityContext: (*corev1.SecurityContext)(nil),
			},
		},
	}, job.Spec.Template.Spec)
}

func TestProcessNegativeSteps(t *testing.T) {
	p := &processor{}
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{StepBase: testworkflowsv1.StepBase{Name: "A", Shell: "shell-test"}},
				{
					StepBase: testworkflowsv1.StepBase{Name: "B", Negative: true},
					Steps: []testworkflowsv1.Step{
						{StepBase: testworkflowsv1.StepBase{Name: "C", Shell: "shell-test-2"}},
						{StepBase: testworkflowsv1.StepBase{Name: "D", Shell: "shell-test-3"}},
					},
				},
				{StepBase: testworkflowsv1.StepBase{Name: "E", Shell: "shell-test-4"}},
			},
		},
	}

	res, err := p.Process(wf)
	job, err2 := res.Job(&dummyInspector{})
	sig := res.Signature()

	assert.NoError(t, err)
	assert.NoError(t, err2)
	assert.Equal(t, corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyNever,
		Volumes:       res.Resources().Volumes(),
		InitContainers: []corev1.Container{
			{
				Name:            "copy-init",
				Image:           defaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/bin/sh", "-c"},
				Args:            []string{fmt.Sprintf("cp /init %s && touch %s && chmod 777 %s", defaultInitPath, defaultStatePath, defaultStatePath)},
				VolumeMounts:    res.ContainerDefaults().VolumeMounts(),
			},
			{
				Name:            sig[0].Ref(),
				ImagePullPolicy: "",
				Image:           defaultImage,
				Command: []string{
					"/.tktw/init",
					sig[0].Ref(),
					"-c", fmt.Sprintf("%s=passed", sig[0].Ref()),
					"-r", fmt.Sprintf("=%s&&%s&&%s", sig[0].Ref(), sig[1].Ref(), sig[2].Ref()),
					"--",
				},
				Args:            []string{defaultShell, "-c", "shell-test"},
				WorkingDir:      "",
				EnvFrom:         []corev1.EnvFromSource(nil),
				Env:             []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:       corev1.ResourceRequirements{},
				VolumeMounts:    res.ContainerDefaults().VolumeMounts(),
				SecurityContext: (*corev1.SecurityContext)(nil),
			},
			{
				Name:            sig[1].Children()[0].Ref(),
				ImagePullPolicy: "",
				Image:           defaultImage,
				Command: []string{
					"/.tktw/init",
					sig[1].Children()[0].Ref(),
					"-i", fmt.Sprintf("%s.v", sig[1].Ref()),
					"-c", fmt.Sprintf("%s,%s,%s,%s.v=passed", sig[1].Ref(), sig[1].Children()[0].Ref(), sig[1].Children()[1].Ref(), sig[1].Ref()),
					"-r", fmt.Sprintf("=%s&&%s&&%s", sig[0].Ref(), sig[1].Ref(), sig[2].Ref()),
					"-r", fmt.Sprintf("%s=!%s.v", sig[1].Ref(), sig[1].Ref()),
					"-r", fmt.Sprintf("%s.v=%s&&%s", sig[1].Ref(), sig[1].Children()[0].Ref(), sig[1].Children()[1].Ref()),
					"--",
				},
				Args:            []string{defaultShell, "-c", "shell-test-2"},
				WorkingDir:      "",
				EnvFrom:         []corev1.EnvFromSource(nil),
				Env:             []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:       corev1.ResourceRequirements{},
				VolumeMounts:    res.ContainerDefaults().VolumeMounts(),
				SecurityContext: (*corev1.SecurityContext)(nil),
			},
			{
				Name:            sig[1].Children()[1].Ref(),
				ImagePullPolicy: "",
				Image:           defaultImage,
				Command: []string{
					"/.tktw/init",
					sig[1].Children()[1].Ref(),
					"-i", fmt.Sprintf("%s.v", sig[1].Ref()),
					"-c", fmt.Sprintf("%s=passed", sig[1].Children()[1].Ref()),
					"-r", fmt.Sprintf("=%s&&%s&&%s", sig[0].Ref(), sig[1].Ref(), sig[2].Ref()),
					"-r", fmt.Sprintf("%s=!%s.v", sig[1].Ref(), sig[1].Ref()),
					"-r", fmt.Sprintf("%s.v=%s&&%s", sig[1].Ref(), sig[1].Children()[0].Ref(), sig[1].Children()[1].Ref()),
					"--",
				},
				Args:            []string{defaultShell, "-c", "shell-test-3"},
				WorkingDir:      "",
				EnvFrom:         []corev1.EnvFromSource(nil),
				Env:             []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:       corev1.ResourceRequirements{},
				VolumeMounts:    res.ContainerDefaults().VolumeMounts(),
				SecurityContext: (*corev1.SecurityContext)(nil),
			},
		},
		Containers: []corev1.Container{
			{
				Name:            sig[2].Ref(),
				ImagePullPolicy: "",
				Image:           defaultImage,
				Command: []string{
					"/.tktw/init",
					sig[2].Ref(),
					"-c", fmt.Sprintf("%s=passed", sig[2].Ref()),
					"-r", fmt.Sprintf("=%s&&%s&&%s", sig[0].Ref(), sig[1].Ref(), sig[2].Ref()),
					"--",
				},
				Args:            []string{defaultShell, "-c", "shell-test-4"},
				WorkingDir:      "",
				EnvFrom:         []corev1.EnvFromSource(nil),
				Env:             []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:       corev1.ResourceRequirements{},
				VolumeMounts:    res.ContainerDefaults().VolumeMounts(),
				SecurityContext: (*corev1.SecurityContext)(nil),
			},
		},
	}, job.Spec.Template.Spec)
}

func TestProcessNegativeContainerStep(t *testing.T) {
	p := &processor{}
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{StepBase: testworkflowsv1.StepBase{Shell: "shell-test"}},
				{StepBase: testworkflowsv1.StepBase{Shell: "shell-test-2", Negative: true}},
			},
		},
	}

	res, err := p.Process(wf)
	job, err2 := res.Job(&dummyInspector{})
	sig := res.Signature()
	sigSerialized, _ := json.Marshal(res.RootStage().Signature().Children())

	assert.NoError(t, err)
	assert.NoError(t, err2)
	assert.Equal(t, batchv1.Job{
		TypeMeta: metav1.TypeMeta{Kind: "Job", APIVersion: "batch/v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "{{execution.id}}",
			Labels: map[string]string(nil),
			Annotations: map[string]string{
				"testworkflows.testkube.io/signature": string(sigSerialized),
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: common.Ptr(int32(0)),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "{{execution.id}}-pod",
					Labels:      map[string]string(nil),
					Annotations: map[string]string(nil),
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Volumes:       res.Resources().Volumes(),
					InitContainers: []corev1.Container{
						{
							Name:            "copy-init",
							Image:           defaultInitImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command:         []string{"/bin/sh", "-c"},
							Args:            []string{fmt.Sprintf("cp /init %s && touch %s && chmod 777 %s", defaultInitPath, defaultStatePath, defaultStatePath)},
							VolumeMounts:    res.ContainerDefaults().VolumeMounts(),
						},
						{
							Name:            sig[0].Ref(),
							ImagePullPolicy: "",
							Image:           defaultImage,
							Command: []string{
								"/.tktw/init",
								sig[0].Ref(),
								"-c", fmt.Sprintf("%s=passed", sig[0].Ref()),
								"-r", fmt.Sprintf("=%s&&%s", sig[0].Ref(), sig[1].Ref()),
								"--",
							},
							Args:            []string{defaultShell, "-c", "shell-test"},
							WorkingDir:      "",
							EnvFrom:         []corev1.EnvFromSource(nil),
							Env:             []corev1.EnvVar{{Name: "CI", Value: "1"}},
							Resources:       corev1.ResourceRequirements{},
							VolumeMounts:    res.ContainerDefaults().VolumeMounts(),
							SecurityContext: (*corev1.SecurityContext)(nil),
						},
					},
					Containers: []corev1.Container{
						{
							Name:            sig[1].Ref(),
							ImagePullPolicy: "",
							Image:           defaultImage,
							Command: []string{
								"/.tktw/init",
								sig[1].Ref(),
								"--negative", "true",
								"-c", fmt.Sprintf("%s=passed", sig[1].Ref()),
								"-r", fmt.Sprintf("=%s&&%s", sig[0].Ref(), sig[1].Ref()),
								"--",
							},
							Args:            []string{defaultShell, "-c", "shell-test-2"},
							WorkingDir:      "",
							EnvFrom:         []corev1.EnvFromSource(nil),
							Env:             []corev1.EnvVar{{Name: "CI", Value: "1"}},
							Resources:       corev1.ResourceRequirements{},
							VolumeMounts:    res.ContainerDefaults().VolumeMounts(),
							SecurityContext: (*corev1.SecurityContext)(nil),
						},
					},
				},
			},
		},
	}, job)
}

func TestProcessOptionalContainerStep(t *testing.T) {
	p := &processor{}
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{StepBase: testworkflowsv1.StepBase{Shell: "shell-test"}},
				{StepBase: testworkflowsv1.StepBase{Shell: "shell-test-2", Optional: true}},
			},
		},
	}

	res, err := p.Process(wf)
	job, err2 := res.Job(&dummyInspector{})
	sig := res.Signature()
	sigSerialized, _ := json.Marshal(res.RootStage().Signature().Children())

	assert.NoError(t, err)
	assert.NoError(t, err2)
	assert.Equal(t, batchv1.Job{
		TypeMeta: metav1.TypeMeta{Kind: "Job", APIVersion: "batch/v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "{{execution.id}}",
			Labels: map[string]string(nil),
			Annotations: map[string]string{
				"testworkflows.testkube.io/signature": string(sigSerialized),
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: common.Ptr(int32(0)),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "{{execution.id}}-pod",
					Labels:      map[string]string(nil),
					Annotations: map[string]string(nil),
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Volumes:       res.Resources().Volumes(),
					InitContainers: []corev1.Container{
						{
							Name:            "copy-init",
							Image:           defaultInitImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command:         []string{"/bin/sh", "-c"},
							Args:            []string{fmt.Sprintf("cp /init %s && touch %s && chmod 777 %s", defaultInitPath, defaultStatePath, defaultStatePath)},
							VolumeMounts:    res.ContainerDefaults().VolumeMounts(),
						},
						{
							Name:            sig[0].Ref(),
							ImagePullPolicy: "",
							Image:           defaultImage,
							Command: []string{
								"/.tktw/init",
								sig[0].Ref(),
								"-c", fmt.Sprintf("%s=passed", sig[0].Ref()),
								"-r", fmt.Sprintf("=%s", sig[0].Ref()),
								"--",
							},
							Args:            []string{defaultShell, "-c", "shell-test"},
							WorkingDir:      "",
							EnvFrom:         []corev1.EnvFromSource(nil),
							Env:             []corev1.EnvVar{{Name: "CI", Value: "1"}},
							Resources:       corev1.ResourceRequirements{},
							VolumeMounts:    res.ContainerDefaults().VolumeMounts(),
							SecurityContext: (*corev1.SecurityContext)(nil),
						},
					},
					Containers: []corev1.Container{
						{
							Name:            sig[1].Ref(),
							ImagePullPolicy: "",
							Image:           defaultImage,
							Command: []string{
								"/.tktw/init",
								sig[1].Ref(),
								"-c", fmt.Sprintf("%s=passed", sig[1].Ref()),
								"--",
							},
							Args:            []string{defaultShell, "-c", "shell-test-2"},
							WorkingDir:      "",
							EnvFrom:         []corev1.EnvFromSource(nil),
							Env:             []corev1.EnvVar{{Name: "CI", Value: "1"}},
							Resources:       corev1.ResourceRequirements{},
							VolumeMounts:    res.ContainerDefaults().VolumeMounts(),
							SecurityContext: (*corev1.SecurityContext)(nil),
						},
					},
				},
			},
		},
	}, job)
}
