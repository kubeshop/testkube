package action

import (
	"fmt"
	"reflect"
	"regexp"

	"k8s.io/apimachinery/pkg/util/rand"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes"
)

func optimize(actions []actiontypes.Action) ([]actiontypes.Action, error) {
	// Detect all the step references
	refs := make(map[string]struct{})
	executableRefs := make(map[string]struct{})
	for i := range actions {
		if actions[i].Result != nil {
			refs[actions[i].Result.Ref] = struct{}{}
		}

		if actions[i].Execute != nil {
			refs[actions[i].Execute.Ref] = struct{}{}
			executableRefs[actions[i].Execute.Ref] = struct{}{}
		}
	}
	//delete(refs, "")

	// Delete empty `container` declarations
	for i := 0; i < len(actions); i++ {
		if actions[i].Container != nil && reflect.ValueOf(actions[i].Container.Config).IsZero() {
			actions = append(actions[0:i], actions[i+1:]...)
			i--
		}
	}

	// Wrap all the references with boolean function, and simplify values
	refReplacements := make(map[string]string)
	refResults := make(map[string]string)
	wrapStartRef := expressions.NewMachine().RegisterAccessor(func(name string) (interface{}, bool) {
		if _, ok := executableRefs[name]; !ok {
			return nil, false
		}
		if _, ok := refReplacements[name]; !ok {
			hashStart := rand.String(10)
			hashEnd := rand.String(10)
			refReplacements[name] = fmt.Sprintf("_%s_%s_%s_", hashStart, name, hashEnd)
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
	for i := range actions {
		if actions[i].CurrentStatus != nil {
			actions[i].CurrentStatus = common.Ptr(simplifyExpression(*actions[i].CurrentStatus, wrapStartRef))
			actions[i].CurrentStatus = common.Ptr(simplifyExpression(*actions[i].CurrentStatus, wrapEndRef))
		}
		if actions[i].Declare != nil {
			actions[i].Declare.Condition = simplifyExpression(actions[i].Declare.Condition, wrapStartRef)
			actions[i].Declare.Condition = simplifyExpression(actions[i].Declare.Condition, wrapEndRef)
		}
		if actions[i].Result != nil {
			actions[i].Result.Value = simplifyExpression(actions[i].Result.Value, wrapStartRef)
			actions[i].Result.Value = simplifyExpression(actions[i].Result.Value, wrapEndRef)
		}
	}

	// Detect immediately skipped steps
	skipped := make(map[string]struct{})
	for i := range actions {
		if actions[i].Declare != nil {
			v, err := expressions.EvalExpressionPartial(actions[i].Declare.Condition)
			if err == nil && v.Static() != nil {
				b, err := v.Static().BoolValue()
				if err == nil && !b {
					skipped[actions[i].Declare.Ref] = struct{}{}
				}
			}
		}
	}

	// List all the results
	results := make(map[string]expressions.Expression)
	conditions := make(map[string]expressions.Expression)
	for i := range actions {
		if actions[i].Result != nil {
			var err error
			refs[actions[i].Result.Ref] = struct{}{}
			results[actions[i].Result.Ref], err = expressions.EvalExpressionPartial(actions[i].Result.Value)
			if err != nil {
				return nil, err
			}
		}

		if actions[i].Declare != nil {
			var err error
			conditions[actions[i].Declare.Ref], err = expressions.EvalExpressionPartial(actions[i].Declare.Condition)
			if err != nil {
				return nil, err
			}
		}

		if actions[i].Execute != nil {
			refs[actions[i].Execute.Ref] = struct{}{}
		}
	}

	// Pre-resolve conditions
	currentStatus := expressions.MustCompile("true")
	executed := make(map[string]struct{})
	for i := range actions {
		// Update current status
		if actions[i].CurrentStatus != nil {
			var err error
			currentStatus, err = expressions.Compile(*actions[i].CurrentStatus)
			if err != nil {
				return nil, err
			}
		}

		// Mark step as executed
		if actions[i].Execute != nil {
			executed[actions[i].Execute.Ref] = struct{}{}
		}

		// Simplify the condition
		if actions[i].Declare != nil {
			// TODO: Handle `never` and other aliases
			machine := expressions.NewMachine().RegisterAccessor(func(name string) (interface{}, bool) {
				if name == "passed" || name == "success" {
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
			actions[i].Declare.Condition = simplifyExpression(actions[i].Declare.Condition, machine)
			for _, parentRef := range actions[i].Declare.Parents {
				if _, ok := skipped[parentRef]; ok {
					actions[i].Declare.Condition = "false"
					break
				}
			}
		}
	}

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
	for i := range actions {
		if actions[i].CurrentStatus != nil {
			actions[i].CurrentStatus = common.Ptr(uncastBoolRefs(*actions[i].CurrentStatus))
		}
		if actions[i].Declare != nil {
			actions[i].Declare.Condition = uncastBoolRefs(actions[i].Declare.Condition)
		}
		if actions[i].Result != nil {
			actions[i].Result.Value = uncastBoolRefs(actions[i].Result.Value)
		}
	}

	//// TODO: Delete empty conditions
	//for i := 0; i < len(actions); i++ {
	//	if actions[i].Declare != nil && actions[i].Declare.Condition == "true" {
	//		actions = append(actions[:i], actions[i+1:]...)
	//		i--
	//	}
	//}

	// Detect immediately skipped steps
	skipped = make(map[string]struct{})
	for i := range actions {
		if actions[i].Declare != nil {
			v, err := expressions.EvalExpressionPartial(actions[i].Declare.Condition)
			if err == nil && v.Static() != nil {
				b, err := v.Static().BoolValue()
				if err == nil && !b {
					skipped[actions[i].Declare.Ref] = struct{}{}
				}
			}
		}
	}

	// Avoid executing skipped steps (Execute, Timeout, Retry, Result & End)
	for i := 0; i < len(actions); i++ {
		if actions[i].Execute != nil {
			if _, ok := skipped[actions[i].Execute.Ref]; ok {
				actions = append(actions[:i], actions[i+1:]...)
				i--
			}
		}
		if actions[i].End != nil {
			if _, ok := skipped[*actions[i].End]; ok {
				actions = append(actions[:i], actions[i+1:]...)
				i--
			}
		}
		if actions[i].Result != nil {
			if _, ok := skipped[actions[i].Result.Ref]; ok {
				actions = append(actions[:i], actions[i+1:]...)
				i--
			}
		}
		if actions[i].Timeout != nil {
			if _, ok := skipped[actions[i].Timeout.Ref]; ok {
				actions = append(actions[:i], actions[i+1:]...)
				i--
			}
		}
		if actions[i].Retry != nil {
			if _, ok := skipped[actions[i].Retry.Ref]; ok {
				actions = append(actions[:i], actions[i+1:]...)
				i--
			}
		}
		if actions[i].Pause != nil {
			if _, ok := skipped[actions[i].Pause.Ref]; ok {
				actions = append(actions[:i], actions[i+1:]...)
				i--
			}
		}
		if actions[i].Container != nil {
			if _, ok := skipped[actions[i].Container.Ref]; ok {
				actions = append(actions[:i], actions[i+1:]...)
				i--
			}
		}
	}

	// Ignore parents for already statically skipped conditions
	for i := range actions {
		if actions[i].Declare != nil {
			if _, ok := skipped[actions[i].Declare.Ref]; ok {
				actions[i].Declare.Parents = nil
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
	for i := range actions {
		if actions[i].CurrentStatus != nil {
			actions[i].CurrentStatus = common.Ptr(simplifyExpression(*actions[i].CurrentStatus, skipMachine))
		}
		if actions[i].Result != nil {
			actions[i].Result.Value = simplifyExpression(actions[i].Result.Value, skipMachine)
		}
	}

	//// TODO
	//// Avoid unused consecutive initial statuses
	//lastIndex := -1
	//wasRequired := true
	//for i := 0; i < len(actions); i++ {
	//	wasRequired = wasRequired || actions[i].End != nil || actions[i].Retry != nil || actions[i].Pause != nil
	//	if actions[i].CurrentStatus != nil {
	//		if !wasRequired {
	//			actions = append(actions[:lastIndex], actions[lastIndex+1:]...)
	//			i--
	//		}
	//		lastIndex = i
	//		wasRequired = false
	//	}
	//}

	return actions, nil
}
