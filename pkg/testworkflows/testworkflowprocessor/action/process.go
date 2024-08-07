package action

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
	stage2 "github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

func process(currentStatus string, parents []string, stage stage2.Stage, inheritedPure *bool, machines ...expressions.Machine) (actions actiontypes.ActionList, err error) {
	// Store the init status
	actions = append(actions, actiontypes.Action{
		CurrentStatus: common.Ptr(currentStatus),
	})

	// Compute the skip condition
	condition := stage.Condition()
	if condition == "" || condition == "null" {
		condition = "passed"
	}
	actions = append(actions, actiontypes.Action{
		Declare: &lite.ActionDeclare{Ref: stage.Ref(), Condition: condition, Parents: parents},
	})

	// Configure the container for action
	var containerConfig stage2.Container
	if group, ok := stage.(stage2.GroupStage); ok {
		containerConfig = group.ContainerDefaults()
	} else {
		containerConfig = stage.(stage2.ContainerStage).Container()
	}
	if containerConfig != nil {
		c := containerConfig.Detach()
		err = c.Resolve(machines...)
		if err != nil {
			return nil, err
		}
		actions = append(actions, actiontypes.Action{
			Container: &actiontypes.ActionContainer{Ref: stage.Ref(), Config: c.ToContainerConfig()},
		})
	}

	// Mark the current operation as started
	actions = append(actions, actiontypes.Action{
		Start: common.Ptr(stage.Ref()),
	})

	// Store the timeout information
	if stage.Timeout() != "" {
		actions = append(actions, actiontypes.Action{
			Timeout: &lite.ActionTimeout{Ref: stage.Ref(), Timeout: stage.Timeout()},
		})
	}

	// Store the retry condition
	if stage.RetryPolicy().Count != 0 {
		actions = append(actions, actiontypes.Action{
			Retry: &lite.ActionRetry{Ref: stage.Ref(), Count: stage.RetryPolicy().Count, Until: stage.RetryPolicy().Until},
		})
	}

	// Handle pause
	if stage.Paused() {
		actions = append(actions, actiontypes.Action{
			Pause: &lite.ActionPause{Ref: stage.Ref()},
		})
	}

	// Handle executable action
	if exec, ok := stage.(stage2.ContainerStage); ok {
		toolkit := exec.IsToolkit()
		pure := exec.Pure()
		if !toolkit && !pure && inheritedPure != nil {
			pure = *inheritedPure
		}
		actions = append(actions, actiontypes.Action{
			Execute: &lite.ActionExecute{
				Ref:      exec.Ref(),
				Negative: exec.Negative(),
				Toolkit:  toolkit,
				Pure:     pure,
			},
		})
	}

	// Handle group
	if group, ok := stage.(stage2.GroupStage); ok {
		// Build initial status for children
		if currentStatus == "true" {
			currentStatus = stage.Ref()
		} else {
			currentStatus = fmt.Sprintf("%s && %s", stage.Ref(), currentStatus)
		}
		parents = append(parents, group.Ref())

		// Adjust the inherited purity
		if group.Pure() != nil {
			inheritedPure = group.Pure()
		}

		// Handle children
		refs := make([]string, 0)
		for _, ch := range group.Children() {
			sub, err := process(currentStatus, parents, ch, inheritedPure, machines...)
			if err != nil {
				return nil, errors.Wrap(err, "processing group children")
			}
			if !ch.Optional() {
				currentStatus = fmt.Sprintf("%s && %s", ch.Ref(), currentStatus)
				refs = append(refs, ch.Ref())
			}
			actions = append(actions, sub...)
		}

		// Handle results
		result := "true"
		if group.Negative() {
			result = "false"
		}
		if len(refs) > 0 {
			if group.Negative() {
				result = strings.Join(common.MapSlice(refs, func(ref string) string {
					return "!" + ref
				}), "||")
			} else {
				result = strings.Join(refs, "&&")
			}
		}
		actions = append(actions, actiontypes.Action{Result: &lite.ActionResult{Ref: group.Ref(), Value: result}})
	}

	// Mark the current operation as finished
	actions = append(actions, actiontypes.Action{
		End: common.Ptr(stage.Ref()),
	})

	return
}

func Process(root stage2.Stage, inheritedPure *bool, machines ...expressions.Machine) (actiontypes.ActionList, error) {
	actions, err := process("true", nil, root, inheritedPure, machines...)
	if err != nil {
		return nil, err
	}
	actions = append([]actiontypes.Action{{Start: common.Ptr("")}}, actions...)
	actions = append(actions, actiontypes.Action{Result: &lite.ActionResult{Ref: "", Value: root.Ref()}}, actiontypes.Action{End: common.Ptr("")})

	// Optimize until simplest list of operations
	for {
		prevLength := len(actions)
		actions, err = optimize(actions)

		// Continue until final optimizations are applied
		if err == nil && len(actions) != prevLength {
			continue
		}

		// Sort for easier reading
		sort(actions)

		return actions, errors.Wrap(err, "processing operations")
	}
}
