// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowprocessor

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
	"github.com/kubeshop/testkube/pkg/tcl/mapperstcl/testworkflows"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor/constants"
)

func ProcessDelay(_ InternalProcessor, layer Intermediate, container Container, step testworkflowsv1.Step) (Stage, error) {
	if step.Delay == "" {
		return nil, nil
	}
	t, err := time.ParseDuration(step.Delay)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("invalid duration: %s", step.Delay))
	}
	shell := container.CreateChild().
		SetCommand("sleep").
		SetArgs(fmt.Sprintf("%g", t.Seconds()))
	stage := NewContainerStage(layer.NextRef(), shell)
	stage.SetCategory(fmt.Sprintf("Delay: %s", step.Delay))
	return stage, nil
}

func ProcessShellCommand(_ InternalProcessor, layer Intermediate, container Container, step testworkflowsv1.Step) (Stage, error) {
	if step.Shell == "" {
		return nil, nil
	}
	shell := container.CreateChild().SetCommand(constants.DefaultShellPath).SetArgs("-c", constants.DefaultShellHeader+step.Shell)
	stage := NewContainerStage(layer.NextRef(), shell)
	stage.SetCategory("Run shell command")
	stage.SetRetryPolicy(step.Retry)
	return stage, nil
}

func ProcessRunCommand(_ InternalProcessor, layer Intermediate, container Container, step testworkflowsv1.Step) (Stage, error) {
	if step.Run == nil {
		return nil, nil
	}
	container = container.CreateChild().ApplyCR(&step.Run.ContainerConfig)
	stage := NewContainerStage(layer.NextRef(), container)
	stage.SetRetryPolicy(step.Retry)
	stage.SetCategory("Run")
	if step.Run.Shell != nil {
		if step.Run.ContainerConfig.Command != nil || step.Run.ContainerConfig.Args != nil {
			return nil, errors.New("run.shell should not be used in conjunction with run.command or run.args")
		}
		stage.SetCategory("Run shell command")
		stage.Container().SetCommand(constants.DefaultShellPath).SetArgs("-c", constants.DefaultShellHeader+*step.Run.Shell)
	}
	return stage, nil
}

func ProcessNestedSetupSteps(p InternalProcessor, layer Intermediate, container Container, step testworkflowsv1.Step) (Stage, error) {
	group := NewGroupStage(layer.NextRef(), true)
	for _, n := range step.Setup {
		stage, err := p.Process(layer, container.CreateChild(), n)
		if err != nil {
			return nil, err
		}
		group.Add(stage)
	}
	return group, nil
}

func ProcessNestedSteps(p InternalProcessor, layer Intermediate, container Container, step testworkflowsv1.Step) (Stage, error) {
	group := NewGroupStage(layer.NextRef(), true)
	for _, n := range step.Steps {
		stage, err := p.Process(layer, container.CreateChild(), n)
		if err != nil {
			return nil, err
		}
		group.Add(stage)
	}
	return group, nil
}

func ProcessExecute(_ InternalProcessor, layer Intermediate, container Container, step testworkflowsv1.Step) (Stage, error) {
	if step.Execute == nil {
		return nil, nil
	}
	container = container.CreateChild()
	stage := NewContainerStage(layer.NextRef(), container)
	stage.SetRetryPolicy(step.Retry)
	hasWorkflows := len(step.Execute.Workflows) > 0
	hasTests := len(step.Execute.Tests) > 0

	// Fail if there is nothing to run
	if !hasTests && !hasWorkflows {
		return nil, errors.New("no test workflows and tests provided to the 'execute' step")
	}

	container.
		SetImage(constants.DefaultToolkitImage).
		SetImagePullPolicy(corev1.PullIfNotPresent).
		SetCommand("/toolkit", "execute").
		EnableToolkit(stage.Ref()).
		AppendVolumeMounts(layer.AddEmptyDirVolume(nil, constants.DefaultTransferDirPath))
	args := make([]string, 0)
	for _, t := range step.Execute.Tests {
		b, err := json.Marshal(t)
		if err != nil {
			return nil, errors.Wrap(err, "execute: serializing Test")
		}
		args = append(args, "-t", expressionstcl.NewStringValue(string(b)).Template())
	}
	for _, w := range step.Execute.Workflows {
		b, err := json.Marshal(w)
		if err != nil {
			return nil, errors.Wrap(err, "execute: serializing TestWorkflow")
		}
		args = append(args, "-w", expressionstcl.NewStringValue(string(b)).Template())
	}
	if step.Execute.Async {
		args = append(args, "--async")
	}
	if step.Execute.Parallelism > 0 {
		args = append(args, "-p", strconv.Itoa(int(step.Execute.Parallelism)))
	}
	container.SetArgs(args...)

	// Add default label
	types := make([]string, 0)
	if hasWorkflows {
		types = append(types, "test workflows")
	}
	if hasTests {
		types = append(types, "tests")
	}
	stage.SetCategory("Execute " + strings.Join(types, " & "))

	return stage, nil
}

