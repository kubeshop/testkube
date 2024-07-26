package actiontypes

import (
	"encoding/json"
	"fmt"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
)

type ActionContainer struct {
	Ref    string                          `json:"r"`
	Config testworkflowsv1.ContainerConfig `json:"c"`
}

type Action struct {
	CurrentStatus *string             `json:"s,omitempty"`
	Start         *string             `json:"S,omitempty"`
	End           *string             `json:"E,omitempty"`
	Setup         *lite.ActionSetup   `json:"_,omitempty"`
	Declare       *lite.ActionDeclare `json:"d,omitempty"`
	Result        *lite.ActionResult  `json:"r,omitempty"`
	Container     *ActionContainer    `json:"c,omitempty"`
	Execute       *lite.ActionExecute `json:"e,omitempty"`
	Timeout       *lite.ActionTimeout `json:"t,omitempty"`
	Pause         *lite.ActionPause   `json:"p,omitempty"`
	Retry         *lite.ActionRetry   `json:"R,omitempty"`
}

func (a *Action) Type() lite.ActionType {
	if a.Declare != nil {
		return lite.ActionTypeDeclare
	} else if a.Pause != nil {
		return lite.ActionTypePause
	} else if a.Result != nil {
		return lite.ActionTypeResult
	} else if a.Timeout != nil {
		return lite.ActionTypeTimeout
	} else if a.Retry != nil {
		return lite.ActionTypeRetry
	} else if a.Container != nil {
		return lite.ActionTypeContainerTransition
	} else if a.CurrentStatus != nil {
		return lite.ActionTypeCurrentStatus
	} else if a.Start != nil {
		return lite.ActionTypeStart
	} else if a.End != nil {
		return lite.ActionTypeEnd
	} else if a.Setup != nil {
		return lite.ActionTypeSetup
	} else if a.Execute != nil {
		return lite.ActionTypeExecute
	}
	v, e := json.Marshal(a)
	panic(fmt.Sprintf("unknown action type: %s, %v", string(v), e))
}
