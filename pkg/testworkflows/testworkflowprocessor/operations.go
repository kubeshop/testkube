package testworkflowprocessor

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

func ProcessDelay(_ InternalProcessor, layer Intermediate, container stage.Container, step testworkflowsv1.Step) (stage.Stage, error) {
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
	stage := stage.NewContainerStage(layer.NextRef(), shell)
	stage.SetCategory(fmt.Sprintf("Delay: %s", step.Delay))
	return stage, nil
}

func ProcessShellCommand(_ InternalProcessor, layer Intermediate, container stage.Container, step testworkflowsv1.Step) (stage.Stage, error) {
	if step.Shell == "" {
		return nil, nil
	}
	shell := container.CreateChild().SetCommand(constants.DefaultShellPath).SetArgs("-c", constants.DefaultShellHeader+step.Shell)
	stage := stage.NewContainerStage(layer.NextRef(), shell)
	stage.SetCategory("Run shell command")
	stage.SetRetryPolicy(step.Retry)
	return stage, nil
}

func ProcessRunCommand(_ InternalProcessor, layer Intermediate, container stage.Container, step testworkflowsv1.Step) (stage.Stage, error) {
	if step.Run == nil {
		return nil, nil
	}
	container = container.CreateChild().ApplyCR(&step.Run.ContainerConfig)
	stage := stage.NewContainerStage(layer.NextRef(), container)
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

func ProcessNestedSetupSteps(p InternalProcessor, layer Intermediate, container stage.Container, step testworkflowsv1.Step) (stage.Stage, error) {
	group := stage.NewGroupStage(layer.NextRef(), true)
	for _, n := range step.Setup {
		stage, err := p.Process(layer, container.CreateChild(), n)
		if err != nil {
			return nil, err
		}
		group.Add(stage)
	}
	return group, nil
}

func ProcessNestedSteps(p InternalProcessor, layer Intermediate, container stage.Container, step testworkflowsv1.Step) (stage.Stage, error) {
	group := stage.NewGroupStage(layer.NextRef(), true)
	for _, n := range step.Steps {
		stage, err := p.Process(layer, container.CreateChild(), n)
		if err != nil {
			return nil, err
		}
		group.Add(stage)
	}
	return group, nil
}

func ProcessContentFiles(_ InternalProcessor, layer Intermediate, container stage.Container, step testworkflowsv1.Step) (stage.Stage, error) {
	if step.Content == nil {
		return nil, nil
	}
	for _, f := range step.Content.Files {
		if f.ContentFrom == nil {
			vm, err := layer.AddTextFile(f.Content, f.Mode)
			if err != nil {
				return nil, fmt.Errorf("file %s: could not append: %s", f.Path, err.Error())
			}
			vm.MountPath = f.Path
			container.AppendVolumeMounts(vm)
			continue
		}

		volRef := "{{resource.id}}-" + layer.NextRef()

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

func ProcessContentGit(_ InternalProcessor, layer Intermediate, container stage.Container, step testworkflowsv1.Step) (stage.Stage, error) {
	if step.Content == nil || step.Content.Git == nil {
		return nil, nil
	}

	selfContainer := container.CreateChild()
	stage := stage.NewContainerStage(layer.NextRef(), selfContainer)
	stage.SetRetryPolicy(step.Retry)
	stage.SetCategory("Clone Git repository")

	// Compute mount path
	mountPath := step.Content.Git.MountPath
	if mountPath == "" {
		mountPath = filepath.Join(constants.DefaultDataPath, "repo")
	}

	// Build a temporary volume to clone the repository initially.
	// This will allow mounting files in the destination at the same level (i.e. overriding the configuration).
	container.AppendVolumeMounts(layer.AddEmptyDirVolume(nil, constants.DefaultTmpDirPath))

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

	// Provide SSH key
	if step.Content.Git.SshKeyFrom != nil {
		container.AppendEnv(corev1.EnvVar{Name: "TK_SSH_KEY", ValueFrom: step.Content.Git.SshKeyFrom})
		args = append(args, "-s", "{{env.TK_SSH_KEY}}")
	} else if step.Content.Git.SshKey != "" {
		args = append(args, "-s", step.Content.Git.SshKey)
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

func ProcessContentTarball(_ InternalProcessor, layer Intermediate, container stage.Container, step testworkflowsv1.Step) (stage.Stage, error) {
	if step.Content == nil || len(step.Content.Tarball) == 0 {
		return nil, nil
	}

	selfContainer := container.CreateChild()
	stage := stage.NewContainerStage(layer.NextRef(), selfContainer)
	stage.SetRetryPolicy(step.Retry)
	stage.SetCategory("Fetch tarball")

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
			needsMount = !selfContainer.HasVolumeAt(t.Path)
		}

		if needsMount && t.Mount != nil && !*t.Mount {
			return nil, fmt.Errorf("content.tarball[%d]: %s: is not part of any volume: should be mounted", i, t.Path)
		}

		if needsMount {
			volumeMount := layer.AddEmptyDirVolume(nil, t.Path)
			container.AppendVolumeMounts(volumeMount)
		}
	}
	selfContainer.SetArgs(args...)

	return stage, nil
}

func ProcessArtifacts(_ InternalProcessor, layer Intermediate, container stage.Container, step testworkflowsv1.Step) (stage.Stage, error) {
	if step.Artifacts == nil {
		return nil, nil
	}

	if len(step.Artifacts.Paths) == 0 {
		return nil, errors.New("there needs to be at least one path to scrap for artifacts")
	}

	selfContainer := container.CreateChild().
		ApplyCR(&testworkflowsv1.ContainerConfig{WorkingDir: step.Artifacts.WorkingDir})
	stage := stage.NewContainerStage(layer.NextRef(), selfContainer)
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

func StubExecute(_ InternalProcessor, _ Intermediate, _ stage.Container, step testworkflowsv1.Step) (stage.Stage, error) {
	if step.Execute != nil {
		return nil, constants.ErrOpenSourceExecuteOperationIsNotAvailable
	}
	return nil, nil
}

func StubParallel(_ InternalProcessor, _ Intermediate, _ stage.Container, step testworkflowsv1.Step) (stage.Stage, error) {
	if step.Parallel != nil {
		return nil, constants.ErrOpenSourceParallelOperationIsNotAvailable
	}
	return nil, nil
}

func StubServices(_ InternalProcessor, _ Intermediate, _ stage.Container, step testworkflowsv1.Step) (stage.Stage, error) {
	if len(step.Services) != 0 {
		return nil, constants.ErrOpenSourceServicesOperationIsNotAvailable
	}
	return nil, nil
}
