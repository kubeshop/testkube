package testworkflowprocessor

import (
	"encoding/json"
	"fmt"
	"reflect"
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
	constants2 "github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
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

//func buildKubernetesContainers(stage Stage, init *initProcess, fsGroup *int64, machines ...expressions.Machine) (containers []corev1.Container, err error) {
//	if stage.Paused() {
//		init.SetPaused(stage.Paused())
//	}
//	if stage.Timeout() != "" {
//		init.AddTimeout(stage.Timeout(), stage.Ref())
//	}
//	if stage.Ref() != "" {
//		init.AddCondition(stage.Condition(), stage.Ref())
//	}
//	init.AddRetryPolicy(stage.RetryPolicy(), stage.Ref())
//
//	group, ok := stage.(GroupStage)
//	if ok {
//		recursiveRefs := common.MapSlice(group.RecursiveChildren(), getRef)
//		directRefResults := common.MapSlice(common.FilterSlice(group.Children(), isNotOptional), getRef)
//
//		init.AddCondition(stage.Condition(), recursiveRefs...)
//
//		if group.Negative() {
//			// Create virtual layer that will be put down into actual negative step
//			init.SetRef(stage.Ref() + ".v")
//			init.AddCondition(stage.Condition(), stage.Ref()+".v")
//			init.PrependInitialStatus(stage.Ref() + ".v")
//			init.AddResult("!"+stage.Ref()+".v", stage.Ref())
//		} else if stage.Ref() != "" {
//			init.PrependInitialStatus(stage.Ref())
//		}
//
//		if group.Optional() {
//			init.ResetResults()
//		}
//
//		if group.Negative() {
//			init.AddResult(strings.Join(directRefResults, "&&"), stage.Ref()+".v")
//		} else {
//			init.AddResult(strings.Join(directRefResults, "&&"), stage.Ref())
//		}
//
//		for i, ch := range group.Children() {
//			// Condition should be executed only in the first leaf
//			if i == 1 {
//				init.ResetCondition().SetPaused(false)
//			}
//			// Pass down to another group or container
//			sub, serr := buildKubernetesContainers(ch, init.Children(ch.Ref()), fsGroup, machines...)
//			if serr != nil {
//				return nil, fmt.Errorf("%s: %s: resolving children: %s", stage.Ref(), stage.Name(), serr.Error())
//			}
//			containers = append(containers, sub...)
//		}
//		return
//	}
//	c, ok := stage.(ContainerStage)
//	if !ok {
//		return nil, fmt.Errorf("%s: %s: stage that is neither container nor group", stage.Ref(), stage.Name())
//	}
//	err = c.Container().Detach().Resolve(machines...)
//	if err != nil {
//		return nil, fmt.Errorf("%s: %s: resolving container: %s", stage.Ref(), stage.Name(), err.Error())
//	}
//
//	cr, err := c.Container().ToKubernetesTemplate()
//	if err != nil {
//		return nil, fmt.Errorf("%s: %s: building container template: %s", stage.Ref(), stage.Name(), err.Error())
//	}
//	cr.Name = c.Ref()
//
//	if c.Optional() {
//		init.ResetResults()
//	}
//
//	bypass := false
//	refEnvVar := ""
//	for _, e := range cr.Env {
//		if e.Name == BypassToolkitCheck.Name && e.Value == BypassToolkitCheck.Value {
//			bypass = true
//		}
//		if e.Name == "TK_REF" {
//			refEnvVar = e.Value
//		}
//	}
//
//	init.
//		SetNegative(c.Negative()).
//		AddRetryPolicy(c.RetryPolicy(), c.Ref()).
//		SetCommand(cr.Command...).
//		SetArgs(cr.Args...).
//		SetWorkingDir(cr.WorkingDir).
//		SetToolkit(bypass || (cr.Image == constants.DefaultToolkitImage && c.Ref() == refEnvVar))
//
//	for _, env := range cr.Env {
//		if strings.Contains(env.Value, "{{") {
//			init.AddComputedEnvs(env.Name)
//		}
//	}
//
//	if init.Error() != nil {
//		return nil, init.Error()
//	}
//
//	cr.Command = init.Command()
//	cr.Args = init.Args()
//	cr.WorkingDir = ""
//
//	// Ensure the container will have proper access to FS
//	if cr.SecurityContext == nil {
//		cr.SecurityContext = &corev1.SecurityContext{}
//	}
//	if cr.SecurityContext.RunAsGroup == nil {
//		cr.SecurityContext.RunAsGroup = fsGroup
//	}
//
//	containers = []corev1.Container{cr}
//	return
//}

type ActionResult struct {
	Ref   string `json:"r"`
	Value string `json:"v"`
}

type ActionDeclare struct {
	Condition string   `json:"c"`
	Ref       string   `json:"r"`
	Parents   []string `json:"p,omitempty"`
}

type ActionExecute struct {
	Ref      string `json:"r"`
	Negative bool   `json:"n,omitempty"`
}

type ActionContainer struct {
	Ref    string                          `json:"r"`
	Config testworkflowsv1.ContainerConfig `json:"c"`
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

type ActionSetup struct {
	CopyInit     bool `json:"i,omitempty"`
	CopyBinaries bool `json:"b,omitempty"`
}

type Action struct {
	CurrentStatus *string          `json:"s,omitempty"`
	Start         *string          `json:"S,omitempty"`
	End           *string          `json:"E,omitempty"`
	Setup         *ActionSetup     `json:"_,omitempty"`
	Declare       *ActionDeclare   `json:"d,omitempty"`
	Result        *ActionResult    `json:"r,omitempty"`
	Container     *ActionContainer `json:"c,omitempty"`
	Execute       *ActionExecute   `json:"e,omitempty"`
	Timeout       *ActionTimeout   `json:"t,omitempty"`
	Pause         *ActionPause     `json:"p,omitempty"`
	Retry         *ActionRetry     `json:"R,omitempty"`
}

type ActionType string

const (
	// Declarations
	ActionTypeDeclare ActionType = "declare"
	ActionTypePause              = "pause"
	ActionTypeResult             = "result"
	ActionTypeTimeout            = "timeout"
	ActionTypeRetry              = "retry"

	// Operations
	ActionTypeContainerTransition = "container"
	ActionTypeCurrentStatus       = "status"
	ActionTypeStart               = "start"
	ActionTypeEnd                 = "end"
	ActionTypeSetup               = "setup"
	ActionTypeExecute             = "execute"
)

func (a *Action) Type() ActionType {
	if a.Declare != nil {
		return ActionTypeDeclare
	} else if a.Pause != nil {
		return ActionTypePause
	} else if a.Result != nil {
		return ActionTypeResult
	} else if a.Timeout != nil {
		return ActionTypeTimeout
	} else if a.Retry != nil {
		return ActionTypeRetry
	} else if a.Container != nil {
		return ActionTypeContainerTransition
	} else if a.CurrentStatus != nil {
		return ActionTypeCurrentStatus
	} else if a.Start != nil {
		return ActionTypeStart
	} else if a.End != nil {
		return ActionTypeEnd
	} else if a.Setup != nil {
		return ActionTypeSetup
	} else if a.Execute != nil {
		return ActionTypeExecute
	}
	v, e := json.Marshal(a)
	panic(fmt.Sprintf("unknown action type: %s, %v", v, e))
}

// TODO: Wrap all errors in this file
// TODO: tail-recursive
func analyzeOperations(currentStatus string, parents []string, stage Stage, machines ...expressions.Machine) (actions []Action, err error) {
	// Store the init status
	actions = append(actions, Action{
		CurrentStatus: common.Ptr(currentStatus),
	})

	// Compute the skip condition
	condition := stage.Condition()
	if condition == "" || condition == "null" {
		condition = "passed" // TODO: Think if it should default the condition to "passed"
	}
	actions = append(actions, Action{
		Declare: &ActionDeclare{Ref: stage.Ref(), Condition: condition, Parents: parents},
	})

	// Configure the container for action
	// TODO: Handle the ContainerDefaults properly
	var containerConfig Container
	if group, ok := stage.(GroupStage); ok {
		containerConfig = group.ContainerDefaults()
	} else {
		containerConfig = stage.(ContainerStage).Container()
	}
	if containerConfig != nil {
		c := containerConfig.Detach()
		err = c.Resolve(machines...)
		if err != nil {
			return nil, err
		}
		actions = append(actions, Action{
			Container: &ActionContainer{Ref: stage.Ref(), Config: c.ToContainerConfig()},
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
		actions = append(actions, Action{
			Execute: &ActionExecute{
				Ref:      exec.Ref(),
				Negative: exec.Negative(),
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

	// Move declarations to top
	slices.SortStableFunc(actions, func(a Action, b Action) int {
		if (a.Declare == nil) == (b.Declare == nil) {
			return 0
		}
		if a.Declare == nil {
			return 1
		}
		return -1
	})

	// Move setup to top
	slices.SortStableFunc(actions, func(a Action, b Action) int {
		if (a.Setup == nil) == (b.Setup == nil) {
			return 0
		}
		if a.Setup == nil {
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
	actions = append([]Action{{Setup: &ActionSetup{CopyInit: true, CopyBinaries: true}}, {Start: common.Ptr("")}}, actions...)
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

// TODO: Handle Group Stages properly with isolation (to have conditions working perfectly fine, i.e. for isolated image + file() clause)
func GroupActions(actions []Action) (groups [][]Action) {
	// Detect "start" and "execute" instructions
	startIndexes := make([]int, 0)
	startInstructions := make(map[string]int)
	containerInstructions := make(map[string]int)
	executeInstructions := make(map[string]int)
	executeIndexes := make([]int, 0)
	for i := range actions {
		if actions[i].Start != nil {
			startInstructions[*actions[i].Start] = i
			startIndexes = append(startIndexes, i)
		} else if actions[i].Execute != nil {
			executeInstructions[actions[i].Execute.Ref] = i
			executeIndexes = append(executeIndexes, i)
		} else if actions[i].Container != nil {
			containerInstructions[actions[i].Container.Ref] = i
		} else if actions[i].Setup != nil {
			executeIndexes = append(executeIndexes, i)
		}
	}

	// Start from end, to fill as much as it's possible
	slices.Reverse(executeIndexes)
	slices.Reverse(startIndexes)

	// Fast-track when there is only a single instruction to execute
	if len(executeIndexes) <= 1 {
		return [][]Action{actions}
	}

	// Basic behavior: split based on each execute instruction
	for _, executeIndex := range executeIndexes {
		if actions[executeIndex].Setup != nil {
			groups = append([][]Action{actions[executeIndex:]}, groups...)
			actions = actions[:executeIndex]
			continue
		}
		ref := actions[executeIndex].Execute.Ref
		startIndex := startInstructions[ref]
		if containerIndex, ok := containerInstructions[ref]; ok && containerIndex < startIndex {
			startIndex = containerIndex
		}

		//// FIXME: Delete, it's a hack to combine steps with same image into a single container
		//if i != len(executeIndexes)-1 {
		//	prevRef := actions[executeIndex].Execute.Ref
		//	prevContainerIndex, prevOk := containerInstructions[prevRef]
		//	containerIndex, containerOk := containerInstructions[ref]
		//	if !containerOk || (prevOk && actions[prevContainerIndex].Container.Config.Image == actions[containerIndex].Container.Config.Image) {
		//		continue
		//	}
		//}

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

// TODO: [{"c":{"c":"true","r":"root"}},{"c":{"c":"true","r":"rbdtn40","p":["root"]}},{"r":{"r":"root","v":"rbdtn40"}},{"r":{"r":"","v":"root"}},{},{"s":"true"},{"S":"root"},{"s":"root"},{"S":"rbdtn40"},{"e":{"r":"rbdtn40","c":{"command":["/.tktw/bin/sh"],"args":["-c","set -e\nsleep 10 \u0026\u0026 echo $SOMETHING \u0026\u0026 tree /.tktw"]}}},{"E":"rbdtn40"},{"E":"root"},{}]
//       There is some empty object on the last one

// TODO: Disallow bypassing
func BuildContainer(groupId int, defaultContainer Container, actions []Action) (cr corev1.Container, actionsCleanup []Action, err error) {
	actions = slices.Clone(actions)
	actionsCleanup = actions

	// Find the container configurations and executable/setup steps
	var setup *Action
	executable := map[string]bool{}
	containerConfigs := make([]*Action, 0)
	for i := range actions {
		if actions[i].Container != nil {
			containerConfigs = append(containerConfigs, &actions[i])
		} else if actions[i].Setup != nil {
			setup = &actions[i]
		} else if actions[i].Execute != nil {
			executable[actions[i].Execute.Ref] = true
		}
	}

	// Find the highest priority container configuration
	var bestContainerConfig *Action
	for i := range containerConfigs {
		if executable[containerConfigs[i].Container.Ref] {
			bestContainerConfig = containerConfigs[i]
			break
		}
	}
	if bestContainerConfig == nil && len(containerConfigs) > 0 {
		bestContainerConfig = containerConfigs[len(containerConfigs)-1]
	}

	// Build the cr base
	// TODO: Handle the case when there are multiple exclusive execution configurations
	// TODO: Handle a case when that configuration should join multiple configurations (i.e. envs/volumeMounts)
	if len(containerConfigs) > 0 {
		cr, err = NewContainer().ApplyCR(&bestContainerConfig.Container.Config).ToKubernetesTemplate()
		if err != nil {
			return corev1.Container{}, nil, err
		}

		// Combine environment variables from each execution
		cr.Env = nil
		cr.EnvFrom = nil
		for i := range containerConfigs {
			for _, e := range containerConfigs[i].Container.Config.Env {
				newEnv := *e.DeepCopy()
				if strings.Contains(newEnv.Value, "{{") {
					newEnv.Name = fmt.Sprintf("_%dC_%s", i, e.Name)
				} else {
					newEnv.Name = fmt.Sprintf("_%d_%s", i, e.Name)
				}
				cr.Env = append(cr.Env, newEnv)
			}
			for _, e := range containerConfigs[i].Container.Config.EnvFrom {
				newEnvFrom := *e.DeepCopy()
				newEnvFrom.Prefix = fmt.Sprintf("_%d_%s", i, e.Prefix)
				cr.EnvFrom = append(cr.EnvFrom, newEnvFrom)
			}
		}
		// TODO: Combine the rest
	}

	// Set up a default image when not specified
	if cr.Image == "" {
		cr.Image = constants.DefaultInitImage
		cr.ImagePullPolicy = corev1.PullIfNotPresent
	}

	// Provide the data required for setup step
	if setup != nil {
		cr.Env = append(cr.Env,
			corev1.EnvVar{Name: fmt.Sprintf("_00_%s", constants2.EnvNodeName), ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"},
			}},
			corev1.EnvVar{Name: fmt.Sprintf("_00_%s", constants2.EnvPodName), ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"},
			}},
			corev1.EnvVar{Name: fmt.Sprintf("_00_%s", constants2.EnvNamespaceName), ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"},
			}},
			corev1.EnvVar{Name: fmt.Sprintf("_00_%s", constants2.EnvServiceAccountName), ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.serviceAccountName"},
			}},
			corev1.EnvVar{Name: fmt.Sprintf("_01_%s", constants2.EnvInstructions), ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: fmt.Sprintf("metadata.annotations['%s']", constants.SpecAnnotationName)},
			}})

		// Apply basic mounts, so there is a state provided
		for _, volumeMount := range defaultContainer.VolumeMounts() {
			if !slices.ContainsFunc(cr.VolumeMounts, func(mount corev1.VolumeMount) bool {
				return mount.Name == volumeMount.Name
			}) {
				cr.VolumeMounts = append(cr.VolumeMounts, volumeMount)
			}
		}
	}

	// TODO: Avoid using /.tktw/init if there is Init Image - use /init then
	initPath := constants.DefaultInitPath
	if cr.Image == constants.DefaultInitImage {
		initPath = "/init"
	}

	// TODO: Avoid using /.tktw/toolkit if there is Toolkit image

	// TODO: Avoid using /.tktw/bin/sh (and other binaries) if there is Init image - use /bin/* then

	// TODO: Copy /init and /toolkit in the Init Container only if there is a need to.
	//       Probably, include Setup step in the Actions list, so it can be simplified too into a single container,
	//       and optimized along with others.

	// Point the Init Process to the proper group
	cr.Name = fmt.Sprintf("%d", groupId+1)
	cr.Command = []string{initPath, fmt.Sprintf("%d", groupId)}
	cr.Args = nil

	// Clean up the executions
	for i := range containerConfigs {
		// TODO: Clean it up
		newConfig := testworkflowsv1.ContainerConfig{}
		if executable[containerConfigs[i].Container.Ref] {
			newConfig.Command = containerConfigs[i].Container.Config.Command
			newConfig.Args = containerConfigs[i].Container.Config.Args
		}
		newConfig.WorkingDir = containerConfigs[i].Container.Config.WorkingDir
		// TODO: expose more?

		containerConfigs[i].Container = &ActionContainer{
			Ref:    containerConfigs[i].Container.Ref,
			Config: newConfig,
		}
	}

	return
}