func ProcessContentFiles(_ InternalProcessor, layer Intermediate, container Container, step testworkflowsv1.Step) (Stage, error) {
	if step.Content == nil {
		return nil, nil
	}
	for _, f := range step.Content.Files {
		if f.ContentFrom == nil {
			vm, err := layer.AddTextFile(f.Content)
			if err != nil {
				return nil, fmt.Errorf("file %s: could not append: %s", f.Path, err.Error())
			}
			vm.MountPath = f.Path
			container.AppendVolumeMounts(vm)
			continue
		}

		volRef := "{{execution.id}}-" + layer.NextRef()

		if f.ContentFrom.ConfigMapKeyRef != nil {
			layer.AddVolume(corev1.Volume{
				Name: volRef,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: f.ContentFrom.ConfigMapKeyRef.LocalObjectReference,
						Items:                []corev1.KeyToPath{{Key: f.ContentFrom.ConfigMapKeyRef.Key, Path: "file"}},
						DefaultMode:          f.Mode,
						Optional:             f.ContentFrom.ConfigMapKeyRef.Optional,
					},
				},
			})
			container.AppendVolumeMounts(corev1.VolumeMount{Name: volRef, MountPath: f.Path, SubPath: "file"})
		} else if f.ContentFrom.SecretKeyRef != nil {
			layer.AddVolume(corev1.Volume{
				Name: volRef,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName:  f.ContentFrom.SecretKeyRef.Name,
						Items:       []corev1.KeyToPath{{Key: f.ContentFrom.SecretKeyRef.Key, Path: "file"}},
						DefaultMode: f.Mode,
						Optional:    f.ContentFrom.SecretKeyRef.Optional,
					},
				},
			})
			container.AppendVolumeMounts(corev1.VolumeMount{Name: volRef, MountPath: f.Path, SubPath: "file"})
		} else if f.ContentFrom.FieldRef != nil || f.ContentFrom.ResourceFieldRef != nil {
			layer.AddVolume(corev1.Volume{
				Name: volRef,
				VolumeSource: corev1.VolumeSource{
					Projected: &corev1.ProjectedVolumeSource{
						Sources: []corev1.VolumeProjection{{
							DownwardAPI: &corev1.DownwardAPIProjection{
								Items: []corev1.DownwardAPIVolumeFile{{
									Path:             "file",
									FieldRef:         f.ContentFrom.FieldRef,
									ResourceFieldRef: f.ContentFrom.ResourceFieldRef,
									Mode:             f.Mode,
								}},
							},
						}},
						DefaultMode: f.Mode,
					},
				},
			})
			container.AppendVolumeMounts(corev1.VolumeMount{Name: volRef, MountPath: f.Path, SubPath: "file"})
		} else {
			return nil, fmt.Errorf("file %s: unrecognized ContentFrom provided for file", f.Path)
		}
	}
	return nil, nil
}

