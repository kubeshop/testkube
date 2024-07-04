package testworkflowprocessor

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	quantity "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

var BypassToolkitCheck = corev1.EnvVar{
	Name:  "TK_TC_SECURITY",
	Value: rand.String(20),
}

func MapResourcesToKubernetesResources(resources *testworkflowsv1.Resources) (corev1.ResourceRequirements, error) {
	result := corev1.ResourceRequirements{}
	if resources != nil {
		if len(resources.Requests) > 0 {
			result.Requests = make(corev1.ResourceList)
		}
		if len(resources.Limits) > 0 {
			result.Limits = make(corev1.ResourceList)
		}
		for k, v := range resources.Requests {
			var err error
			result.Requests[k], err = quantity.ParseQuantity(v.String())
			if err != nil {
				return corev1.ResourceRequirements{}, errors.Wrap(err, "parsing resources")
			}
		}
		for k, v := range resources.Limits {
			var err error
			result.Limits[k], err = quantity.ParseQuantity(v.String())
			if err != nil {
				return corev1.ResourceRequirements{}, errors.Wrap(err, "parsing resources")
			}
		}
	}
	return result, nil
}

func AnnotateControlledBy(obj metav1.Object, rootId, id string) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels[constants.RootResourceIdLabelName] = rootId
	labels[constants.ResourceIdLabelName] = id
	obj.SetLabels(labels)

	// Annotate Pod template in the Job
	if v, ok := obj.(*batchv1.Job); ok {
		AnnotateControlledBy(&v.Spec.Template, rootId, id)
	}
}

func AnnotateGroupId(obj metav1.Object, id string) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels[constants.GroupIdLabelName] = id
	obj.SetLabels(labels)

	// Annotate Pod template in the Job
	if v, ok := obj.(*batchv1.Job); ok {
		AnnotateGroupId(&v.Spec.Template, id)
	}
}

func getRef(stage Stage) string {
	return stage.Ref()
}

func isNotOptional(stage Stage) bool {
	return !stage.Optional()
}

func buildKubernetesContainers(stage Stage, init *initProcess, fsGroup *int64, machines ...expressions.Machine) (containers []corev1.Container, err error) {
	if stage.Paused() {
		init.SetPaused(stage.Paused())
	}
	if stage.Timeout() != "" {
		init.AddTimeout(stage.Timeout(), stage.Ref())
	}
	if stage.Ref() != "" {
		init.AddCondition(stage.Condition(), stage.Ref())
	}
	init.AddRetryPolicy(stage.RetryPolicy(), stage.Ref())

	group, ok := stage.(GroupStage)
	if ok {
		recursiveRefs := common.MapSlice(group.RecursiveChildren(), getRef)
		directRefResults := common.MapSlice(common.FilterSlice(group.Children(), isNotOptional), getRef)

		init.AddCondition(stage.Condition(), recursiveRefs...)

		if group.Negative() {
			// Create virtual layer that will be put down into actual negative step
			init.SetRef(stage.Ref() + ".v")
			init.AddCondition(stage.Condition(), stage.Ref()+".v")
			init.PrependInitialStatus(stage.Ref() + ".v")
			init.AddResult("!"+stage.Ref()+".v", stage.Ref())
		} else if stage.Ref() != "" {
			init.PrependInitialStatus(stage.Ref())
		}

		if group.Optional() {
			init.ResetResults()
		}

		if group.Negative() {
			init.AddResult(strings.Join(directRefResults, "&&"), stage.Ref()+".v")
		} else {
			init.AddResult(strings.Join(directRefResults, "&&"), stage.Ref())
		}

		for i, ch := range group.Children() {
			// Condition should be executed only in the first leaf
			if i == 1 {
				init.ResetCondition().SetPaused(false)
			}
			// Pass down to another group or container
			sub, serr := buildKubernetesContainers(ch, init.Children(ch.Ref()), fsGroup, machines...)
			if serr != nil {
				return nil, fmt.Errorf("%s: %s: resolving children: %s", stage.Ref(), stage.Name(), serr.Error())
			}
			containers = append(containers, sub...)
		}
		return
	}
	c, ok := stage.(ContainerStage)
	if !ok {
		return nil, fmt.Errorf("%s: %s: stage that is neither container nor group", stage.Ref(), stage.Name())
	}
	err = c.Container().Detach().Resolve(machines...)
	if err != nil {
		return nil, fmt.Errorf("%s: %s: resolving container: %s", stage.Ref(), stage.Name(), err.Error())
	}

	cr, err := c.Container().ToKubernetesTemplate()
	if err != nil {
		return nil, fmt.Errorf("%s: %s: building container template: %s", stage.Ref(), stage.Name(), err.Error())
	}
	cr.Name = c.Ref()

	if c.Optional() {
		init.ResetResults()
	}

	bypass := false
	refEnvVar := ""
	for _, e := range cr.Env {
		if e.Name == BypassToolkitCheck.Name && e.Value == BypassToolkitCheck.Value {
			bypass = true
		}
		if e.Name == "TK_REF" {
			refEnvVar = e.Value
		}
	}

	init.
		SetNegative(c.Negative()).
		AddRetryPolicy(c.RetryPolicy(), c.Ref()).
		SetCommand(cr.Command...).
		SetArgs(cr.Args...).
		SetWorkingDir(cr.WorkingDir).
		SetToolkit(bypass || (cr.Image == constants.DefaultToolkitImage && c.Ref() == refEnvVar))

	for _, env := range cr.Env {
		if strings.Contains(env.Value, "{{") {
			init.AddComputedEnvs(env.Name)
		}
	}

	if init.Error() != nil {
		return nil, init.Error()
	}

	cr.Command = init.Command()
	cr.Args = init.Args()
	cr.WorkingDir = ""

	// Ensure the container will have proper access to FS
	if cr.SecurityContext == nil {
		cr.SecurityContext = &corev1.SecurityContext{}
	}
	if cr.SecurityContext.RunAsGroup == nil {
		cr.SecurityContext.RunAsGroup = fsGroup
	}

	containers = []corev1.Container{cr}
	return
}

