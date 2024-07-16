package action

import (
	"slices"

	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes"
)

// TODO: Handle Group Stages properly with isolation (to have conditions working perfectly fine, i.e. for isolated image + file() clause)
func Group(actions []actiontypes.Action) (groups [][]actiontypes.Action) {
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
		return [][]actiontypes.Action{actions}
	}

	// Basic behavior: split based on each execute instruction
	for _, executeIndex := range executeIndexes {
		if actions[executeIndex].Setup != nil {
			groups = append([][]actiontypes.Action{actions[executeIndex:]}, groups...)
			actions = actions[:executeIndex]
			continue
		}
		ref := actions[executeIndex].Execute.Ref
		startIndex := startInstructions[ref]
		if containerIndex, ok := containerInstructions[ref]; ok && containerIndex < startIndex {
			startIndex = containerIndex
		}

		//// FIXME: Delete, it's a hack to combine steps with same image into a single container
		//if i != len(executeIndexes)-1 {
		//	prevRef := actions[executeIndex].Execute.Ref
		//	prevContainerIndex, prevOk := containerInstructions[prevRef]
		//	containerIndex, containerOk := containerInstructions[ref]
		//	if !containerOk || (prevOk && actions[prevContainerIndex].Container.Config.Image == actions[containerIndex].Container.Config.Image) {
		//		continue
		//	}
		//}

		groups = append([][]actiontypes.Action{actions[startIndex:]}, groups...)
		actions = actions[:startIndex]
	}
	if len(actions) > 0 {
		groups[0] = append(actions, groups[0]...)
	}

	// TODO: Behavior: allow selected Toolkit actions to be executed in the same container
	// TODO: Behavior: split based on the image used (use all mounts and variables altogether)
	// TODO: Behavior: split based on the image used (isolate variables)

	return groups
}
