package testworkflowprocessor

//
//import (
//	"errors"
//	"fmt"
//	"maps"
//	"strconv"
//	"strings"
//
//	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
//	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
//	"github.com/kubeshop/testkube/internal/common"
//	"github.com/kubeshop/testkube/pkg/expressions"
//	constants2 "github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
//)
//
//type initProcess struct {
//	ref        string
//	workingDir string
//	init       []string
//	params     []string
//	retry      map[string]testworkflowsv1.RetryPolicy
//	paused     bool
//	command    []string
//	args       []string
//	envs       []string
//	results    []string
//	conditions map[string][]string
//	negative   bool
//	toolkit    bool
//	errors     []error
//}
//
//func NewInitProcess() *initProcess {
//	return &initProcess{
//		conditions: map[string][]string{},
//		retry:      map[string]testworkflowsv1.RetryPolicy{},
//	}
//}
//
//func (p *initProcess) Error() error {
//	if len(p.errors) == 0 {
//		return nil
//	}
//	return errors.Join(p.errors...)
//}
//
//func (p *initProcess) SetRef(ref string) *initProcess {
//	p.ref = ref
//	return p
//}
//
//func (p *initProcess) Command() []string {
//	args := p.params
//
//	// TODO: Support nested retries
//	policy, ok := p.retry[p.ref]
//	if ok {
//		args = append(args, constants.ArgRetryCount, strconv.Itoa(int(policy.Count)), constants.ArgRetryUntil, expressions.Escape(policy.Until))
//	}
//	if p.negative {
//		args = append(args, constants.ArgNegative, "true")
//	}
//	if len(p.init) > 0 {
//		args = append(args, constants.ArgInit, strings.Join(p.init, "&&"))
//	}
//	if len(p.envs) > 0 {
//		args = append(args, constants.ArgComputeEnv, strings.Join(p.envs, ","))
//	}
//	if len(p.conditions) > 0 {
//		for k, v := range p.conditions {
//			args = append(args, constants.ArgCondition, fmt.Sprintf("%s=%s", strings.Join(common.UniqueSlice(v), ","), k))
//		}
//	}
//	for _, r := range p.results {
//		args = append(args, constants.ArgResult, r)
//	}
//	if p.paused {
//		args = append(args, constants.ArgPaused)
//	}
//	if p.workingDir != "" {
//		args = append(args, constants.ArgWorkingDir, p.workingDir)
//	}
//	if p.toolkit {
//		args = append(args, constants.ArgToolkit)
//	}
//	return append([]string{constants2.DefaultInitPath, p.ref}, append(args, constants.ArgSeparator)...)
//}
//
//func (p *initProcess) Args() []string {
//	args := make([]string, 0)
//	if len(p.command) > 0 {
//		args = p.command
//	}
//	if len(p.command) > 0 || len(p.args) > 0 {
//		args = append(args, p.args...)
//	}
//	return args
//}
//
//func (p *initProcess) param(args ...string) *initProcess {
//	p.params = append(p.params, args...)
//	return p
//}
//
//func (p *initProcess) compile(expr ...string) []string {
//	for i, e := range expr {
//		res, err := expressions.Compile(e)
//		if err == nil {
//			expr[i] = res.String()
//		} else {
//			p.errors = append(p.errors, fmt.Errorf("resolving expression: %s: %s", expr[i], err.Error()))
//		}
//	}
//	return expr
//}
//
//func (p *initProcess) SetCommand(command ...string) *initProcess {
//	p.command = command
//	return p
//}
//
//func (p *initProcess) SetArgs(args ...string) *initProcess {
//	p.args = args
//	return p
//}
//
//func (p *initProcess) SetToolkit(toolkit bool) *initProcess {
//	p.toolkit = toolkit
//	return p
//}
//
//func (p *initProcess) AddTimeout(duration string, refs ...string) *initProcess {
//	return p.param(constants.ArgTimeout, fmt.Sprintf("%s=%s", strings.Join(refs, ","), duration))
//}
//
//func (p *initProcess) SetInitialStatus(expr ...string) *initProcess {
//	p.init = nil
//	for _, v := range p.compile(expr...) {
//		p.init = append(p.init, v)
//	}
//	return p
//}
//
//func (p *initProcess) PrependInitialStatus(expr ...string) *initProcess {
//	init := []string(nil)
//	for _, v := range p.compile(expr...) {
//		init = append(init, v)
//	}
//	p.init = append(init, p.init...)
//	return p
//}
//
//func (p *initProcess) AddComputedEnvs(names ...string) *initProcess {
//	p.envs = append(p.envs, names...)
//	return p
//}
//
//func (p *initProcess) SetNegative(negative bool) *initProcess {
//	p.negative = negative
//	return p
//}
//
//func (p *initProcess) AddResult(condition string, refs ...string) *initProcess {
//	if len(refs) == 0 || condition == "" {
//		return p
//	}
//	p.results = append(p.results, fmt.Sprintf("%s=%s", strings.Join(refs, ","), p.compile(condition)[0]))
//	return p
//}
//
//func (p *initProcess) ResetResults() *initProcess {
//	p.results = nil
//	return p
//}
//
//func (p *initProcess) AddCondition(condition string, refs ...string) *initProcess {
//	if len(refs) == 0 || condition == "" {
//		return p
//	}
//	expr := p.compile(condition)[0]
//	p.conditions[expr] = append(p.conditions[expr], refs...)
//	return p
//}
//
//func (p *initProcess) ResetCondition() *initProcess {
//	p.conditions = make(map[string][]string)
//	return p
//}
//
//func (p *initProcess) AddRetryPolicy(policy testworkflowsv1.RetryPolicy, ref string) *initProcess {
//	if policy.Count <= 0 {
//		delete(p.retry, ref)
//		return p
//	}
//	until := policy.Until
//	if until == "" {
//		until = "passed"
//	}
//	p.retry[ref] = testworkflowsv1.RetryPolicy{Count: policy.Count, Until: until}
//	return p
//}
//
//func (p *initProcess) SetPaused(paused bool) *initProcess {
//	p.paused = paused
//	return p
//}
//
//func (p *initProcess) SetWorkingDir(workingDir string) *initProcess {
//	p.workingDir = workingDir
//	return p
//}
//
//func (p *initProcess) Children(ref string) *initProcess {
//	return &initProcess{
//		ref:        ref,
//		params:     p.params,
//		retry:      maps.Clone(p.retry),
//		paused:     p.paused,
//		command:    p.command,
//		args:       p.args,
//		workingDir: p.workingDir,
//		init:       p.init,
//		envs:       p.envs,
//		results:    p.results,
//		conditions: maps.Clone(p.conditions),
//		negative:   p.negative,
//		toolkit:    p.toolkit,
//		errors:     p.errors,
//	}
//}
