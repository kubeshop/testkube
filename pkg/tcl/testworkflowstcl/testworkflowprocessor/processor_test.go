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
	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
)

type dummyInspector struct{}

func (*dummyInspector) Inspect(ctx context.Context, registry, image string, pullPolicy corev1.PullPolicy, pullSecretNames []string) (*imageinspector.Info, error) {
	return &imageinspector.Info{}, nil
}

var (
	ins         = &dummyInspector{}
	proc        = NewFullFeatured(ins)
	execMachine = expressionstcl.NewMachine().
			Register("execution.id", "dummy-id")
)

func TestProcessEmpty(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{}

	_, err := proc.Bundle(context.Background(), wf, execMachine)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "has nothing to run")
}

func TestProcessBasic(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{StepBase: testworkflowsv1.StepBase{Shell: "shell-test"}},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, execMachine)
	assert.NoError(t, err)

	sig := res.Signature
	sigSerialized, _ := json.Marshal(sig)

	volumes := res.Job.Spec.Template.Spec.Volumes
	volumeMounts := res.Job.Spec.Template.Spec.InitContainers[0].VolumeMounts

	want := batchv1.Job{
		TypeMeta: metav1.TypeMeta{Kind: "Job", APIVersion: "batch/v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "dummy-id",
			Labels: map[string]string{ExecutionIdLabelName: "dummy-id"},
			Annotations: map[string]string{
				SignatureAnnotationName: string(sigSerialized),
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: common.Ptr(int32(0)),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						ExecutionIdLabelName:        "dummy-id",
						ExecutionIdMainPodLabelName: "dummy-id",
					},
					Annotations: map[string]string(nil),
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Volumes:       volumes,
					InitContainers: []corev1.Container{
						{
							Name:            "tktw-init",
							Image:           defaultInitImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command:         []string{"/bin/sh", "-c"},
							Args:            []string{"cp /init /.tktw/init && touch /.tktw/state && chmod 777 /.tktw/state && (echo -n ',0' > /dev/termination-log && echo 'Done' && exit 0) || (echo -n 'failed,1' > /dev/termination-log && exit 1)"},
							VolumeMounts:    volumeMounts,
							SecurityContext: &corev1.SecurityContext{
								RunAsGroup: common.Ptr(defaultFsGroup),
							},
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
							Args:         []string{defaultShell, "-c", "shell-test"},
							WorkingDir:   "",
							EnvFrom:      []corev1.EnvFromSource(nil),
							Env:          []corev1.EnvVar{{Name: "CI", Value: "1"}},
							Resources:    corev1.ResourceRequirements{},
							VolumeMounts: volumeMounts,
							SecurityContext: &corev1.SecurityContext{
								RunAsGroup: common.Ptr(defaultFsGroup),
							},
						},
					},
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup: common.Ptr(defaultFsGroup),
					},
				},
			},
		},
	}

	assert.Equal(t, want, res.Job)

	assert.Equal(t, 2, len(volumeMounts))
	assert.Equal(t, 2, len(volumes))
	assert.Equal(t, defaultInternalPath, volumeMounts[0].MountPath)
	assert.Equal(t, defaultDataPath, volumeMounts[1].MountPath)
	assert.True(t, volumeMounts[0].Name == volumes[0].Name)
	assert.True(t, volumeMounts[1].Name == volumes[1].Name)
}

func TestProcessBasicEnvReference(t *testing.T) {
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

	res, err := proc.Bundle(context.Background(), wf, execMachine)
	sig := res.Signature

	volumes := res.Job.Spec.Template.Spec.Volumes
	volumeMounts := res.Job.Spec.Template.Spec.InitContainers[0].VolumeMounts

	want := corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyNever,
		Volumes:       volumes,
		InitContainers: []corev1.Container{
			{
				Name:            "tktw-init",
				Image:           defaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/bin/sh", "-c"},
				Args:            []string{"cp /init /.tktw/init && touch /.tktw/state && chmod 777 /.tktw/state && (echo -n ',0' > /dev/termination-log && echo 'Done' && exit 0) || (echo -n 'failed,1' > /dev/termination-log && exit 1)"},
				VolumeMounts:    volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(defaultFsGroup),
				},
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
				Resources:    corev1.ResourceRequirements{},
				VolumeMounts: volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(defaultFsGroup),
				},
			},
		},
		SecurityContext: &corev1.PodSecurityContext{
			FSGroup: common.Ptr(defaultFsGroup),
		},
	}

	assert.NoError(t, err)
	assert.Equal(t, want, res.Job.Spec.Template.Spec)
}

