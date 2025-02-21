package action

import (
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

func Finalize(groups actiontypes.ActionGroups, isolatedContainers bool) actiontypes.ActionGroups {
	if len(groups) == 0 {
		return actiontypes.ActionGroups{{{Setup: &lite.ActionSetup{
			CopyInit:     false,
			CopyToolkit:  false,
			CopyBinaries: false,
		}}}}
	}

	// Determine if the Init Process should be copied
	copyInit := false
	copyBinaries := false
	for i := range groups {
		for j := range groups[i] {
			if groups[i][j].Type() == lite.ActionTypeContainerTransition {
				if groups[i][j].Container.Config.Image != constants.DefaultInitImage && groups[i][j].Container.Config.Image != constants.DefaultToolkitImage {
					copyInit = true
					copyBinaries = true
				}
			}
		}
	}

	// Determine if the Toolkit should be copied
	copyToolkit := false
	for i := range groups {
		hadToolkit := false
		hadOther := false
		for j := range groups[i] {
			if groups[i][j].Type() == lite.ActionTypeContainerTransition {
				if groups[i][j].Container.Config.Image == constants.DefaultToolkitImage {
					hadToolkit = true
				} else if groups[i][j].Container.Config.Image != constants.DefaultInitImage {
					hadOther = true
				}
			}
		}
		if hadToolkit && hadOther {
			copyToolkit = true
		}
	}

	// Determine if the setup step can be combined with the further group
	canMergeSetup := !isolatedContainers
	maybeCopyToolkit := false
	if canMergeSetup {
		for i := range groups[0] {
			// Ignore non-transition actions
			if groups[0][i].Type() != lite.ActionTypeContainerTransition {
				continue
			}

			// Disallow merging setup step for custom images
			if groups[0][i].Container.Config.Image != constants.DefaultInitImage && groups[0][i].Container.Config.Image != constants.DefaultToolkitImage {
				canMergeSetup = false
				break
			}

			// Allow merging setup step with toolkit image
			if groups[0][i].Container.Config.Image == constants.DefaultToolkitImage {
				maybeCopyToolkit = true
			}
		}
		if maybeCopyToolkit && canMergeSetup {
			copyToolkit = true
		}
	}

	// Avoid copying binaries when all fits single container
	if len(groups) == 1 && canMergeSetup {
		copyToolkit = false
		copyBinaries = false
		copyInit = false
	}

	// Avoid using /.tktw/toolkit when the toolkit is not copied
	if !copyToolkit {
		for i := range groups {
			for j := range groups[i] {
				if groups[i][j].Type() != lite.ActionTypeContainerTransition || groups[i][j].Container.Config.Image != constants.DefaultToolkitImage {
					continue
				}
				if groups[i][j].Container.Config.Command == nil || len(*groups[i][j].Container.Config.Command) == 0 {
					continue
				}
				if (*groups[i][j].Container.Config.Command)[0] == constants.DefaultToolkitPath {
					(*groups[i][j].Container.Config.Command)[0] = "/toolkit"
				}
			}
		}
	}

	// Build the setup action
	setup := actiontypes.ActionList{{Setup: &lite.ActionSetup{
		CopyInit:     copyInit,
		CopyToolkit:  copyToolkit,
		CopyBinaries: copyBinaries,
	}}}

	// Inject into the first group if possible
	if canMergeSetup {
		return append(actiontypes.ActionGroups{append(setup, groups[0]...)}, groups[1:]...)
	}

	// Move non-executable steps from the 2nd group into setup
	for len(groups[0]) > 0 && groups[0][0].Type() != lite.ActionTypeContainerTransition {
		setup = append(setup, groups[0][0])
		groups[0] = groups[0][1:]
	}
	if len(groups[0]) == 0 {
		groups = groups[1:]
	}

	return append(actiontypes.ActionGroups{setup}, groups...)
}
