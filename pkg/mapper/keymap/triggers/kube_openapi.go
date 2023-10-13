package triggers

import (
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/keymap/triggers"
)

func MapTestTriggerKeyMapToAPI(km *triggers.KeyMap) *testkube.TestTriggerKeyMap {
	return &testkube.TestTriggerKeyMap{
		Resources:           km.Resources,
		Actions:             km.Actions,
		Executions:          km.Executions,
		Events:              km.Events,
		Conditions:          km.Conditions,
		ConcurrencyPolicies: km.ConcurrencyPolicies,
	}
}