func TestProcessMultipleSteps(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{StepBase: testworkflowsv1.StepBase{Shell: "shell-test"}},
				{StepBase: testworkflowsv1.StepBase{Shell: "shell-test-2"}},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, execMachine)
	sig := res.Signature

	volumes := res.Job.Spec.Template.Spec.Volumes
	volumeMounts := res.Job.Spec.Template.Spec.InitContainers[0].VolumeMounts

	want := corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyNever,
		Volumes:       volumes,
		InitContainers: []corev1.Container{
			{
				Name:            "tktw-init",
				Image:           defaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/bin/sh", "-c"},
				Args:            []string{"cp /init /.tktw/init && touch /.tktw/state && chmod 777 /.tktw/state && (echo -n ',0' > /dev/termination-log && echo 'Done' && exit 0) || (echo -n 'failed,1' > /dev/termination-log && exit 1)"},
				VolumeMounts:    volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(defaultFsGroup),
				},
			},
			{
				Name:            sig[0].Ref(),
				ImagePullPolicy: "",
				Image:           defaultImage,
				Command: []string{
					"/.tktw/init",
					sig[0].Ref(),
					"-c", fmt.Sprintf("%s,%s=passed", sig[0].Ref(), sig[1].Ref()),
					"-r", fmt.Sprintf("=%s&&%s", sig[0].Ref(), sig[1].Ref()),
					"--",
				},
				Args:         []string{defaultShell, "-c", "shell-test"},
				WorkingDir:   "",
				EnvFrom:      []corev1.EnvFromSource(nil),
				Env:          []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:    corev1.ResourceRequirements{},
				VolumeMounts: volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(defaultFsGroup),
				},
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
				Args:         []string{defaultShell, "-c", "shell-test-2"},
				WorkingDir:   "",
				EnvFrom:      []corev1.EnvFromSource(nil),
				Env:          []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:    corev1.ResourceRequirements{},
				VolumeMounts: volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(defaultFsGroup),
				},
			},
		},
		SecurityContext: &corev1.PodSecurityContext{
			FSGroup: common.Ptr(defaultFsGroup),
		},
	}

	assert.NoError(t, err)
	assert.Equal(t, want, res.Job.Spec.Template.Spec)
}

func TestProcessNestedSteps(t *testing.T) {
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

	res, err := proc.Bundle(context.Background(), wf, execMachine)
	sig := res.Signature

	volumes := res.Job.Spec.Template.Spec.Volumes
	volumeMounts := res.Job.Spec.Template.Spec.InitContainers[0].VolumeMounts

	want := corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyNever,
		Volumes:       volumes,
		InitContainers: []corev1.Container{
			{
				Name:            "tktw-init",
				Image:           defaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/bin/sh", "-c"},
				Args:            []string{"cp /init /.tktw/init && touch /.tktw/state && chmod 777 /.tktw/state && (echo -n ',0' > /dev/termination-log && echo 'Done' && exit 0) || (echo -n 'failed,1' > /dev/termination-log && exit 1)"},
				VolumeMounts:    volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(defaultFsGroup),
				},
			},
			{
				Name:            sig[0].Ref(),
				ImagePullPolicy: "",
				Image:           defaultImage,
				Command: []string{
					"/.tktw/init",
					sig[0].Ref(),
					"-c", fmt.Sprintf("%s,%s,%s,%s=passed", sig[0].Ref(), sig[1].Children()[0].Ref(), sig[1].Children()[1].Ref(), sig[2].Ref()),
					"-r", fmt.Sprintf("=%s&&%s&&%s", sig[0].Ref(), sig[1].Ref(), sig[2].Ref()),
					"--",
				},
				Args:         []string{defaultShell, "-c", "shell-test"},
				WorkingDir:   "",
				EnvFrom:      []corev1.EnvFromSource(nil),
				Env:          []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:    corev1.ResourceRequirements{},
				VolumeMounts: volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(defaultFsGroup),
				},
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
				Args:         []string{defaultShell, "-c", "shell-test-2"},
				WorkingDir:   "",
				EnvFrom:      []corev1.EnvFromSource(nil),
				Env:          []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:    corev1.ResourceRequirements{},
				VolumeMounts: volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(defaultFsGroup),
				},
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
				Args:         []string{defaultShell, "-c", "shell-test-3"},
				WorkingDir:   "",
				EnvFrom:      []corev1.EnvFromSource(nil),
				Env:          []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:    corev1.ResourceRequirements{},
				VolumeMounts: volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(defaultFsGroup),
				},
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
				Args:         []string{defaultShell, "-c", "shell-test-4"},
				WorkingDir:   "",
				EnvFrom:      []corev1.EnvFromSource(nil),
				Env:          []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:    corev1.ResourceRequirements{},
				VolumeMounts: volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(defaultFsGroup),
				},
			},
		},
		SecurityContext: &corev1.PodSecurityContext{
			FSGroup: common.Ptr(defaultFsGroup),
		},
	}

	assert.NoError(t, err)
	assert.Equal(t, want, res.Job.Spec.Template.Spec)
}

