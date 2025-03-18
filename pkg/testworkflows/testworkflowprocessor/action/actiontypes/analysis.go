package actiontypes

import (
	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
)

// List

func (a ActionList) GetLastRef() string {
	for i := len(a) - 1; i >= 0; i-- {
		switch a[i].Type() {
		case lite.ActionTypeStart:
			return *a[i].Start
		case lite.ActionTypeSetup:
			return constants.InitStepName
		}
	}
	return ""
}

func (a ActionList) Refs() map[string]struct{} {
	refs := make(map[string]struct{})
	for i := range a {
		if a[i].Result != nil {
			refs[a[i].Result.Ref] = struct{}{}
		} else if a[i].Execute != nil {
			refs[a[i].Execute.Ref] = struct{}{}
		} else if a[i].End != nil {
			refs[*a[i].End] = struct{}{}
		}
	}
	return refs
}

func (a ActionList) ExecutableRefs() map[string]struct{} {
	refs := make(map[string]struct{})
	for i := range a {
		if a[i].Execute != nil {
			refs[a[i].Execute.Ref] = struct{}{}
		}
	}
	return refs
}

func (a ActionList) SkippedRefs() map[string]struct{} {
	skipped := make(map[string]struct{})
	for i := range a {
		if a[i].Declare != nil {
			v, err := expressions.EvalExpressionPartial(a[i].Declare.Condition)
			if err == nil && v.Static() != nil {
				b, err := v.Static().BoolValue()
				if err == nil && !b {
					skipped[a[i].Declare.Ref] = struct{}{}
				}
			}
		}
	}
	return skipped
}

func (a ActionList) Results() (map[string]expressions.Expression, error) {
	results := make(map[string]expressions.Expression)
	for i := range a {
		if a[i].Result != nil {
			var err error
			results[a[i].Result.Ref], err = expressions.EvalExpressionPartial(a[i].Result.Value)
			if err != nil {
				return results, err
			}
		}
	}
	return results, nil
}

func (a ActionList) Conditions() (map[string]expressions.Expression, error) {
	conditions := make(map[string]expressions.Expression)
	for i := range a {
		if a[i].Declare != nil {
			var err error
			conditions[a[i].Declare.Ref], err = expressions.EvalExpressionPartial(a[i].Declare.Condition)
			if err != nil {
				return conditions, err
			}
		}
	}
	return conditions, nil
}

// Group

func (a ActionGroups) GetLastRef() (ref string) {
	for i := len(a) - 1; i >= 0; i-- {

		for j := len(a[i]) - 1; j >= 0; j-- {
			ref = a[i].GetLastRef()
			if ref != "" {
				return
			}
		}
	}
	return
}
