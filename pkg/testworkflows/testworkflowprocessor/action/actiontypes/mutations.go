package actiontypes

import (
	"fmt"
	"maps"
	"reflect"
	"regexp"
	"strings"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
)

// List

func (a ActionList) DeleteEmptyContainerMutations() ActionList {
	for i := 0; i < len(a); i++ {
		if a[i].Container != nil && reflect.ValueOf(a[i].Container.Config).IsZero() {
			a = append(a[0:i], a[i+1:]...)
			i--
		}
	}
	return a
}

func (a ActionList) Skip(refs map[string]struct{}) ActionList {
	skipped := make(map[string]struct{})
	maps.Copy(skipped, refs)

	// Skip children too
	for i := range a {
		if a[i].Declare != nil {
			for j := range a[i].Declare.Parents {
				if _, ok := skipped[a[i].Declare.Parents[j]]; ok {
					skipped[a[i].Declare.Ref] = struct{}{}
				}
			}
			if _, ok := skipped[a[i].Declare.Ref]; ok {
				a[i].Declare.Condition = "false"
				a[i].Declare.Parents = nil
			}
		}
	}

	// Avoid executing skipped steps (Execute, Timeout, Retry, Result & End)
	for i := 0; i < len(a); i++ {
		if a[i].Execute != nil {
			if _, ok := skipped[a[i].Execute.Ref]; ok {
				a = append(a[:i], a[i+1:]...)
				i--
			}
		}
		if a[i].Result != nil {
			if _, ok := skipped[a[i].Result.Ref]; ok {
				a = append(a[:i], a[i+1:]...)
				i--
			}
		}
		if a[i].Timeout != nil {
			if _, ok := skipped[a[i].Timeout.Ref]; ok {
				a = append(a[:i], a[i+1:]...)
				i--
			}
		}
		if a[i].Retry != nil {
			if _, ok := skipped[a[i].Retry.Ref]; ok {
				a = append(a[:i], a[i+1:]...)
				i--
			}
		}
		if a[i].Pause != nil {
			if _, ok := skipped[a[i].Pause.Ref]; ok {
				a = append(a[:i], a[i+1:]...)
				i--
			}
		}
		if a[i].Container != nil {
			if _, ok := skipped[a[i].Container.Ref]; ok {
				a = append(a[:i], a[i+1:]...)
				i--
			}
		}
	}

	// Get rid of skipped steps from initial statuses and results
	skipMachine := expressions.NewMachine().
		RegisterAccessor(func(name string) (interface{}, bool) {
			if _, ok := skipped[name]; ok {
				return true, true
			}
			return nil, false
		})
	for i := range a {
		if a[i].CurrentStatus != nil {
			a[i].CurrentStatus = common.Ptr(simplifyExpression(*a[i].CurrentStatus, skipMachine))
		}
		if a[i].Result != nil {
			a[i].Result.Value = simplifyExpression(a[i].Result.Value, skipMachine)
		}
	}

	return a
}

func (a ActionList) SimplifyIntermediateStatuses(currentStatus expressions.Expression) (ActionList, error) {
	// Get all requirements
	refs := a.Refs()
	skipped := a.SkippedRefs()
	results, err := a.Results()
	if err != nil {
		return nil, err
	}
	conditions, err := a.Conditions()
	if err != nil {
		return nil, err
	}

	// Build current state
	executed := make(map[string]struct{})
	machine := expressions.NewMachine().RegisterAccessor(func(name string) (interface{}, bool) {
		if name == "never" {
			return false, true
		} else if name == "always" {
			return true, true
		} else if name == "passed" || name == "success" {
			return currentStatus, true
		} else if name == "failed" || name == "error" {
			return expressions.MustCompile("!passed"), true
		} else if _, ok := skipped[name]; ok {
			return true, true
		} else if v, ok := results[name]; ok {
			// Ignore steps that didn't execute yet
			if _, ok := executed[name]; !ok {
				return true, true
			}

			// Do not go deeper if the result is not determined yet
			if v.Static() == nil {
				return nil, false
			}
			c, ok2 := conditions[name]
			if ok2 {
				return expressions.MustCompile(fmt.Sprintf(`(%s) && (%s)`, c.String(), v.String())), true
			}
			return v, true
		} else if _, ok := refs[name]; ok {
			// Ignore steps that didn't execute yet
			if _, ok := executed[name]; !ok {
				return true, true
			}
			return nil, false
		}
		return nil, false
	})

	for i := range a {
		// Update current status
		if a[i].CurrentStatus != nil {
			var err error
			currentStatus, err = expressions.Compile(*a[i].CurrentStatus)
			if err != nil {
				return nil, err
			}
		}

		// Mark step as executed
		if a[i].Execute != nil {
			executed[a[i].Execute.Ref] = struct{}{}
		} else if a[i].End != nil {
			executed[*a[i].End] = struct{}{}
		}

		// Simplify the condition
		if a[i].Declare != nil {
			a[i].Declare.Condition = simplifyExpression(a[i].Declare.Condition, machine)
			conditions[a[i].Declare.Ref] = expressions.MustCompile(a[i].Declare.Condition)
			for _, parentRef := range a[i].Declare.Parents {
				if _, ok := skipped[parentRef]; ok {
					a[i].Declare.Condition = "false"
					break
				}
			}
		}
	}

	return a, nil
}