func TestProcessOptionalSteps(t *testing.T) {
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

	res, err := proc.Bundle(context.Background(), wf, execMachine)
	sig := res.Signature

	volumes := res.Job.Spec.Template.Spec.Volumes
	volumeMounts := res.Job.Spec.Template.Spec.InitContainers[0].VolumeMounts

	want := corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyNever,
		Volumes:       volumes,
		InitContainers: []corev1.Container{
			{
				Name:            "tktw-init",
				Image:           defaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/bin/sh", "-c"},
				Args:            []string{"cp /init /.tktw/init && touch /.tktw/state && chmod 777 /.tktw/state && (echo -n ',0' > /dev/termination-log && echo 'Done' && exit 0) || (echo -n 'failed,1' > /dev/termination-log && exit 1)"},
				VolumeMounts:    volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(defaultFsGroup),
				},
			},
			{
				Name:            sig[0].Ref(),
				ImagePullPolicy: "",
				Image:           defaultImage,
				Command: []string{
					"/.tktw/init",
					sig[0].Ref(),
					"-c", fmt.Sprintf("%s,%s,%s,%s=passed", sig[0].Ref(), sig[1].Children()[0].Ref(), sig[1].Children()[1].Ref(), sig[2].Ref()),
					"-r", fmt.Sprintf("=%s&&%s", sig[0].Ref(), sig[2].Ref()),
					"--",
				},
				Args:         []string{defaultShell, "-c", "shell-test"},
				WorkingDir:   "",
				EnvFrom:      []corev1.EnvFromSource(nil),
				Env:          []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:    corev1.ResourceRequirements{},
				VolumeMounts: volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(defaultFsGroup),
				},
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
				Args:         []string{defaultShell, "-c", "shell-test-2"},
				WorkingDir:   "",
				EnvFrom:      []corev1.EnvFromSource(nil),
				Env:          []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:    corev1.ResourceRequirements{},
				VolumeMounts: volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(defaultFsGroup),
				},
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
				Args:         []string{defaultShell, "-c", "shell-test-3"},
				WorkingDir:   "",
				EnvFrom:      []corev1.EnvFromSource(nil),
				Env:          []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:    corev1.ResourceRequirements{},
				VolumeMounts: volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(defaultFsGroup),
				},
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
				Args:         []string{defaultShell, "-c", "shell-test-4"},
				WorkingDir:   "",
				EnvFrom:      []corev1.EnvFromSource(nil),
				Env:          []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:    corev1.ResourceRequirements{},
				VolumeMounts: volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(defaultFsGroup),
				},
			},
		},
		SecurityContext: &corev1.PodSecurityContext{
			FSGroup: common.Ptr(defaultFsGroup),
		},
	}

	assert.NoError(t, err)
	assert.Equal(t, want, res.Job.Spec.Template.Spec)
}

