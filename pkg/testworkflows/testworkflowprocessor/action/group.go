package action

import (
	"bytes"
	"encoding/json"
	"slices"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

// TODO: Optimize
func isCompatibleContainerConfig(c1, c2 *testworkflowsv1.ContainerConfig) bool {
	// Clean the safe parts of the container configs
	c1 = c1.DeepCopy()
	c2 = c2.DeepCopy()
	c1.Env = nil
	c1.EnvFrom = nil
	c1.WorkingDir = nil
	c1.Command = nil
	c1.Args = nil
	c2.Env = nil
	c2.EnvFrom = nil
	c2.WorkingDir = nil
	c2.Command = nil
	c2.Args = nil

	// Verify if the volume mounts are compatible
	for i1 := range c1.VolumeMounts {
		for i2 := range c2.VolumeMounts {
			if c1.VolumeMounts[i1].MountPath != c2.VolumeMounts[i2].MountPath {
				continue
			}
			if c1.VolumeMounts[i1].Name != c2.VolumeMounts[i2].Name || c1.VolumeMounts[i1].SubPath != c2.VolumeMounts[i2].SubPath || c1.VolumeMounts[i1].SubPathExpr != c2.VolumeMounts[i2].SubPathExpr {
				return false
			}
		}
	}
	c1.VolumeMounts = nil
	c2.VolumeMounts = nil

	// Convert to bytes and compare (ignores order)
	v1, err1 := json.Marshal(c1)
	v2, err2 := json.Marshal(c2)
	return err1 == nil && err2 == nil && bytes.Equal(v1, v2)
}

func getContainerConfigs(actions actiontypes.ActionList) (configs []testworkflowsv1.ContainerConfig, pure bool) {
	pure = true
	for i := range actions {
		switch actions[i].Type() {
		case lite.ActionTypeContainerTransition:
			configs = append(configs, actions[i].Container.Config)
		case lite.ActionTypeExecute:
			if !actions[i].Execute.Pure {
				pure = false
			}
		}
	}
	return
}

func Group(actions actiontypes.ActionList, isolatedContainers bool) (groups actiontypes.ActionGroups) {
	// Detect "start" and "execute" instructions
	startIndexes := make([]int, 0)
	startInstructions := make(map[string]int)
	containerInstructions := make(map[string]int)
	executeInstructions := make(map[string]int)
	executeIndexes := make([]int, 0)
	for i := range actions {
		if actions[i].Start != nil {
			startInstructions[*actions[i].Start] = i
			startIndexes = append(startIndexes, i)
		} else if actions[i].Execute != nil {
			executeInstructions[actions[i].Execute.Ref] = i
			executeIndexes = append(executeIndexes, i)
		} else if actions[i].Container != nil {
			containerInstructions[actions[i].Container.Ref] = i
		} else if actions[i].Setup != nil {
			executeIndexes = append(executeIndexes, i)
		}
	}

	// Start from end, to fill as much as it's possible
	slices.Reverse(executeIndexes)
	slices.Reverse(startIndexes)

	// Fast-track when there is only a single instruction to execute
	if len(executeIndexes) <= 1 {
		return actiontypes.ActionGroups{actions}
	}

	// Basic behavior: split based on each execute instruction
	for _, executeIndex := range executeIndexes {
		if actions[executeIndex].Setup != nil {
			groups = append(actiontypes.ActionGroups{actions[executeIndex:]}, groups...)
			actions = actions[:executeIndex]
			continue
		}
		ref := actions[executeIndex].Execute.Ref
		startIndex := startInstructions[ref]
		if containerIndex, ok := containerInstructions[ref]; ok && containerIndex < startIndex {
			startIndex = containerIndex
		}

		groups = append(actiontypes.ActionGroups{actions[startIndex:]}, groups...)
		actions = actions[:startIndex]
	}
	if len(actions) > 0 {
		groups[0] = append(actions, groups[0]...)
	}

	// Do not try merging the containers
	if isolatedContainers {
		return groups
	}

	// Combine multiple operations in a single container if it's possible
merging:
	for i := len(groups) - 2; i >= 0; i-- {
		// Ignore case when there is last group available
		// TODO: it shouldn't be needed, but it is
		if i+1 >= len(groups) {
			continue
		}

		// Analyze consecutive groups
		g1, p1 := getContainerConfigs(groups[i])
		g2, p2 := getContainerConfigs(groups[i+1])

		// The groups are not pure
		if !p1 && !p2 {
			continue merging
		}

		// One of the groups is not executing anything
		if len(g1) == 0 || len(g2) == 0 {
			groups[i] = append(groups[i], groups[i+1]...)
			groups = append(groups[:i], groups[i+1:]...)
			i++
			continue merging
		}

		// The containers are compatible
		for i1 := range g1 {
			for i2 := range g2 {
				// The pure init or toolkit container is used, so it can be copied
				if (g1[i1].Image == constants.DefaultToolkitImage || g1[i1].Image == constants.DefaultInitImage) && p1 {
					continue
				}
				if (g2[i2].Image == constants.DefaultToolkitImage || g2[i2].Image == constants.DefaultInitImage) && p2 {
					continue
				}

				// We are able to combine the containers
				if isCompatibleContainerConfig(&g1[i1], &g2[i2]) {
					continue
				}

				// The groups cannot be merged together
				continue merging
			}
		}

		groups[i+1] = append(groups[i], groups[i+1]...)
		groups = append(groups[:i], groups[i+1:]...)
		i++
	}

	// Fix the auto-merged internal images
	for i := range groups {
		image := groups[i].Image()
		// Re-use /.tktw/bin/sh when the internal image has been merged into different container
		if image != constants.DefaultToolkitImage && image != constants.DefaultInitImage {
			groups[i] = groups[i].RewireCommandDirectory(constants.DefaultInitImage, constants.DefaultInitImageBusyboxBinaryPath, constants.InternalBinPath)
			groups[i] = groups[i].RewireCommandDirectory(constants.DefaultToolkitImage, constants.DefaultInitImageBusyboxBinaryPath, constants.InternalBinPath)
		}
	}

	return groups
}