func ProcessContentGit(_ InternalProcessor, layer Intermediate, container Container, step testworkflowsv1.Step) (Stage, error) {
	if step.Content == nil || step.Content.Git == nil {
		return nil, nil
	}

	selfContainer := container.CreateChild()
	stage := NewContainerStage(layer.NextRef(), selfContainer)
	stage.SetRetryPolicy(step.Retry)
	stage.SetCategory("Clone Git repository")

	// Compute mount path
	mountPath := step.Content.Git.MountPath
	if mountPath == "" {
		mountPath = filepath.Join(constants.DefaultDataPath, "repo")
	}

	// Build volume pair and share with all siblings
	volumeMount := layer.AddEmptyDirVolume(nil, mountPath)
	container.AppendVolumeMounts(volumeMount)

	selfContainer.
		SetWorkingDir("/").
		SetImage(constants.DefaultToolkitImage).
		SetImagePullPolicy(corev1.PullIfNotPresent).
		SetCommand("/toolkit", "clone", step.Content.Git.Uri).
		EnableToolkit(stage.Ref())

	args := []string{mountPath}

	// Provide Git username
	if step.Content.Git.UsernameFrom != nil {
		container.AppendEnv(corev1.EnvVar{Name: "TK_GIT_USERNAME", ValueFrom: step.Content.Git.UsernameFrom})
		args = append(args, "-u", "{{env.TK_GIT_USERNAME}}")
	} else if step.Content.Git.Username != "" {
		args = append(args, "-u", step.Content.Git.Username)
	}

	// Provide Git token
	if step.Content.Git.TokenFrom != nil {
		container.AppendEnv(corev1.EnvVar{Name: "TK_GIT_TOKEN", ValueFrom: step.Content.Git.TokenFrom})
		args = append(args, "-t", "{{env.TK_GIT_TOKEN}}")
	} else if step.Content.Git.Token != "" {
		args = append(args, "-t", step.Content.Git.Token)
	}

	// Provide auth type
	if step.Content.Git.AuthType != "" {
		args = append(args, "-a", string(step.Content.Git.AuthType))
	}

	// Provide revision
	if step.Content.Git.Revision != "" {
		args = append(args, "-r", step.Content.Git.Revision)
	}

	// Provide sparse paths
	if len(step.Content.Git.Paths) > 0 {
		for _, pattern := range step.Content.Git.Paths {
			args = append(args, "-p", pattern)
		}
	}

	selfContainer.SetArgs(args...)

	return stage, nil
}

func ProcessContentTarball(_ InternalProcessor, layer Intermediate, container Container, step testworkflowsv1.Step) (Stage, error) {
	if step.Content == nil || len(step.Content.Tarball) == 0 {
		return nil, nil
	}

	selfContainer := container.CreateChild()
	stage := NewContainerStage(layer.NextRef(), selfContainer)
	stage.SetRetryPolicy(step.Retry)
	stage.SetCategory("Download tarball")

	selfContainer.
		SetImage(constants.DefaultToolkitImage).
		SetImagePullPolicy(corev1.PullIfNotPresent).
		SetCommand("/toolkit", "tarball").
		EnableToolkit(stage.Ref())

	// Build volume pair and share with all siblings
	args := make([]string, len(step.Content.Tarball))
	for i, t := range step.Content.Tarball {
		args[i] = fmt.Sprintf("%s=%s", t.Path, t.Url)
		needsMount := t.Mount != nil && *t.Mount
		if !needsMount {
			needsMount = selfContainer.HasVolumeAt(t.Path)
		}

		if needsMount && t.Mount != nil && !*t.Mount {
			return nil, fmt.Errorf("content.tarball[%d]: %s: is not part of any volume: should be mounted", i, t.Path)
		}

		if (needsMount && t.Mount == nil) || (t.Mount == nil && *t.Mount) {
			volumeMount := layer.AddEmptyDirVolume(nil, t.Path)
			container.AppendVolumeMounts(volumeMount)
		}
	}
	selfContainer.SetArgs(args...)

	return stage, nil
}

