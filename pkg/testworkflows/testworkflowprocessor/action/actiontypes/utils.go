package actiontypes

import (
	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
)

type ActionList []Action

func NewActionList() ActionList {
	return nil
}

func (a ActionList) Setup(copyInit, copyBinaries bool) ActionList {
	return append(a, Action{Setup: &lite.ActionSetup{CopyInit: copyInit, CopyBinaries: copyBinaries}})
}

func (a ActionList) Declare(ref string, condition string, parents ...string) ActionList {
	return append(a, Action{Declare: &lite.ActionDeclare{Ref: ref, Condition: condition, Parents: parents}})
}

func (a ActionList) Start(ref string) ActionList {
	return append(a, Action{Start: &ref})
}

func (a ActionList) End(ref string) ActionList {
	return append(a, Action{End: &ref})
}

func (a ActionList) Pause(ref string) ActionList {
	return append(a, Action{Pause: &lite.ActionPause{Ref: ref}})
}

func (a ActionList) CurrentStatus(expression string) ActionList {
	return append(a, Action{CurrentStatus: &expression})
}

func (a ActionList) Result(ref, expression string) ActionList {
	return append(a, Action{Result: &lite.ActionResult{Ref: ref, Value: expression}})
}

func (a ActionList) Execute(ref string, negative bool) ActionList {
	return append(a, Action{Execute: &lite.ActionExecute{Ref: ref, Negative: negative}})
}

func (a ActionList) MutateContainer(ref string, config testworkflowsv1.ContainerConfig) ActionList {
	return append(a, Action{Container: &ActionContainer{Ref: ref, Config: config}})
}

type ActionGroups []ActionList

func (a ActionGroups) Append(fn func(list ActionList) ActionList) ActionGroups {
	return append(a, fn(NewActionList()))
}

func NewActionGroups() ActionGroups {
	return nil
}
