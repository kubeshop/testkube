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
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor/constants"
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
		Register(ProcessContentTarball).
		Register(ProcessServicesStart).
		Register(ProcessNestedSetupSteps).
		Register(ProcessRunCommand).
		Register(ProcessShellCommand).
		Register(ProcessExecute).
		Register(ProcessParallel).
		Register(ProcessNestedSteps).
		Register(ProcessServicesStop).
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
	self.SetOptional(step.Optional).SetNegative(step.Negative).SetTimeout(step.Timeout).SetPaused(step.Paused)
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

	// Add virtual pause step in case no other is there
	if self.HasPause() && len(self.Children()) == 0 {
		pause := NewContainerStage(self.Ref()+"pause", container.CreateChild().
			SetCommand(constants.DefaultShellPath).
			SetArgs("-c", "exit 0"))
		pause.SetCategory("Wait for continue")
		self.Add(pause)
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
		ApplyCR(constants.DefaultContainerConfig.DeepCopy()).
		AppendVolumeMounts(layer.AddEmptyDirVolume(nil, constants.DefaultInternalPath)).
		AppendVolumeMounts(layer.AddEmptyDirVolume(nil, constants.DefaultDataPath))

	// Process steps
	rootStep := testworkflowsv1.Step{
		StepSource: testworkflowsv1.StepSource{
			Content: workflow.Spec.Content,
		},
		Services: workflow.Spec.Services,
		StepDefaults: testworkflowsv1.StepDefaults{
			Container: workflow.Spec.Container,
		},
		Steps: append(workflow.Spec.Setup, append(workflow.Spec.Steps, workflow.Spec.After...)...),
	}
	err = expressionstcl.Simplify(&workflow, machines...)
	if err != nil {
		return nil, errors.Wrap(err, "error while simplifying workflow instructions")
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
		AnnotateControlledBy(&configMaps[i], "{{resource.root}}", "{{resource.id}}")
		err = expressionstcl.FinalizeForce(&configMaps[i], machines...)
		if err != nil {
			return nil, errors.Wrap(err, "finalizing ConfigMap")
		}
	}

	// Finalize Secrets
	secrets := layer.Secrets()
	for i := range secrets {
		AnnotateControlledBy(&secrets[i], "{{resource.root}}", "{{resource.id}}")
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
			constants.RootResourceIdLabelName: "{{resource.root}}",
			constants.ResourceIdLabelName:     "{{resource.id}}",
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

	// Adjust the security context in case it's a single container besides the Testkube' containers
	// TODO: Consider flag argument, that would be used only for services?
	containerStages := root.ContainerStages()
	var otherContainers []ContainerStage
	for _, c := range containerStages {
		if c.Container().Image() != constants.DefaultInitImage && c.Container().Image() != constants.DefaultToolkitImage {
			otherContainers = append(otherContainers, c)
		}
	}
	if len(otherContainers) == 1 {
		image := otherContainers[0].Container().Image()
		if _, ok := images[image]; ok {
			sc := otherContainers[0].Container().SecurityContext()
			if sc == nil {
				sc = &corev1.SecurityContext{}
			}
			if podConfig.SecurityContext == nil {
				podConfig.SecurityContext = &corev1.PodSecurityContext{}
			}
			if sc.RunAsGroup == nil && podConfig.SecurityContext.RunAsGroup == nil {
				sc.RunAsGroup = common.Ptr(images[image].Group)
				otherContainers[0].Container().SetSecurityContext(sc)
			}
			if podConfig.SecurityContext.FSGroup == nil {
				podConfig.SecurityContext.FSGroup = sc.RunAsGroup
			}
		}
	}
	containerStages = nil

	// Determine FS Group for the containers
	fsGroup := common.Ptr(constants.DefaultFsGroup)
	if podConfig.SecurityContext != nil && podConfig.SecurityContext.FSGroup != nil {
		fsGroup = podConfig.SecurityContext.FSGroup
	}

	// Build list of the containers
	containers, err := buildKubernetesContainers(root, NewInitProcess().SetRef(root.Ref()), fsGroup, machines...)
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
		workingDir := constants.DefaultDataPath
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
	if podConfig.SecurityContext == nil {
		podConfig.SecurityContext = &corev1.PodSecurityContext{}
	}
	if podConfig.SecurityContext.FSGroup == nil {
		podConfig.SecurityContext.FSGroup = common.Ptr(constants.DefaultFsGroup)
	}
	podSpec := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: podConfig.Annotations,
			Labels:      podConfig.Labels,
		},
		Spec: corev1.PodSpec{
			RestartPolicy:             corev1.RestartPolicyNever,
			EnableServiceLinks:        common.Ptr(false),
			Volumes:                   volumes,
			ImagePullSecrets:          podConfig.ImagePullSecrets,
			ServiceAccountName:        podConfig.ServiceAccountName,
			NodeSelector:              podConfig.NodeSelector,
			ActiveDeadlineSeconds:     podConfig.ActiveDeadlineSeconds,
			DNSPolicy:                 podConfig.DNSPolicy,
			NodeName:                  podConfig.NodeName,
			SecurityContext:           podConfig.SecurityContext,
			Hostname:                  podConfig.Hostname,
			Subdomain:                 podConfig.Subdomain,
			Affinity:                  podConfig.Affinity,
			Tolerations:               podConfig.Tolerations,
			HostAliases:               podConfig.HostAliases,
			PriorityClassName:         podConfig.PriorityClassName,
			Priority:                  podConfig.Priority,
			DNSConfig:                 podConfig.DNSConfig,
			PreemptionPolicy:          podConfig.PreemptionPolicy,
			TopologySpreadConstraints: podConfig.TopologySpreadConstraints,
			SchedulingGates:           podConfig.SchedulingGates,
			ResourceClaims:            podConfig.ResourceClaims,
		},
	}
	AnnotateControlledBy(&podSpec, "{{resource.root}}", "{{resource.id}}")
	err = expressionstcl.FinalizeForce(&podSpec, machines...)
	if err != nil {
		return nil, errors.Wrap(err, "finalizing pod template spec")
	}
	initContainer := corev1.Container{
		Name:            "tktw-init",
		Image:           constants.DefaultInitImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"/bin/sh", "-c"},
		Args:            []string{constants.InitScript},
		VolumeMounts:    layer.ContainerDefaults().VolumeMounts(),
		Env: []corev1.EnvVar{
			{Name: "TK_DEBUG_NODE", ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"},
			}},
			{Name: "TK_DEBUG_POD", ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"},
			}},
			{Name: "TK_DEBUG_NS", ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"},
			}},
			{Name: "TK_DEBUG_SVC", ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.serviceAccountName"},
			}},
		},
		SecurityContext: &corev1.SecurityContext{
			RunAsGroup: fsGroup,
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
			Name:        "{{resource.id}}",
			Annotations: jobConfig.Annotations,
			Labels:      jobConfig.Labels,
			Namespace:   jobConfig.Namespace,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:          common.Ptr(int32(0)),
			ActiveDeadlineSeconds: jobConfig.ActiveDeadlineSeconds,
		},
	}
	AnnotateControlledBy(&jobSpec, "{{resource.root}}", "{{resource.id}}")
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
		constants.SignatureAnnotationName: string(sigSerialized),
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
