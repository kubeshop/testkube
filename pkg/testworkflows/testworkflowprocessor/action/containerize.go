package action

import (
	"fmt"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	constants2 "github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	stage2 "github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

func CreateContainer(groupId int, defaultContainer stage2.Container, actions []actiontypes.Action, usesToolkit bool) (cr corev1.Container, actionsCleanup []actiontypes.Action, err error) {
	actions = slices.Clone(actions)
	actionsCleanup = actions

	// Find the container configurations and executable/setup steps
	var setup *actiontypes.Action
	executable := map[string]bool{}
	toolkit := map[string]bool{}
	containerConfigs := make([]*actiontypes.Action, 0)
	for i := range actions {
		if actions[i].Container != nil {
			containerConfigs = append(containerConfigs, &actions[i])
		} else if actions[i].Setup != nil {
			setup = &actions[i]
		} else if actions[i].Execute != nil {
			executable[actions[i].Execute.Ref] = true
			if actions[i].Execute.Toolkit {
				toolkit[actions[i].Execute.Ref] = true
			}
		}
	}

	// Find the highest priority container configuration
	var bestContainerConfig *actiontypes.Action
	var bestIsToolkit = false
	var bestIsDefaultImage = true
	for i := range containerConfigs {
		if executable[containerConfigs[i].Container.Ref] {
			image := containerConfigs[i].Container.Config.Image
			isDefaultImage := image == "" || image == constants.DefaultInitImage || image == constants.DefaultToolkitImage
			if bestContainerConfig == nil || bestIsToolkit || (bestIsDefaultImage && !isDefaultImage) {
				bestContainerConfig = containerConfigs[i]
				bestIsToolkit = toolkit[bestContainerConfig.Container.Ref]
				bestIsDefaultImage = isDefaultImage
			}
		}
	}
	if bestContainerConfig == nil && len(containerConfigs) > 0 {
		bestContainerConfig = containerConfigs[len(containerConfigs)-1]
	}
	if bestContainerConfig == nil {
		bestContainerConfig = &actiontypes.Action{Container: &actiontypes.ActionContainer{Config: defaultContainer.ToContainerConfig()}}
	}

	// Build the CR base
	cr, _ = defaultContainer.Detach().ToKubernetesTemplate()
	cr.Image = ""
	cr.Env = nil
	cr.EnvFrom = nil
	if len(containerConfigs) > 0 {
		cr, err = stage2.NewContainer().ApplyCR(&bestContainerConfig.Container.Config).ToKubernetesTemplate()
		if err != nil {
			return corev1.Container{}, nil, err
		}

		// Combine environment variables from each execution
		cr.Env = nil
		cr.EnvFrom = nil
		for i := range containerConfigs {
			// TODO: Avoid having multiple copies of the same environment variable
			for _, e := range containerConfigs[i].Container.Config.Env {
				newEnv := *e.DeepCopy()
				computed := strings.Contains(newEnv.Value, "{{")
				sensitive := newEnv.ValueFrom != nil && newEnv.ValueFrom.SecretKeyRef != nil
				newEnv.Name = actiontypes.EnvName(fmt.Sprintf("%d", i), computed, sensitive, e.Name)
				cr.Env = append(cr.Env, newEnv)
			}
			for _, e := range containerConfigs[i].Container.Config.EnvFrom {
				newEnvFrom := *e.DeepCopy()
				sensitive := newEnvFrom.SecretRef != nil
				newEnvFrom.Prefix = actiontypes.EnvName(fmt.Sprintf("%d", i), false, sensitive, e.Prefix)
				cr.EnvFrom = append(cr.EnvFrom, newEnvFrom)
			}
		}

		// Combine the volume mounts
		for i := range containerConfigs {
		loop:
			for _, v := range containerConfigs[i].Container.Config.VolumeMounts {
				for j := range cr.VolumeMounts {
					if cr.VolumeMounts[j].MountPath == v.MountPath {
						continue loop
					}
				}
				cr.VolumeMounts = append(cr.VolumeMounts, v)
			}
		}
	}

	// Set up a default image when not specified
	if cr.Image == "" {
		cr.Image = constants.DefaultInitImage
		cr.ImagePullPolicy = corev1.PullIfNotPresent
	} else if cr.ImagePullPolicy == "" {
		cr.ImagePullPolicy = corev1.PullIfNotPresent
	}

	// Use the Toolkit image instead of Init if it's anyway used
	if usesToolkit && cr.Image == constants.DefaultInitImage {
		cr.Image = constants.DefaultToolkitImage
	}

	// Provide the data required for setup step
	if setup != nil {
		cr.Env = append(cr.Env,
			corev1.EnvVar{Name: fmt.Sprintf("_%s_%s", constants2.EnvGroupDebug, constants2.EnvNodeName), ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"},
			}},
			corev1.EnvVar{Name: fmt.Sprintf("_%s_%s", constants2.EnvGroupDebug, constants2.EnvPodName), ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"},
			}},
			corev1.EnvVar{Name: fmt.Sprintf("_%s_%s", constants2.EnvGroupDebug, constants2.EnvNamespaceName), ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"},
			}},
			corev1.EnvVar{Name: fmt.Sprintf("_%s_%s", constants2.EnvGroupDebug, constants2.EnvServiceAccountName), ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.serviceAccountName"},
			}},
			corev1.EnvVar{Name: fmt.Sprintf("_%s_%s", constants2.EnvGroupActions, constants2.EnvActions), ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: constants.SpecAnnotationFieldPath},
			}},
			corev1.EnvVar{Name: fmt.Sprintf("_%s_%s", constants2.EnvGroupInternal, constants2.EnvInternalConfig), ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: constants.InternalAnnotationFieldPath},
			}})
	}

	// Avoid using /.tktw/init if there is Init Process Image - use /init then
	initPath := constants.DefaultInitPath
	if cr.Image == constants.DefaultInitImage || cr.Image == constants.DefaultToolkitImage {
		initPath = "/init"
	}

	// Point the Init Process to the proper group
	cr.Name = fmt.Sprintf("%d", groupId+1)
	cr.Command = []string{initPath, fmt.Sprintf("%d", groupId)}
	cr.Args = nil

	// Clean up the executions
	for i := range containerConfigs {
		newConfig := testworkflowsv1.ContainerConfig{}
		if executable[containerConfigs[i].Container.Ref] {
			newConfig.Command = containerConfigs[i].Container.Config.Command
			newConfig.Args = containerConfigs[i].Container.Config.Args
		}
		newConfig.WorkingDir = containerConfigs[i].Container.Config.WorkingDir

		containerConfigs[i].Container = &actiontypes.ActionContainer{
			Ref:    containerConfigs[i].Container.Ref,
			Config: newConfig,
		}
	}

	return
}
