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
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/imageinspector"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/ui"
)

//go:generate mockgen -destination=./mock_processor.go -package=testworkflowprocessor "github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor" Processor
type Processor interface {
	Register(operation Operation) Processor
	Bundle(ctx context.Context, workflow *testworkflowsv1.TestWorkflow, machines ...expressions.Machine) (*Bundle, error)
}

//go:generate mockgen -destination=./mock_internalprocessor.go -package=testworkflowprocessor "github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor" InternalProcessor
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

func (p *processor) Bundle(ctx context.Context, workflow *testworkflowsv1.TestWorkflow, machines ...expressions.Machine) (bundle *Bundle, err error) {
	// Initialize intermediate layer
	layer := NewIntermediate().
		AppendPodConfig(workflow.Spec.Pod).
		AppendJobConfig(workflow.Spec.Job)
	layer.ContainerDefaults().
		ApplyCR(constants.DefaultContainerConfig.DeepCopy()).
		AppendVolumeMounts(layer.AddEmptyDirVolume(nil, constants.DefaultInternalPath)).
		AppendVolumeMounts(layer.AddEmptyDirVolume(nil, constants.DefaultDataPath))

	// Fetch resource root and resource ID
	resourceRoot, err := expressions.EvalExpression("resource.root", machines...)
	if err != nil {
		return nil, errors.Wrap(err, "could not resolve resource.root")
	}
	resourceId, err := expressions.EvalExpression("resource.id", machines...)
	if err != nil {
		return nil, errors.Wrap(err, "could not resolve resource.id")
	}

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
	err = expressions.Simplify(&workflow, machines...)
	if err != nil {
		return nil, errors.Wrap(err, "error while simplifying workflow instructions")
	}
	root, err := p.process(layer, layer.ContainerDefaults(), rootStep, "root")
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
		AnnotateControlledBy(&configMaps[i], resourceRoot.Template(), resourceId.Template())
		err = expressions.FinalizeForce(&configMaps[i], machines...)
		if err != nil {
			return nil, errors.Wrap(err, "finalizing ConfigMap")
		}
	}

	// Finalize Secrets
	secrets := layer.Secrets()
	for i := range secrets {
		AnnotateControlledBy(&secrets[i], resourceRoot.Template(), resourceId.Template())
		err = expressions.FinalizeForce(&secrets[i], machines...)
		if err != nil {
			return nil, errors.Wrap(err, "finalizing Secret")
		}
	}

	// Finalize Volumes
	volumes := layer.Volumes()
	for i := range volumes {
		err = expressions.FinalizeForce(&volumes[i], machines...)
		if err != nil {
			return nil, errors.Wrap(err, "finalizing Volume")
		}
	}

	// Append main label for the pod
	layer.AppendPodConfig(&testworkflowsv1.PodConfig{
		Labels: map[string]string{
			constants.RootResourceIdLabelName: resourceRoot.Template(),
			constants.ResourceIdLabelName:     resourceId.Template(),
		},
	})

	// Resolve job & pod config
	jobConfig, podConfig := layer.JobConfig(), layer.PodConfig()
	err = expressions.FinalizeForce(&jobConfig, machines...)
	if err != nil {
		return nil, errors.Wrap(err, "finalizing job config")
	}
	err = expressions.FinalizeForce(&podConfig, machines...)
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
	imageNameResolutions := map[string]string{}
	for image := range imageNames {
		info, err := p.inspector.Inspect(ctx, "", image, corev1.PullIfNotPresent, pullSecretNames)
		imageNameResolutions[image] = p.inspector.ResolveName("", image)
		if err != nil {
			return nil, fmt.Errorf("resolving image error: %s: %s", image, err.Error())
		}
		images[image] = info
	}
	err = root.ApplyImages(images, imageNameResolutions)
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
	actions, err := AnalyzeOperations(root, machines...)
	if err != nil {
		return nil, errors.Wrap(err, "analyzing Kubernetes container operations")
	}
	actionGroups := GroupActions(actions)
	containers := make([]corev1.Container, len(actionGroups))
	for i := range actionGroups {
		var bareActions []action.Action
		containers[i], bareActions, err = BuildContainer(i, layer.ContainerDefaults(), actionGroups[i])
		actionGroups[i] = bareActions
		if err != nil {
			return nil, errors.Wrap(err, "building Kubernetes containers")
		}
	}

	for i := range containers {
		err = expressions.FinalizeForce(&containers[i].EnvFrom, machines...)
		if err != nil {
			return nil, errors.Wrap(err, "finalizing container's envFrom")
		}
		err = expressions.FinalizeForce(&containers[i].VolumeMounts, machines...)
		if err != nil {
			return nil, errors.Wrap(err, "finalizing container's volumeMounts")
		}
		err = expressions.FinalizeForce(&containers[i].Resources, machines...)
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

		// Avoid having working directory set up, so we have the default one
		containers[i].WorkingDir = ""

		// Ensure the cr will have proper access to FS
		if fsGroup != nil {
			if containers[i].SecurityContext == nil {
				containers[i].SecurityContext = &corev1.SecurityContext{}
			}
			if containers[i].SecurityContext.RunAsGroup == nil {
				containers[i].SecurityContext.RunAsGroup = fsGroup
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
	AnnotateControlledBy(&podSpec, resourceRoot.Template(), resourceId.Template())
	podSpec.Spec.InitContainers = containers[:len(containers)-1]
	podSpec.Spec.Containers = containers[len(containers)-1:]

	// Build job spec
	jobSpec := batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: batchv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        resourceId.Template(),
			Annotations: jobConfig.Annotations,
			Labels:      jobConfig.Labels,
			Namespace:   jobConfig.Namespace,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:          common.Ptr(int32(0)),
			ActiveDeadlineSeconds: jobConfig.ActiveDeadlineSeconds,
		},
	}
	AnnotateControlledBy(&jobSpec, resourceRoot.Template(), resourceId.Template())
	err = expressions.FinalizeForce(&jobSpec, machines...)
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

	// Build running instructions
	// TODO: Get rid of the unnecessary ContainerConfig parts
	actionGroupsSerialized, _ := json.Marshal(actionGroups)
	podAnnotations := make(map[string]string)
	maps.Copy(podAnnotations, jobSpec.Spec.Template.Annotations)
	maps.Copy(podAnnotations, map[string]string{
		constants.SpecAnnotationName: string(actionGroupsSerialized),
	})
	jobSpec.Spec.Template.Annotations = podAnnotations

	fmt.Println(ui.Cyan(string(actionGroupsSerialized)))
	jobSpecSerialized, _ := common.SerializeCRD(jobSpec, common.SerializeOptions{})
	fmt.Println(ui.Green(string(jobSpecSerialized)))

	// Build bundle
	bundle = &Bundle{
		ConfigMaps: configMaps,
		Secrets:    secrets,
		Job:        jobSpec,
		Signature:  sig,
	}
	return bundle, nil
}
