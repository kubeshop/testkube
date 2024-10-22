package actiontypes

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
)

func EnvName(group string, computed bool, sensitive bool, name string) string {
	suffix := ""
	if computed {
		suffix = "C"
	}
	if sensitive {
		suffix += "S"
	}
	return fmt.Sprintf("_%s%s_%s", group, suffix, name)
}

func EnvVar(group string, computed, sensitive bool, name, value string) corev1.EnvVar {
	return corev1.EnvVar{
		Name:  EnvName(group, computed, sensitive, name),
		Value: value,
	}
}

func EnvVarFrom(group string, computed, sensitive bool, name string, value corev1.EnvVarSource) corev1.EnvVar {
	return corev1.EnvVar{
		Name:      EnvName(group, computed, sensitive, name),
		ValueFrom: &value,
	}
}

type ActionList []Action

func NewActionList() ActionList {
	return nil
}

func (a ActionList) Setup(copyInit, copyToolkit, copyBinaries bool) ActionList {
	return append(a, Action{Setup: &lite.ActionSetup{CopyInit: copyInit, CopyToolkit: copyToolkit, CopyBinaries: copyBinaries}})
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

func (a ActionList) Execute(ref string, negative, pure bool) ActionList {
	return append(a, Action{Execute: &lite.ActionExecute{Ref: ref, Negative: negative, Pure: pure}})
}

func (a ActionList) MutateContainer(ref string, config testworkflowsv1.ContainerConfig) ActionList {
	return append(a, Action{Container: &ActionContainer{Ref: ref, Config: config}})
}

func (a ActionList) Image() string {
	for i := range a {
		if a[i].Container != nil {
			return a[i].Container.Config.Image
		}
	}
	return ""
}

type ActionGroups []ActionList

func (a ActionGroups) Append(fn func(list ActionList) ActionList) ActionGroups {
	return append(a, fn(NewActionList()))
}

func NewActionGroups() ActionGroups {
	return nil
}