func TestProcessNegativeSteps(t *testing.T) {
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

	res, err := proc.Bundle(context.Background(), wf, execMachine)
	sig := res.Signature

	volumes := res.Job.Spec.Template.Spec.Volumes
	volumeMounts := res.Job.Spec.Template.Spec.InitContainers[0].VolumeMounts

	want := corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyNever,
		Volumes:       volumes,
		InitContainers: []corev1.Container{
			{
				Name:            "tktw-init",
				Image:           defaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/bin/sh", "-c"},
				Args:            []string{"cp /init /.tktw/init && touch /.tktw/state && chmod 777 /.tktw/state && (echo -n ',0' > /dev/termination-log && echo 'Done' && exit 0) || (echo -n 'failed,1' > /dev/termination-log && exit 1)"},
				VolumeMounts:    volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(defaultFsGroup),
				},
			},
			{
				Name:            sig[0].Ref(),
				ImagePullPolicy: "",
				Image:           defaultImage,
				Command: []string{
					"/.tktw/init",
					sig[0].Ref(),
					"-c", fmt.Sprintf("%s,%s,%s,%s=passed", sig[0].Ref(), sig[1].Children()[0].Ref(), sig[1].Children()[1].Ref(), sig[2].Ref()),
					"-r", fmt.Sprintf("=%s&&%s&&%s", sig[0].Ref(), sig[1].Ref(), sig[2].Ref()),
					"--",
				},
				Args:         []string{defaultShell, "-c", "shell-test"},
				WorkingDir:   "",
				EnvFrom:      []corev1.EnvFromSource(nil),
				Env:          []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:    corev1.ResourceRequirements{},
				VolumeMounts: volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(defaultFsGroup),
				},
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
				Args:         []string{defaultShell, "-c", "shell-test-2"},
				WorkingDir:   "",
				EnvFrom:      []corev1.EnvFromSource(nil),
				Env:          []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:    corev1.ResourceRequirements{},
				VolumeMounts: volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(defaultFsGroup),
				},
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
				Args:         []string{defaultShell, "-c", "shell-test-3"},
				WorkingDir:   "",
				EnvFrom:      []corev1.EnvFromSource(nil),
				Env:          []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:    corev1.ResourceRequirements{},
				VolumeMounts: volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(defaultFsGroup),
				},
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
				Args:         []string{defaultShell, "-c", "shell-test-4"},
				WorkingDir:   "",
				EnvFrom:      []corev1.EnvFromSource(nil),
				Env:          []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:    corev1.ResourceRequirements{},
				VolumeMounts: volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(defaultFsGroup),
				},
			},
		},
		SecurityContext: &corev1.PodSecurityContext{
			FSGroup: common.Ptr(defaultFsGroup),
		},
	}

	assert.NoError(t, err)
	assert.Equal(t, want, res.Job.Spec.Template.Spec)
}

func TestProcessNegativeContainerStep(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{StepBase: testworkflowsv1.StepBase{Shell: "shell-test"}},
				{StepBase: testworkflowsv1.StepBase{Shell: "shell-test-2", Negative: true}},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, execMachine)
	sig := res.Signature

	volumes := res.Job.Spec.Template.Spec.Volumes
	volumeMounts := res.Job.Spec.Template.Spec.InitContainers[0].VolumeMounts

	want := corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyNever,
		Volumes:       volumes,
		InitContainers: []corev1.Container{
			{
				Name:            "tktw-init",
				Image:           defaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/bin/sh", "-c"},
				Args:            []string{"cp /init /.tktw/init && touch /.tktw/state && chmod 777 /.tktw/state && (echo -n ',0' > /dev/termination-log && echo 'Done' && exit 0) || (echo -n 'failed,1' > /dev/termination-log && exit 1)"},
				VolumeMounts:    volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(defaultFsGroup),
				},
			},
			{
				Name:            sig[0].Ref(),
				ImagePullPolicy: "",
				Image:           defaultImage,
				Command: []string{
					"/.tktw/init",
					sig[0].Ref(),
					"-c", fmt.Sprintf("%s,%s=passed", sig[0].Ref(), sig[1].Ref()),
					"-r", fmt.Sprintf("=%s&&%s", sig[0].Ref(), sig[1].Ref()),
					"--",
				},
				Args:         []string{defaultShell, "-c", "shell-test"},
				WorkingDir:   "",
				EnvFrom:      []corev1.EnvFromSource(nil),
				Env:          []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:    corev1.ResourceRequirements{},
				VolumeMounts: volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(defaultFsGroup),
				},
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
					"-n", "true",
					"-c", fmt.Sprintf("%s=passed", sig[1].Ref()),
					"-r", fmt.Sprintf("=%s&&%s", sig[0].Ref(), sig[1].Ref()),
					"--",
				},
				Args:         []string{defaultShell, "-c", "shell-test-2"},
				WorkingDir:   "",
				EnvFrom:      []corev1.EnvFromSource(nil),
				Env:          []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:    corev1.ResourceRequirements{},
				VolumeMounts: volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(defaultFsGroup),
				},
			},
		},

		SecurityContext: &corev1.PodSecurityContext{
			FSGroup: common.Ptr(defaultFsGroup),
		},
	}

	assert.NoError(t, err)
	assert.Equal(t, want, res.Job.Spec.Template.Spec)
}

