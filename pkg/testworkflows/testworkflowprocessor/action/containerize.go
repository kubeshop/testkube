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

// TODO: Disallow bypassing
func CreateContainer(groupId int, defaultContainer stage2.Container, actions []actiontypes.Action) (cr corev1.Container, actionsCleanup []actiontypes.Action, err error) {
	actions = slices.Clone(actions)
	actionsCleanup = actions

	// Find the container configurations and executable/setup steps
	var setup *actiontypes.Action
	executable := map[string]bool{}
	containerConfigs := make([]*actiontypes.Action, 0)
	for i := range actions {
		if actions[i].Container != nil {
			containerConfigs = append(containerConfigs, &actions[i])
		} else if actions[i].Setup != nil {
			setup = &actions[i]
		} else if actions[i].Execute != nil {
			executable[actions[i].Execute.Ref] = true
		}
	}

	// Find the highest priority container configuration
	var bestContainerConfig *actiontypes.Action
	for i := range containerConfigs {
		if executable[containerConfigs[i].Container.Ref] {
			bestContainerConfig = containerConfigs[i]
			break
		}
	}
	if bestContainerConfig == nil && len(containerConfigs) > 0 {
		bestContainerConfig = containerConfigs[len(containerConfigs)-1]
	}

	// Build the cr base
	// TODO: Handle the case when there are multiple exclusive execution configurations
	// TODO: Handle a case when that configuration should join multiple configurations (i.e. envs/volumeMounts)
	if len(containerConfigs) > 0 {
		cr, err = stage2.NewContainer().ApplyCR(&bestContainerConfig.Container.Config).ToKubernetesTemplate()
		if err != nil {
			return corev1.Container{}, nil, err
		}

		// Combine environment variables from each execution
		cr.Env = nil
		cr.EnvFrom = nil
		for i := range containerConfigs {
			for _, e := range containerConfigs[i].Container.Config.Env {
				newEnv := *e.DeepCopy()
				if strings.Contains(newEnv.Value, "{{") {
					newEnv.Name = fmt.Sprintf("_%dC_%s", i, e.Name)
				} else {
					newEnv.Name = fmt.Sprintf("_%d_%s", i, e.Name)
				}
				cr.Env = append(cr.Env, newEnv)
			}
			for _, e := range containerConfigs[i].Container.Config.EnvFrom {
				newEnvFrom := *e.DeepCopy()
				newEnvFrom.Prefix = fmt.Sprintf("_%d_%s", i, e.Prefix)
				cr.EnvFrom = append(cr.EnvFrom, newEnvFrom)
			}
		}
		// TODO: Combine the rest
	}

	// Set up a default image when not specified
	if cr.Image == "" {
		cr.Image = constants.DefaultInitImage
		cr.ImagePullPolicy = corev1.PullIfNotPresent
	}

	// Provide the data required for setup step
	if setup != nil {
		cr.Env = append(cr.Env,
			corev1.EnvVar{Name: fmt.Sprintf("_00_%s", constants2.EnvNodeName), ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"},
			}},
			corev1.EnvVar{Name: fmt.Sprintf("_00_%s", constants2.EnvPodName), ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"},
			}},
			corev1.EnvVar{Name: fmt.Sprintf("_00_%s", constants2.EnvNamespaceName), ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"},
			}},
			corev1.EnvVar{Name: fmt.Sprintf("_00_%s", constants2.EnvServiceAccountName), ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.serviceAccountName"},
			}},
			corev1.EnvVar{Name: fmt.Sprintf("_01_%s", constants2.EnvInstructions), ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: fmt.Sprintf("metadata.annotations['%s']", constants.SpecAnnotationName)},
			}})

		// Apply basic mounts, so there is a state provided
		for _, volumeMount := range defaultContainer.VolumeMounts() {
			if !slices.ContainsFunc(cr.VolumeMounts, func(mount corev1.VolumeMount) bool {
				return mount.Name == volumeMount.Name
			}) {
				cr.VolumeMounts = append(cr.VolumeMounts, volumeMount)
			}
		}
	}

	// TODO: Avoid using /.tktw/init if there is Init Image - use /init then
	initPath := constants.DefaultInitPath
	if cr.Image == constants.DefaultInitImage {
		initPath = "/init"
	}

	// TODO: Avoid using /.tktw/toolkit if there is Toolkit image

	// TODO: Avoid using /.tktw/bin/sh (and other binaries) if there is Init image - use /bin/* then

	// TODO: Copy /init and /toolkit in the Init Container only if there is a need to.
	//       Probably, include Setup step in the action.Actions list, so it can be simplified too into a single container,
	//       and optimized along with others.

	// Point the Init Process to the proper group
	cr.Name = fmt.Sprintf("%d", groupId+1)
	cr.Command = []string{initPath, fmt.Sprintf("%d", groupId)}
	cr.Args = nil

	// Clean up the executions
	for i := range containerConfigs {
		// TODO: Clean it up
		newConfig := testworkflowsv1.ContainerConfig{}
		if executable[containerConfigs[i].Container.Ref] {
			newConfig.Command = containerConfigs[i].Container.Config.Command
			newConfig.Args = containerConfigs[i].Container.Config.Args
		}
		newConfig.WorkingDir = containerConfigs[i].Container.Config.WorkingDir
		// TODO: expose more?

		containerConfigs[i].Container = &actiontypes.ActionContainer{
			Ref:    containerConfigs[i].Container.Ref,
			Config: newConfig,
		}
	}

	return
}
