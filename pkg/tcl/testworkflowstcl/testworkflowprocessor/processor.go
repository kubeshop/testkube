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
	"maps"
	"path/filepath"

	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/imageinspector"
	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
)

//go:generate mockgen -destination=./mock_processor.go -package=testworkflowprocessor "github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor" Processor
type Processor interface {
	Register(operation Operation) Processor
	Bundle(ctx context.Context, workflow *testworkflowsv1.TestWorkflow, machines ...expressionstcl.Machine) (*Bundle, error)
}

//go:generate mockgen -destination=./mock_internalprocessor.go -package=testworkflowprocessor "github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor" InternalProcessor
type InternalProcessor interface {
	Process(layer Intermediate, container Container, step testworkflowsv1.Step) (Stage, error)
}

type Operation = func(processor InternalProcessor, layer Intermediate, container Container, step testworkflowsv1.Step) (Stage, error)

type processor struct {
	inspector  imageinspector.Inspector
	operations []Operation
}

func New(inspector imageinspector.Inspector) Processor {
	return &processor{inspector: inspector}
}

func NewFullFeatured(inspector imageinspector.Inspector) Processor {
	return New(inspector).
		Register(ProcessDelay).
		Register(ProcessContentFiles).
		Register(ProcessContentGit).
		Register(ProcessNestedSetupSteps).
		Register(ProcessRunCommand).
		Register(ProcessShellCommand).
		Register(ProcessExecute).
		Register(ProcessNestedSteps).
		Register(ProcessArtifacts)
}

func (p *processor) Register(operation Operation) Processor {
	p.operations = append(p.operations, operation)
	return p
}

func (p *processor) process(layer Intermediate, container Container, step testworkflowsv1.Step, ref string) (Stage, error) {
	// Configure defaults
	if step.WorkingDir != nil {
		container.SetWorkingDir(*step.WorkingDir)
	}
	container.ApplyCR(step.Container)

	// Build an initial group for the inner items
	self := NewGroupStage(ref, false)
	self.SetName(step.Name)
	self.SetOptional(step.Optional).SetNegative(step.Negative).SetTimeout(step.Timeout)
	if step.Condition != "" {
		self.SetCondition(step.Condition)
	} else {
		self.SetCondition("passed")
	}

	// Run operations
	for _, op := range p.operations {
		stage, err := op(p, layer, container, step)
		if err != nil {
			return nil, err
		}
		self.Add(stage)
	}

	return self, nil
}

func (p *processor) Process(layer Intermediate, container Container, step testworkflowsv1.Step) (Stage, error) {
	return p.process(layer, container, step, layer.NextRef())
}