func TestProcessOptionalContainerStep(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{StepBase: testworkflowsv1.StepBase{Shell: "shell-test"}},
				{StepBase: testworkflowsv1.StepBase{Shell: "shell-test-2", Optional: true}},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, execMachine)
	sig := res.Signature

	volumes := res.Job.Spec.Template.Spec.Volumes
	volumeMounts := res.Job.Spec.Template.Spec.InitContainers[0].VolumeMounts

	want := corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyNever,
		Volumes:       volumes,
		InitContainers: []corev1.Container{
			{
				Name:            "tktw-init",
				Image:           defaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/bin/sh", "-c"},
				Args:            []string{"cp /init /.tktw/init && touch /.tktw/state && chmod 777 /.tktw/state && (echo -n ',0' > /dev/termination-log && echo 'Done' && exit 0) || (echo -n 'failed,1' > /dev/termination-log && exit 1)"},
				VolumeMounts:    volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(defaultFsGroup),
				},
			},
			{
				Name:            sig[0].Ref(),
				ImagePullPolicy: "",
				Image:           defaultImage,
				Command: []string{
					"/.tktw/init",
					sig[0].Ref(),
					"-c", fmt.Sprintf("%s,%s=passed", sig[0].Ref(), sig[1].Ref()),
					"-r", fmt.Sprintf("=%s", sig[0].Ref()),
					"--",
				},
				Args:         []string{defaultShell, "-c", "shell-test"},
				WorkingDir:   "",
				EnvFrom:      []corev1.EnvFromSource(nil),
				Env:          []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:    corev1.ResourceRequirements{},
				VolumeMounts: volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(defaultFsGroup),
				},
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
				Args:         []string{defaultShell, "-c", "shell-test-2"},
				WorkingDir:   "",
				EnvFrom:      []corev1.EnvFromSource(nil),
				Env:          []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:    corev1.ResourceRequirements{},
				VolumeMounts: volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(defaultFsGroup),
				},
			},
		},
		SecurityContext: &corev1.PodSecurityContext{
			FSGroup: common.Ptr(defaultFsGroup),
		},
	}

	assert.NoError(t, err)
	assert.Equal(t, want, res.Job.Spec.Template.Spec)
}

func TestProcessLocalContent(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{StepBase: testworkflowsv1.StepBase{
					Shell: "shell-test",
					Content: &testworkflowsv1.Content{
						Files: []testworkflowsv1.ContentFile{{
							Path:    "/some/path",
							Content: `some-{{"{{"}}content`,
						}},
					},
				}},
				{StepBase: testworkflowsv1.StepBase{Shell: "shell-test-2"}},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, execMachine)
	assert.NoError(t, err)

	sig := res.Signature

	volumes := res.Job.Spec.Template.Spec.Volumes
	volumeMounts := res.Job.Spec.Template.Spec.InitContainers[0].VolumeMounts
	volumeMountsWithContent := res.Job.Spec.Template.Spec.InitContainers[1].VolumeMounts

	want := corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyNever,
		Volumes:       volumes,
		InitContainers: []corev1.Container{
			{
				Name:            "tktw-init",
				Image:           defaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/bin/sh", "-c"},
				Args:            []string{"cp /init /.tktw/init && touch /.tktw/state && chmod 777 /.tktw/state && (echo -n ',0' > /dev/termination-log && echo 'Done' && exit 0) || (echo -n 'failed,1' > /dev/termination-log && exit 1)"},
				VolumeMounts:    volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(defaultFsGroup),
				},
			},
			{
				Name:            sig[0].Ref(),
				ImagePullPolicy: "",
				Image:           defaultImage,
				Command: []string{
					"/.tktw/init",
					sig[0].Ref(),
					"-c", fmt.Sprintf("%s,%s=passed", sig[0].Ref(), sig[1].Ref()),
					"-r", fmt.Sprintf("=%s&&%s", sig[0].Ref(), sig[1].Ref()),
					"--",
				},
				Args:         []string{defaultShell, "-c", "shell-test"},
				WorkingDir:   "",
				EnvFrom:      []corev1.EnvFromSource(nil),
				Env:          []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:    corev1.ResourceRequirements{},
				VolumeMounts: volumeMountsWithContent,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(defaultFsGroup),
				},
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
				Args:         []string{defaultShell, "-c", "shell-test-2"},
				WorkingDir:   "",
				EnvFrom:      []corev1.EnvFromSource(nil),
				Env:          []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:    corev1.ResourceRequirements{},
				VolumeMounts: volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(defaultFsGroup),
				},
			},
		},
		SecurityContext: &corev1.PodSecurityContext{
			FSGroup: common.Ptr(defaultFsGroup),
		},
	}

	assert.Equal(t, want, res.Job.Spec.Template.Spec)
	assert.Equal(t, 2, len(volumeMounts))
	assert.Equal(t, 3, len(volumeMountsWithContent))
	assert.Equal(t, volumeMounts, volumeMountsWithContent[:2])
	assert.Equal(t, "/some/path", volumeMountsWithContent[2].MountPath)
	assert.Equal(t, 1, len(res.ConfigMaps))
	assert.Equal(t, volumeMountsWithContent[2].Name, volumes[2].Name)
	assert.Equal(t, volumes[2].ConfigMap.Name, res.ConfigMaps[0].Name)
	assert.Equal(t, "some-{{content", res.ConfigMaps[0].Data[volumeMountsWithContent[2].SubPath])
}

