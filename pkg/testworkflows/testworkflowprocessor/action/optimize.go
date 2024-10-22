package action

import (
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

func optimize(actions actiontypes.ActionList) (actiontypes.ActionList, error) {
	// Delete empty `container` declarations
	actions = actions.DeleteEmptyContainerMutations()

	// Wrap all the references with boolean function, and simplify values
	actions = actions.CastRefStatusToBool()

	// Pre-resolve conditions
	actions, err := actions.SimplifyIntermediateStatuses(expressions.MustCompile("true"))
	if err != nil {
		return nil, err
	}

	// Avoid unnecessary casting to boolean
	actions = actions.UncastRefStatusFromBool()

	// Detect immediately skipped steps
	skipped := actions.SkippedRefs()

	// Avoid executing skipped steps (Execute, Timeout, Retry, Result & End)
	actions = actions.Skip(skipped)

	// Avoid using /.tktw/bin/sh when it is internal image used, with direct binaries
	actions = actions.RewireCommandDirectory(constants.DefaultInitImage, constants.InternalBinPath, constants.DefaultInitImageBusyboxBinaryPath)
	actions = actions.RewireCommandDirectory(constants.DefaultToolkitImage, constants.InternalBinPath, constants.DefaultInitImageBusyboxBinaryPath)

	return actions, nil
}
