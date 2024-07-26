package lite

type LiteActionList []LiteAction

func NewLiteActionList() LiteActionList {
	return nil
}

func (a LiteActionList) Setup(copyInit, copyBinaries bool) LiteActionList {
	return append(a, LiteAction{Setup: &ActionSetup{CopyInit: copyInit, CopyBinaries: copyBinaries}})
}

func (a LiteActionList) Declare(ref string, condition string, parents ...string) LiteActionList {
	return append(a, LiteAction{Declare: &ActionDeclare{Ref: ref, Condition: condition, Parents: parents}})
}

func (a LiteActionList) Start(ref string) LiteActionList {
	return append(a, LiteAction{Start: &ref})
}

func (a LiteActionList) End(ref string) LiteActionList {
	return append(a, LiteAction{End: &ref})
}

func (a LiteActionList) Pause(ref string) LiteActionList {
	return append(a, LiteAction{Pause: &ActionPause{Ref: ref}})
}

func (a LiteActionList) CurrentStatus(expression string) LiteActionList {
	return append(a, LiteAction{CurrentStatus: &expression})
}

func (a LiteActionList) Result(ref, expression string) LiteActionList {
	return append(a, LiteAction{Result: &ActionResult{Ref: ref, Value: expression}})
}

func (a LiteActionList) Execute(ref string, negative bool) LiteActionList {
	return append(a, LiteAction{Execute: &ActionExecute{Ref: ref, Negative: negative}})
}

func (a LiteActionList) MutateContainer(config LiteContainerConfig) LiteActionList {
	return append(a, LiteAction{Container: &LiteActionContainer{Config: config}})
}

type LiteActionGroups []LiteActionList

func (a LiteActionGroups) Append(fn func(list LiteActionList) LiteActionList) LiteActionGroups {
	return append(a, fn(NewLiteActionList()))
}

func NewLiteActionGroups() LiteActionGroups {
	return nil
}