func (p *processor) Bundle(ctx context.Context, workflow *testworkflowsv1.TestWorkflow, machines ...expressionstcl.Machine) (bundle *Bundle, err error) {
	// Initialize intermediate layer
	layer := NewIntermediate().
		AppendPodConfig(workflow.Spec.Pod).
		AppendJobConfig(workflow.Spec.Job)
	layer.ContainerDefaults().
		ApplyCR(defaultContainerConfig.DeepCopy()).
		AppendVolumeMounts(layer.AddEmptyDirVolume(nil, defaultInternalPath)).
		AppendVolumeMounts(layer.AddEmptyDirVolume(nil, defaultDataPath))

	// Process steps
	rootStep := testworkflowsv1.Step{
		StepBase: testworkflowsv1.StepBase{
			Content:   workflow.Spec.Content,
			Container: workflow.Spec.Container,
		},
		Steps: append(workflow.Spec.Setup, append(workflow.Spec.Steps, workflow.Spec.After...)...),
	}
	root, err := p.process(layer, layer.ContainerDefaults(), rootStep, "")
	if err != nil {
		return nil, errors.Wrap(err, "processing error")
	}

	// Validate if there is anything to run
	if root.Len() == 0 {
		return nil, errors.New("test workflow has nothing to run")
	}

	// Finalize ConfigMaps
	configMaps := layer.ConfigMaps()
	for i := range configMaps {
		AnnotateControlledBy(&configMaps[i], "{{execution.id}}")
		err = expressionstcl.FinalizeForce(&configMaps[i], machines...)
		if err != nil {
			return nil, errors.Wrap(err, "finalizing ConfigMap")
		}
	}

	// Finalize Secrets
	secrets := layer.Secrets()
	for i := range secrets {
		AnnotateControlledBy(&secrets[i], "{{execution.id}}")
		err = expressionstcl.FinalizeForce(&secrets[i], machines...)
		if err != nil {
			return nil, errors.Wrap(err, "finalizing Secret")
		}
	}

	// Finalize Volumes
	volumes := layer.Volumes()
	for i := range volumes {
		err = expressionstcl.FinalizeForce(&volumes[i], machines...)
		if err != nil {
			return nil, errors.Wrap(err, "finalizing Volume")
		}
	}

	// Append main label for the pod
	layer.AppendPodConfig(&testworkflowsv1.PodConfig{
		Labels: map[string]string{
			ExecutionIdMainPodLabelName: "{{execution.id}}",
		},
	})

	// Resolve job & pod config
	jobConfig, podConfig := layer.JobConfig(), layer.PodConfig()
	err = expressionstcl.FinalizeForce(&jobConfig, machines...)
	if err != nil {
		return nil, errors.Wrap(err, "finalizing job config")
	}
	err = expressionstcl.FinalizeForce(&podConfig, machines...)
	if err != nil {
		return nil, errors.Wrap(err, "finalizing pod config")
	}

	// Build signature
	sig := root.Signature().Children()

	// Load the image pull secrets
	pullSecretNames := make([]string, len(podConfig.ImagePullSecrets))
	for i, v := range podConfig.ImagePullSecrets {
		pullSecretNames[i] = v.Name
	}

	// Load the image details
	imageNames := root.GetImages()
	images := make(map[string]*imageinspector.Info)
	for image := range imageNames {
		info, err := p.inspector.Inspect(ctx, "", image, corev1.PullIfNotPresent, pullSecretNames)
		if err != nil {
			return nil, fmt.Errorf("resolving image error: %s: %s", image, err.Error())
		}
		images[image] = info
	}
	err = root.ApplyImages(images)
	if err != nil {
		return nil, errors.Wrap(err, "applying image data")
	}

	// Build list of the containers
	containers, err := buildKubernetesContainers(root, NewInitProcess().SetRef(root.Ref()), machines...)
	if err != nil {
		return nil, errors.Wrap(err, "building Kubernetes containers")
	}
	for i := range containers {
		err = expressionstcl.FinalizeForce(&containers[i].EnvFrom, machines...)
		if err != nil {
			return nil, errors.Wrap(err, "finalizing container's envFrom")
		}
		err = expressionstcl.FinalizeForce(&containers[i].VolumeMounts, machines...)
		if err != nil {
			return nil, errors.Wrap(err, "finalizing container's volumeMounts")
		}
		err = expressionstcl.FinalizeForce(&containers[i].Resources, machines...)
		if err != nil {
			return nil, errors.Wrap(err, "finalizing container's resources")
		}

		// Resolve relative paths in the volumeMounts relatively to the working dir
		workingDir := defaultDataPath
		if containers[i].WorkingDir != "" {
			workingDir = containers[i].WorkingDir
		}
		for j := range containers[i].VolumeMounts {
			if !filepath.IsAbs(containers[i].VolumeMounts[j].MountPath) {
				containers[i].VolumeMounts[j].MountPath = filepath.Clean(filepath.Join(workingDir, containers[i].VolumeMounts[j].MountPath))
			}
		}
	}

	// Build pod template
	podSpec := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: podConfig.Annotations,
			Labels:      podConfig.Labels,
		},
		Spec: corev1.PodSpec{
			RestartPolicy:      corev1.RestartPolicyNever,
			Volumes:            volumes,
			ImagePullSecrets:   podConfig.ImagePullSecrets,
			ServiceAccountName: podConfig.ServiceAccountName,
			NodeSelector:       podConfig.NodeSelector,
			SecurityContext: &corev1.PodSecurityContext{
				FSGroup: common.Ptr(defaultFsGroup),
			},
		},
	}
	AnnotateControlledBy(&podSpec, "{{execution.id}}")
	err = expressionstcl.FinalizeForce(&podSpec, machines...)
	if err != nil {
		return nil, errors.Wrap(err, "finalizing pod template spec")
	}
	initContainer := corev1.Container{
		Name:            "tktw-init",
		Image:           defaultInitImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"/bin/sh", "-c"},
		Args:            []string{fmt.Sprintf("cp /init %s && touch %s && chmod 777 %s && (echo -n ',0' > %s && echo 'Done' && exit 0) || (echo -n 'failed,1' > %s && exit 1)", defaultInitPath, defaultStatePath, defaultStatePath, "/dev/termination-log", "/dev/termination-log")},
		VolumeMounts:    layer.ContainerDefaults().VolumeMounts(),
		SecurityContext: &corev1.SecurityContext{
			RunAsGroup: common.Ptr(defaultFsGroup),
		},
	}
	err = expressionstcl.FinalizeForce(&initContainer, machines...)
	if err != nil {
		return nil, errors.Wrap(err, "finalizing container's resources")
	}
	podSpec.Spec.InitContainers = append([]corev1.Container{initContainer}, containers[:len(containers)-1]...)
	podSpec.Spec.Containers = containers[len(containers)-1:]

	// Build job spec
	jobSpec := batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: batchv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "{{execution.id}}",
			Annotations: jobConfig.Annotations,
			Labels:      jobConfig.Labels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: common.Ptr(int32(0)),
		},
	}
	AnnotateControlledBy(&jobSpec, "{{execution.id}}")
	err = expressionstcl.FinalizeForce(&jobSpec, machines...)
	if err != nil {
		return nil, errors.Wrap(err, "finalizing job spec")
	}
	jobSpec.Spec.Template = podSpec

	// Build signature
	sigSerialized, _ := json.Marshal(sig)
	jobAnnotations := make(map[string]string)
	maps.Copy(jobAnnotations, jobSpec.Annotations)
	maps.Copy(jobAnnotations, map[string]string{
		SignatureAnnotationName: string(sigSerialized),
	})
	jobSpec.Annotations = jobAnnotations

	// Build bundle
	bundle = &Bundle{
		ConfigMaps: configMaps,
		Secrets:    secrets,
		Job:        jobSpec,
		Signature:  sig,
	}
	return bundle, nil
}