type ActionResult struct {
	Ref   string `json:"r"`
	Value string `json:"v"`
}

type ActionCondition struct {
	Condition string   `json:"c"`
	Ref       string   `json:"r"`
	Parents   []string `json:"p"`
}

type ActionExecute struct {
	Ref      string                          `json:"r"`
	Parents  []string                        `json:"p,omitempty"`
	Negative bool                            `json:"n,omitempty"`
	Config   testworkflowsv1.ContainerConfig `json:"c"`
}

// TODO: Consider for groups too?
type ActionPause struct {
	Ref string `json:"r"`
}

type ActionTimeout struct {
	Ref     string `json:"r"`
	Timeout string `json:"t"`
}

// TODO: RetryAction as a conditional GoTo back?
type ActionRetry struct {
	Ref   string `json:"r"`
	Count int32  `json:"c,omitempty"`
	Until string `json:"u,omitempty"`
}

type Action struct {
	CurrentStatus *string          `json:"s,omitempty"`
	Start         *string          `json:"S,omitempty"`
	End           *string          `json:"E,omitempty"`
	Condition     *ActionCondition `json:"c,omitempty"`
	Result        *ActionResult    `json:"r,omitempty"`
	Execute       *ActionExecute   `json:"e,omitempty"`
	Timeout       *ActionTimeout   `json:"t,omitempty"`
	Pause         *ActionPause     `json:"p,omitempty"`
	Retry         *ActionRetry     `json:"R,omitempty"`
}