func (a ActionList) CastRefStatusToBool() ActionList {
	refs := a.Refs()

	// Wrap all the references with boolean function, and simplify values
	refReplacements := make(map[string]string)
	refResults := make(map[string]string)
	wrapStartRef := expressions.NewMachine().RegisterAccessor(func(name string) (interface{}, bool) {
		if _, ok := refs[name]; !ok {
			return nil, false
		}
		if _, ok := refReplacements[name]; !ok {
			refReplacements[name] = fmt.Sprintf("_WREF_%s_", name)
			refResults[refReplacements[name]] = fmt.Sprintf("bool(%s)", name)
		}
		return expressions.MustCompile(refReplacements[name]), true
	})
	wrapEndRef := expressions.NewMachine().RegisterAccessor(func(name string) (interface{}, bool) {
		if result, ok := refResults[name]; ok {
			return expressions.MustCompile(result), true
		}
		return nil, false
	})
	for i := range a {
		if a[i].CurrentStatus != nil {
			a[i].CurrentStatus = common.Ptr(simplifyExpression(*a[i].CurrentStatus, wrapStartRef))
			a[i].CurrentStatus = common.Ptr(simplifyExpression(*a[i].CurrentStatus, wrapEndRef))
		}
		if a[i].Declare != nil {
			a[i].Declare.Condition = simplifyExpression(a[i].Declare.Condition, wrapStartRef)
			a[i].Declare.Condition = simplifyExpression(a[i].Declare.Condition, wrapEndRef)
		}
		if a[i].Result != nil {
			a[i].Result.Value = simplifyExpression(a[i].Result.Value, wrapStartRef)
			a[i].Result.Value = simplifyExpression(a[i].Result.Value, wrapEndRef)
		}
	}
	return a
}

func (a ActionList) UncastRefStatusFromBool() ActionList {
	refs := a.Refs()

	// Avoid unnecessary casting to boolean
	uncastRegex := regexp.MustCompile(`bool\([^)]+\)`)
	uncastBoolRefs := func(expr string) string {
		return uncastRegex.ReplaceAllStringFunc(expr, func(s string) string {
			ref := s[5 : len(s)-1]
			if _, ok := refs[ref]; ok {
				return ref
			}
			return s
		})
	}
	for i := range a {
		if a[i].CurrentStatus != nil {
			a[i].CurrentStatus = common.Ptr(uncastBoolRefs(*a[i].CurrentStatus))
		}
		if a[i].Declare != nil {
			a[i].Declare.Condition = uncastBoolRefs(a[i].Declare.Condition)
		}
		if a[i].Result != nil {
			a[i].Result.Value = uncastBoolRefs(a[i].Result.Value)
		}
	}
	return a
}

func (a ActionList) RewireCommandDirectory(imageName string, src, dest string) ActionList {
	for i := range a {
		if a[i].Type() != lite.ActionTypeContainerTransition || a[i].Container.Config.Image != imageName {
			continue
		}
		if a[i].Container.Config.Command != nil && len(*a[i].Container.Config.Command) > 0 && strings.HasPrefix((*a[i].Container.Config.Command)[0], src+"/") {
			(*a[i].Container.Config.Command)[0] = dest + (*a[i].Container.Config.Command)[0][len(src):]
		}
	}
	return a
}

func simplifyExpression(expr string, machines ...expressions.Machine) string {
	v, err := expressions.EvalExpressionPartial(expr, machines...)
	if err == nil {
		return v.String()
	}
	return expr
}
