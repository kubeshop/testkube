package action

import (
	"slices"

	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes"
)

func sort(actions []actiontypes.Action) {
	// Move retry policies to top
	slices.SortStableFunc(actions, func(a actiontypes.Action, b actiontypes.Action) int {
		if (a.Retry == nil) == (b.Retry == nil) {
			return 0
		}
		if a.Retry == nil {
			return 1
		}
		return -1
	})

	// Move timeouts to top
	slices.SortStableFunc(actions, func(a actiontypes.Action, b actiontypes.Action) int {
		if (a.Timeout == nil) == (b.Timeout == nil) {
			return 0
		}
		if a.Timeout == nil {
			return 1
		}
		return -1
	})

	// Move results to top
	slices.SortStableFunc(actions, func(a actiontypes.Action, b actiontypes.Action) int {
		if (a.Result == nil) == (b.Result == nil) {
			return 0
		}
		if a.Result == nil {
			return 1
		}
		return -1
	})

	// Move pause information to top
	slices.SortStableFunc(actions, func(a actiontypes.Action, b actiontypes.Action) int {
		if (a.Pause == nil) == (b.Pause == nil) {
			return 0
		}
		if a.Pause == nil {
			return 1
		}
		return -1
	})

	// Move declarations to top
	slices.SortStableFunc(actions, func(a actiontypes.Action, b actiontypes.Action) int {
		if (a.Declare == nil) == (b.Declare == nil) {
			return 0
		}
		if a.Declare == nil {
			return 1
		}
		return -1
	})

	// Move setup to top
	slices.SortStableFunc(actions, func(a actiontypes.Action, b actiontypes.Action) int {
		if (a.Setup == nil) == (b.Setup == nil) {
			return 0
		}
		if a.Setup == nil {
			return 1
		}
		return -1
	})
}