// TODO: tail-recursive
func analyzeOperations(currentStatus string, parents []string, stage Stage, machines ...expressions.Machine) (actions []Action, err error) {
	// Store the init status
	actions = append(actions, Action{
		CurrentStatus: common.Ptr(currentStatus),
	})

	// Compute the skip condition
	if stage.Ref() != "" {
		condition := stage.Condition()
		if condition == "" {
			condition = "passed" // TODO: Think if it should default the condition to "passed"
		}
		actions = append(actions, Action{
			Condition: &ActionCondition{Ref: stage.Ref(), Condition: condition, Parents: parents},
		})
	}

	// Mark the current operation as started
	actions = append(actions, Action{
		Start: common.Ptr(stage.Ref()),
	})

	// Store the timeout information
	if stage.Timeout() != "" {
		actions = append(actions, Action{
			Timeout: &ActionTimeout{Ref: stage.Ref(), Timeout: stage.Timeout()},
		})
	}

	// Store the retry condition
	if stage.RetryPolicy().Count != 0 {
		actions = append(actions, Action{
			Retry: &ActionRetry{Ref: stage.Ref(), Count: stage.RetryPolicy().Count, Until: stage.RetryPolicy().Until},
		})
	}

	// Handle pause
	if stage.Paused() {
		actions = append(actions, Action{
			Pause: &ActionPause{Ref: stage.Ref()},
		})
	}

	// Handle executable action
	if exec, ok := stage.(ContainerStage); ok {
		// Handle execution
		actions = append(actions, Action{
			Execute: &ActionExecute{
				Ref:      exec.Ref(),
				Parents:  parents,
				Negative: exec.Negative(),
				Config:   exec.Container().Detach().ToContainerConfig(),
			},
		})
	}

	// Handle group
	if group, ok := stage.(GroupStage); ok {
		// Build initial status for children
		// TODO: Handle negative
		// TODO: Consider enum value instead of boolean
		if currentStatus == "true" {
			currentStatus = stage.Ref()
		} else {
			currentStatus = fmt.Sprintf("%s && %s", stage.Ref(), currentStatus)
		}
		parents = append(parents, group.Ref())

		// Handle children
		refs := make([]string, 0)
		for _, ch := range group.Children() {
			sub, err := analyzeOperations(currentStatus, parents, ch, machines...)
			if err != nil {
				return nil, err
			}
			if !ch.Optional() {
				currentStatus = fmt.Sprintf("%s && %s", ch.Ref(), currentStatus)
				refs = append(refs, ch.Ref())
			}
			actions = append(actions, sub...)
		}

		// Handle results
		result := "true"
		if group.Negative() {
			result = "false"
		}
		if len(refs) > 0 {
			if group.Negative() {
				result = strings.Join(common.MapSlice(refs, func(ref string) string {
					return "!" + ref
				}), "||")
			} else {
				result = strings.Join(refs, "&&")
			}
		}
		actions = append(actions, Action{Result: &ActionResult{Ref: group.Ref(), Value: result}})
	}

	// Mark the current operation as finished
	actions = append(actions, Action{
		End: common.Ptr(stage.Ref()),
	})

	return
}

func simplifyExpression(expr string, machines ...expressions.Machine) string {
	v, err := expressions.EvalExpressionPartial(expr, machines...)
	if err == nil {
		return v.String()
	}
	return expr
}