func TestProcessGlobalContent(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Content: &testworkflowsv1.Content{
					Files: []testworkflowsv1.ContentFile{{
						Path:    "/some/path",
						Content: `some-{{"{{"}}content`,
					}},
				},
			},
			Steps: []testworkflowsv1.Step{
				{StepBase: testworkflowsv1.StepBase{Shell: "shell-test"}},
				{StepBase: testworkflowsv1.StepBase{Shell: "shell-test-2"}},
			},
		},
	}

	res, err := proc.Bundle(context.Background(), wf, execMachine)
	assert.NoError(t, err)

	sig := res.Signature

	volumes := res.Job.Spec.Template.Spec.Volumes
	volumeMounts := res.Job.Spec.Template.Spec.InitContainers[0].VolumeMounts

	want := corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyNever,
		Volumes:       volumes,
		InitContainers: []corev1.Container{
			{
				Name:            "tktw-init",
				Image:           defaultInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/bin/sh", "-c"},
				Args:            []string{"cp /init /.tktw/init && touch /.tktw/state && chmod 777 /.tktw/state && (echo -n ',0' > /dev/termination-log && echo 'Done' && exit 0) || (echo -n 'failed,1' > /dev/termination-log && exit 1)"},
				VolumeMounts:    volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(defaultFsGroup),
				},
			},
			{
				Name:            sig[0].Ref(),
				ImagePullPolicy: "",
				Image:           defaultImage,
				Command: []string{
					"/.tktw/init",
					sig[0].Ref(),
					"-c", fmt.Sprintf("%s,%s=passed", sig[0].Ref(), sig[1].Ref()),
					"-r", fmt.Sprintf("=%s&&%s", sig[0].Ref(), sig[1].Ref()),
					"--",
				},
				Args:         []string{defaultShell, "-c", "shell-test"},
				WorkingDir:   "",
				EnvFrom:      []corev1.EnvFromSource(nil),
				Env:          []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:    corev1.ResourceRequirements{},
				VolumeMounts: volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(defaultFsGroup),
				},
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
				Args:         []string{defaultShell, "-c", "shell-test-2"},
				WorkingDir:   "",
				EnvFrom:      []corev1.EnvFromSource(nil),
				Env:          []corev1.EnvVar{{Name: "CI", Value: "1"}},
				Resources:    corev1.ResourceRequirements{},
				VolumeMounts: volumeMounts,
				SecurityContext: &corev1.SecurityContext{
					RunAsGroup: common.Ptr(defaultFsGroup),
				},
			},
		},
		SecurityContext: &corev1.PodSecurityContext{
			FSGroup: common.Ptr(defaultFsGroup),
		},
	}

	assert.Equal(t, want, res.Job.Spec.Template.Spec)
	assert.Equal(t, 3, len(volumeMounts))
	assert.Equal(t, "/some/path", volumeMounts[2].MountPath)
	assert.Equal(t, 1, len(res.ConfigMaps))
	assert.Equal(t, volumeMounts[2].Name, volumes[2].Name)
	assert.Equal(t, volumes[2].ConfigMap.Name, res.ConfigMaps[0].Name)
	assert.Equal(t, "some-{{content", res.ConfigMaps[0].Data[volumeMounts[2].SubPath])
}
