package lite

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

func EnvName(group string, computed bool, name string) string {
	suffix := ""
	if computed {
		suffix = "C"
	}
	return fmt.Sprintf("_%s%s_%s", group, suffix, name)
}

func EnvVar(group string, computed bool, name, value string) corev1.EnvVar {
	return corev1.EnvVar{
		Name:  EnvName(group, computed, name),
		Value: value,
	}
}

func EnvVarFrom(group string, computed bool, name string, value corev1.EnvVarSource) corev1.EnvVar {
	return corev1.EnvVar{
		Name:      EnvName(group, computed, name),
		ValueFrom: &value,
	}
}

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