func optimizeActions(root Stage, actions []Action) ([]Action, error) {
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
		if actions[i].Condition != nil {
			actions[i].Condition.Condition = simplifyExpression(actions[i].Condition.Condition, wrapStartRef)
			actions[i].Condition.Condition = simplifyExpression(actions[i].Condition.Condition, wrapEndRef)
		}
		if actions[i].Result != nil {
			actions[i].Result.Value = simplifyExpression(actions[i].Result.Value, wrapStartRef)
			actions[i].Result.Value = simplifyExpression(actions[i].Result.Value, wrapEndRef)
		}
	}

	// Detect immediately skipped steps
	skipped := make(map[string]struct{})
	for i := range actions {
		if actions[i].Condition != nil {
			v, err := expressions.EvalExpressionPartial(actions[i].Condition.Condition)
			if err == nil && v.Static() != nil {
				b, err := v.Static().BoolValue()
				if err == nil && !b {
					skipped[actions[i].Condition.Ref] = struct{}{}
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

		if actions[i].Condition != nil {
			var err error
			conditions[actions[i].Condition.Ref], err = expressions.EvalExpressionPartial(actions[i].Condition.Condition)
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
		if actions[i].Condition != nil {
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
			actions[i].Condition.Condition = simplifyExpression(actions[i].Condition.Condition, machine)
			for _, parentRef := range actions[i].Condition.Parents {
				if _, ok := skipped[parentRef]; ok {
					actions[i].Condition.Condition = "false"
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
		if actions[i].Condition != nil {
			actions[i].Condition.Condition = uncastBoolRefs(actions[i].Condition.Condition)
		}
		if actions[i].Result != nil {
			actions[i].Result.Value = uncastBoolRefs(actions[i].Result.Value)
		}
	}

	//// TODO: Delete empty conditions
	//for i := 0; i < len(actions); i++ {
	//	if actions[i].Condition != nil && actions[i].Condition.Condition == "true" {
	//		actions = append(actions[:i], actions[i+1:]...)
	//		i--
	//	}
	//}

	// Detect immediately skipped steps
	skipped = make(map[string]struct{})
	for i := range actions {
		if actions[i].Condition != nil {
			v, err := expressions.EvalExpressionPartial(actions[i].Condition.Condition)
			if err == nil && v.Static() != nil {
				b, err := v.Static().BoolValue()
				if err == nil && !b {
					skipped[actions[i].Condition.Ref] = struct{}{}
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
	}

	// Ignore parents for already statically skipped conditions
	for i := range actions {
		if actions[i].Condition != nil {
			if _, ok := skipped[actions[i].Condition.Ref]; ok {
				actions[i].Condition.Parents = nil
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

func sortActions(actions []Action) {
	// Move retry policies to top
	slices.SortStableFunc(actions, func(a Action, b Action) int {
		if (a.Retry == nil) == (b.Retry == nil) {
			return 0
		}
		if a.Retry == nil {
			return 1
		}
		return -1
	})

	// Move timeouts to top
	slices.SortStableFunc(actions, func(a Action, b Action) int {
		if (a.Timeout == nil) == (b.Timeout == nil) {
			return 0
		}
		if a.Timeout == nil {
			return 1
		}
		return -1
	})

	// Move results to top
	slices.SortStableFunc(actions, func(a Action, b Action) int {
		if (a.Result == nil) == (b.Result == nil) {
			return 0
		}
		if a.Result == nil {
			return 1
		}
		return -1
	})

	// Move pause information to top
	slices.SortStableFunc(actions, func(a Action, b Action) int {
		if (a.Pause == nil) == (b.Pause == nil) {
			return 0
		}
		if a.Pause == nil {
			return 1
		}
		return -1
	})

	// Move conditions to top
	slices.SortStableFunc(actions, func(a Action, b Action) int {
		if (a.Condition == nil) == (b.Condition == nil) {
			return 0
		}
		if a.Condition == nil {
			return 1
		}
		return -1
	})
}

func AnalyzeOperations(root Stage, machines ...expressions.Machine) ([]Action, error) {
	actions, err := analyzeOperations("true", nil, root, machines...)
	if err != nil {
		return nil, err
	}
	actions = append([]Action{{Start: common.Ptr("")}}, actions...)
	actions = append(actions, Action{Result: &ActionResult{Ref: "", Value: root.Ref()}}, Action{End: common.Ptr("")})

	// Optimize until simplest list of operations
	for {
		prevLength := len(actions)
		actions, err = optimizeActions(root, actions)
		if err != nil || len(actions) == prevLength {
			sortActions(actions)
			return actions, err
		}
	}
}

func GroupActions(actions []Action) (groups [][]Action) {
	// Detect "start" and "execute" instructions
	startIndexes := make([]int, 0)
	startInstructions := make(map[string]int)
	executeInstructions := make(map[string]int)
	executeIndexes := make([]int, 0)
	for i := range actions {
		if actions[i].Start != nil {
			startInstructions[*actions[i].Start] = i
			startIndexes = append(startIndexes, i)
		} else if actions[i].Execute != nil {
			executeInstructions[actions[i].Execute.Ref] = i
			executeIndexes = append(executeIndexes, i)
		}
	}

	// Start from end, to fill as much as it's possible
	slices.Reverse(executeIndexes)
	slices.Reverse(startIndexes)

	// Fast-track when there is only a single instruction to execute
	if len(executeInstructions) <= 1 {
		return [][]Action{actions}
	}

	// Basic behavior: split based on each execute instruction
	for _, executeIndex := range executeIndexes {
		ref := actions[executeIndex].Execute.Ref
		startIndex := startInstructions[ref]
		groups = append([][]Action{actions[startIndex:]}, groups...)
		actions = actions[:startIndex]
	}
	if len(actions) > 0 {
		groups[0] = append(actions, groups[0]...)
	}

	// TODO: Behavior: allow selected Toolkit actions to be executed in the same container
	// TODO: Behavior: split based on the image used (use all mounts and variables altogether)
	// TODO: Behavior: split based on the image used (isolate variables)

	return groups
}