func ProcessArtifacts(_ InternalProcessor, layer Intermediate, container Container, step testworkflowsv1.Step) (Stage, error) {
	if step.Artifacts == nil {
		return nil, nil
	}

	if len(step.Artifacts.Paths) == 0 {
		return nil, errors.New("there needs to be at least one path to scrap for artifacts")
	}

	selfContainer := container.CreateChild().
		ApplyCR(&testworkflowsv1.ContainerConfig{WorkingDir: step.Artifacts.WorkingDir})
	stage := NewContainerStage(layer.NextRef(), selfContainer)
	stage.SetRetryPolicy(step.Retry)
	stage.SetCondition("always")
	stage.SetCategory("Upload artifacts")

	selfContainer.
		SetImage(constants.DefaultToolkitImage).
		SetImagePullPolicy(corev1.PullIfNotPresent).
		SetCommand("/toolkit", "artifacts", "-m", constants.DefaultDataPath).
		EnableToolkit(stage.Ref())

	args := make([]string, 0)
	if step.Artifacts.Compress != nil {
		args = append(args, "--compress", step.Artifacts.Compress.Name)
	}
	args = append(args, step.Artifacts.Paths...)
	selfContainer.SetArgs(args...)

	return stage, nil
}

func ProcessDistribute(_ InternalProcessor, layer Intermediate, container Container, step testworkflowsv1.Step) (Stage, error) {
	if len(step.Distribute) == 0 {
		return nil, nil
	}

	podsRef := layer.NextRef()
	container.AppendEnv(corev1.EnvVar{Name: "TK_SRV_REF", Value: podsRef})

	stage := NewContainerStage(layer.NextRef(), container.CreateChild())
	stage.SetCategory("Run distributed pods")

	stage.Container().
		SetImage(defaultToolkitImage).
		SetImagePullPolicy(corev1.PullIfNotPresent).
		SetCommand("/toolkit", "spawn", "{{env.TK_SRV_REF}}").
		EnableToolkit(stage.Ref())

	args := make([]string, 0)
	for k, s := range step.Distribute {
		instruction := s.DeepCopy()
		if s.Container != nil {
			instruction.Pod.Spec.Containers = append(instruction.Pod.Spec.Containers, *s.Container)
		}
		b, err := json.Marshal(instruction.SpawnInstructionBase)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("spawn[%s]: marshaling error", k))
		}
		args = append(args, "-i", fmt.Sprintf("%s=%s", k, expressionstcl.NewStringValue(string(b)).Template()))
	}
	stage.Container().SetArgs(args...)

	return stage, nil
}

func ProcessServicesStart(_ InternalProcessor, layer Intermediate, container Container, step testworkflowsv1.Step) (Stage, error) {
	if len(step.Services) == 0 {
		return nil, nil
	}

	podsRef := layer.NextRef()
	container.AppendEnv(corev1.EnvVar{Name: "TK_SRV_REF", Value: podsRef})

	stage := NewContainerStage(layer.NextRef(), container.CreateChild())
	stage.SetCategory("Start assisting pods")

	stage.Container().
		SetImage(defaultToolkitImage).
		SetImagePullPolicy(corev1.PullIfNotPresent).
		SetCommand("/toolkit", "spawn", "{{env.TK_SRV_REF}}").
		EnableToolkit(stage.Ref())

	args := []string{"--services"}
	for k, s := range step.Services {
		instruction := s.DeepCopy()
		if s.Container != nil {
			instruction.Pod.Spec.Containers = append(instruction.Pod.Spec.Containers, *s.Container)
		}
		b, err := json.Marshal(instruction.SpawnInstructionBase)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("spawn[%s]: marshaling error", k))
		}
		args = append(args, "-i", fmt.Sprintf("%s=%s", k, expressionstcl.NewStringValue(string(b)).Template()))
	}
	stage.Container().SetArgs(args...)

	return stage, nil
}

func ProcessServicesStop(_ InternalProcessor, layer Intermediate, container Container, step testworkflowsv1.Step) (Stage, error) {
	if len(step.Services) == 0 {
		return nil, nil
	}

	stage := NewContainerStage(layer.NextRef(), container.CreateChild())
	stage.SetCondition("always") // TODO: actually, it's enough to do it when ServicesInit is not skipped
	stage.SetOptional(true)
	stage.SetRetryPolicy(step.Retry)
	stage.SetCategory("Cleanup assisting pods")

	stage.Container().
		SetImage(defaultToolkitImage).
		SetImagePullPolicy(corev1.PullIfNotPresent).
		SetCommand("/toolkit", "kill", "{{env.TK_SRV_REF}}").
		EnableToolkit(stage.Ref())

	return stage, nil
}
