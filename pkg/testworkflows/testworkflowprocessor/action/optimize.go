package action

import (
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

func optimize(actions actiontypes.ActionList) (actiontypes.ActionList, error) {
	// Delete empty `container` declarations
	actions = actions.DeleteEmptyContainerMutations()

	// Wrap all the references with boolean function, and simplify values
	actions = actions.CastRefStatusToBool()

	// Detect immediately skipped steps
	skipped := actions.SkippedRefs()

	// Pre-resolve conditions
	actions, err := actions.SimplifyIntermediateStatuses(expressions.MustCompile("true"))
	if err != nil {
		return nil, err
	}

	// Avoid unnecessary casting to boolean
	actions = actions.UncastRefStatusFromBool()

	// Detect immediately skipped steps
	skipped = actions.SkippedRefs()

	// Avoid executing skipped steps (Execute, Timeout, Retry, Result & End)
	actions = actions.Skip(skipped)

	// Avoid using /.tktw/bin/sh when it is internal image used, with direct binaries
	actions = actions.RewireCommandDirectory(constants.DefaultInitImage, constants.InternalBinPath, "/tktw-bin")
	actions = actions.RewireCommandDirectory(constants.DefaultToolkitImage, constants.InternalBinPath, "/tktw-bin")

	// Avoid copying init process, toolkit and common binaries, when it is not necessary
	copyInit := false
	hasToolkit := false
	copyBinaries := false
	for i := range actions {
		if actions[i].Type() == lite.ActionTypeContainerTransition {
			if actions[i].Container.Config.Image != constants.DefaultInitImage && actions[i].Container.Config.Image != constants.DefaultToolkitImage {
				copyInit = true
				copyBinaries = true
			}
			if actions[i].Container.Config.Image == constants.DefaultToolkitImage {
				hasToolkit = true
			}
		}
	}
	for i := range actions {
		if actions[i].Type() == lite.ActionTypeSetup {
			actions[i].Setup.CopyInit = copyInit
			actions[i].Setup.CopyToolkit = copyInit && hasToolkit
			actions[i].Setup.CopyBinaries = copyBinaries
		}
	}

	return actions, nil
}
