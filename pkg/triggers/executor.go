package triggers

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/pkg/errors"
	"k8s.io/client-go/util/jsonpath"

	commonv1 "github.com/kubeshop/testkube/api/common/v1"
	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	triggerstcl "github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/triggers"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
	"github.com/kubeshop/testkube/pkg/utils"
)

type Execution string

const (
	ExecutionTestWorkflow = "testworkflow"
	JsonPathPrefix        = "jsonpath="
)

type ExecutorF func(context.Context, *watcherEvent, *internalTrigger) error

func (s *Service) execute(ctx context.Context, e *watcherEvent, t *internalTrigger) error {
	// If the trigger was removed between match() and execute() (concurrent
	// DeleteFunc), the status entry is gone — don't fire executions for a
	// trigger that's no longer registered.
	status := s.getStatusForTrigger(t)
	if status == nil {
		s.logger.Debugf("trigger service: executor component: trigger %s/%s no longer tracked, skipping execution", t.Namespace, t.Name)
		return nil
	}

	variables := map[string]testkube.Variable{
		"WATCHER_EVENT_RESOURCE": {
			Name:  "WATCHER_EVENT_RESOURCE",
			Value: string(e.resource),
			Type_: testkube.VariableTypeBasic,
		},
		"WATCHER_EVENT_NAME": {
			Name:  "WATCHER_EVENT_NAME",
			Value: e.name,
			Type_: testkube.VariableTypeBasic,
		},
		"WATCHER_EVENT_NAMESPACE": {
			Name:  "WATCHER_EVENT_NAMESPACE",
			Value: e.Namespace,
			Type_: testkube.VariableTypeBasic,
		},
		"WATCHER_EVENT_EVENT_TYPE": {
			Name:  "WATCHER_EVENT_EVENT_TYPE",
			Value: string(e.eventType),
			Type_: testkube.VariableTypeBasic,
		},
	}

	testWorkflows, err := s.getTestWorkflowsFromInternal(t)
	if err != nil {
		return err
	}

	request := &cloud.ScheduleRequest{
		Executions: common.MapSlice(testWorkflows, func(w testworkflowsv1.TestWorkflow) *cloud.ScheduleExecution {
			execution := &cloud.ScheduleExecution{
				Selector: &cloud.ScheduleResourceSelector{Name: w.Name},
				Config:   make(map[string]string, len(variables)),
			}

			for _, variable := range variables {
				execution.Config[variable.Name] = variable.Value
			}

			if len(t.Tags) > 0 {
				execution.Tags = make(map[string]string)
			}

			// Resolve config and tag values
			var parameters = []struct {
				name string
				s    map[string]string
				d    *map[string]string
			}{
				{"config", t.Config, &execution.Config},
				{"tag", t.Tags, &execution.Tags},
			}

			// Build expression machine once, reuse for all config/tag keys
			var exprMachine expressions.Machine
			if t.Source == triggerSourceV2 {
				exprMachine = buildEventExpressionMachine(e)
			}

			for _, parameter := range parameters {
				for key, value := range parameter.s {
					if t.Source == triggerSourceV2 {
						// v2: use expression engine
						resolved, err := resolveExpressionWithMachine(exprMachine, value)
						if err != nil {
							s.logger.Errorf("trigger service: executor component: trigger %s/%s resolving %s %s error %v",
								t.Namespace, t.Name, parameter.name, key, err)
							continue
						}
						(*parameter.d)[key] = resolved
					} else if strings.HasPrefix(value, JsonPathPrefix) {
						// v1: jsonpath= prefix
						data, err := s.getJsonPathData(e, strings.TrimPrefix(value, JsonPathPrefix))
						if err != nil {
							s.logger.Errorf("trigger service: executor component: trigger %s/%s parsing jsonpath %s for %s %s error %v",
								t.Namespace, t.Name, key, value, parameter.name, err)
							continue
						}
						(*parameter.d)[key] = data
					} else {
						// v1: Go template
						data, err := s.getTemplateData(e, value)
						if err != nil {
							s.logger.Errorf("trigger service: executor component: trigger %s/%s parsing template %s for %s %s error %v",
								t.Namespace, t.Name, key, value, parameter.name, err)
							continue
						}
						(*parameter.d)[key] = string(data)
					}
				}
			}

			if t.Target != nil {
				if target := s.mapTargetKubeToGrpcWithEvent(e, t, t.Target); target != nil {
					execution.Targets = []*cloud.ExecutionTarget{target}
				}
			}

			return execution
		}),
	}

	// Pro edition only (tcl protected code)
	if s.proContext != nil && s.proContext.APIKey != "" {
		request.RunningContext, _ = testworkflowexecutor.GetNewRunningContext(triggerstcl.GetRunningContext(t.Name), nil)
	}

	if t.Delay != nil {
		s.logger.Infof(
			"trigger service: executor component: trigger %s/%s has delayed testworkflow execution configured for %f seconds",
			t.Namespace, t.Name, t.Delay.Seconds(),
		)
		time.Sleep(*t.Delay)
	}
	s.logger.Infof(
		"trigger service: executor component: scheduling testworkflow executions for trigger %s/%s",
		t.Namespace, t.Name,
	)

	executions, err := s.testWorkflowExecutor.Execute(ctx, request)
	if err != nil {
		log.DefaultLogger.Errorw(fmt.Sprintf("trigger service: error executing testworkflow for trigger %s/%s", t.Namespace, t.Name), "error", err)
		return nil
	}
	for _, exec := range executions {
		status.addTestWorkflowExecutionID(exec.Id)
	}

	status.start()
	s.logger.Debugf("trigger service: executor component: started test execution for trigger %s/%s", t.Namespace, t.Name)

	return nil
}

func (s *Service) getJsonPathData(e *watcherEvent, value string) (string, error) {
	jp := jsonpath.New("field")
	err := jp.Parse(value)
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	err = jp.Execute(buf, e.Object)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (s *Service) getTemplateData(e *watcherEvent, value string) ([]byte, error) {
	var tmpl *template.Template
	tmpl, err := utils.NewTemplate("field").Parse(value)
	if err != nil {
		return nil, err
	}

	var buffer bytes.Buffer
	if err = tmpl.ExecuteTemplate(&buffer, "field", e.Object); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// getTemplateDataFromEvent evaluates a template against the full watcher event structure
func (s *Service) getTemplateDataFromEvent(e *watcherEvent, value string) ([]byte, error) {
	var tmpl *template.Template
	tmpl, err := utils.NewTemplate("field").Parse(value)
	if err != nil {
		return nil, err
	}

	var buffer bytes.Buffer
	if err = tmpl.ExecuteTemplate(&buffer, "field", e); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// getJsonPathDataFromEvent evaluates a JSONPath against the full watcher event structure
// including agent context.
func (s *Service) getJsonPathDataFromEvent(e *watcherEvent, value string) (string, error) {
	jp := jsonpath.New("field")
	if err := jp.Parse(value); err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)
	if err := jp.Execute(buf, e); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// buildEventExpressionMachine creates a reusable expression machine for a watcher event.
// Call once per execution, reuse across all config/tag value resolutions.
func buildEventExpressionMachine(e *watcherEvent) expressions.Machine {
	machine := expressions.NewMachine().
		Register("resource", derefPtr(e.Object)).
		Register("event", map[string]interface{}{
			"type":      string(e.eventType),
			"name":      e.name,
			"namespace": e.Namespace,
		}).
		Register("agent", map[string]interface{}{
			"name":   e.Agent.Name,
			"labels": e.Agent.Labels,
		})
	if e.OldObject != nil {
		machine.Register("oldResource", derefPtr(e.OldObject))
	}
	return expressions.CombinedMachines(expressions.StdLibMachine, machine)
}

// resolveExpressionWithMachine resolves a v2 expression value against a pre-built machine.
func resolveExpressionWithMachine(machine expressions.Machine, value string) (string, error) {
	compiled, err := expressions.CompileTemplate(value)
	if err != nil {
		return "", fmt.Errorf("compile expression %q: %w", value, err)
	}

	result, err := compiled.Resolve(machine)
	if err != nil {
		return "", fmt.Errorf("resolve expression %q: %w", value, err)
	}

	if result.Static() == nil {
		return "", fmt.Errorf("expression %q could not be fully resolved", value)
	}

	if result.Static().IsNone() {
		return "", fmt.Errorf("expression %q resolved to null", value)
	}

	val, err := result.Static().StringValue()
	if err != nil {
		return "", fmt.Errorf("expression %q: %w", value, err)
	}
	return val, nil
}

// mapTargetKubeToGrpcWithEvent converts a K8s Target into a gRPC ExecutionTarget,
// resolving JSONPath (jsonpath=...) and Go templates using the watcher event object.
func (s *Service) mapTargetKubeToGrpcWithEvent(e *watcherEvent, t *internalTrigger, src *commonv1.Target) *cloud.ExecutionTarget {
	if src == nil {
		return nil
	}
	s.logger.Debugw("Trigger Service: Executor Component: Event", "event", e)

	resolve := func(kind, key string, value string) string {
		if strings.HasPrefix(value, JsonPathPrefix) {
			s.logger.Debugf("trigger service: executor component: trigger %s/%s parsing jsonpath %s for %s %s",
				t.Namespace, t.Name, key, kind, value)
			data, err := s.getJsonPathDataFromEvent(e, strings.TrimPrefix(value, JsonPathPrefix))
			if err != nil {
				s.logger.Errorf("trigger service: executor component: trigger %s/%s parsing jsonpath %s for %s %s error %v",
					t.Namespace, t.Name, key, value, kind, err)
				return ""
			}
			return data
		}

		s.logger.Debugf("trigger service: executor component: trigger %s/%s parsing template %s for %s %s",
			t.Namespace, t.Name, key, kind, value)
		data, err := s.getTemplateDataFromEvent(e, value)
		if err != nil {
			s.logger.Errorf("trigger service: executor component: trigger %s/%s parsing template %s for %s %s error %v",
				t.Namespace, t.Name, key, value, kind, err)
			return ""
		}
		return string(data)
	}

	target := &cloud.ExecutionTarget{}

	// Replicate
	if len(src.Replicate) > 0 {
		resolved := make([]string, 0, len(src.Replicate))
		for _, v := range src.Replicate {
			val := resolve("replicate", "", v)
			if val != "" {
				resolved = append(resolved, val)
			}
		}
		target.Replicate = resolved
	}

	// Match
	if src.Match != nil {
		target.Match = make(map[string]*cloud.ExecutionTargetLabels)
		for k, vs := range src.Match {
			labels := make([]string, 0, len(vs))
			for _, v := range vs {
				val := resolve("target.match", k, v)
				if val != "" {
					labels = append(labels, val)
				}
			}
			if len(labels) > 0 {
				target.Match[k] = &cloud.ExecutionTargetLabels{Labels: labels}
			}
		}
	}

	// Not
	if src.Not != nil {
		target.Not = make(map[string]*cloud.ExecutionTargetLabels)
		for k, vs := range src.Not {
			labels := make([]string, 0, len(vs))
			for _, v := range vs {
				val := resolve("target.not", k, v)
				if val != "" {
					labels = append(labels, val)
				}
			}
			if len(labels) > 0 {
				target.Not[k] = &cloud.ExecutionTargetLabels{Labels: labels}
			}
		}
	}

	return target
}

func (s *Service) getEnvironmentId() string {
	if s.proContext != nil {
		return s.proContext.EnvID
	}
	return ""
}

func (s *Service) getTestWorkflowsFromInternal(t *internalTrigger) ([]testworkflowsv1.TestWorkflow, error) {
	sel := t.WorkflowSelector
	var testWorkflows []testworkflowsv1.TestWorkflow

	if sel.Name != "" {
		s.logger.Debugf("trigger service: executor component: fetching TestWorkflow with name %s", sel.Name)
		testWorkflow, err := s.testWorkflowsClient.Get(context.Background(), s.getEnvironmentId(), sel.Name)
		if err != nil {
			return nil, err
		}
		testWorkflows = append(testWorkflows, *testworkflows.MapAPIToKube(testWorkflow))
	}

	if sel.NameRegex != "" {
		s.logger.Debugf("trigger service: executor component: fetching TestWorkflow with name regex %s", sel.NameRegex)
		testWorkflowsList, err := s.testWorkflowsClient.List(context.Background(), s.getEnvironmentId(), testworkflowclient.ListOptions{})
		if err != nil {
			return nil, err
		}

		re, err := regexp.Compile(sel.NameRegex)
		if err != nil {
			return nil, err
		}

		for i := range testWorkflowsList {
			if re.MatchString(testWorkflowsList[i].Name) {
				testWorkflows = append(testWorkflows, *testworkflows.MapAPIToKube(&testWorkflowsList[i]))
			}
		}
	}

	if sel.LabelSelector != nil {
		if len(sel.LabelSelector.MatchExpressions) > 0 {
			return nil, errors.New("error creating selector from test resource label selector: MatchExpressions not supported")
		}
		s.logger.Debugf("trigger service: executor component: fetching TestWorkflow with label %s", sel.LabelSelector.MatchLabels)
		testWorkflowsList, err := s.testWorkflowsClient.List(context.Background(), s.getEnvironmentId(), testworkflowclient.ListOptions{
			Labels: sel.LabelSelector.MatchLabels,
		})
		if err != nil {
			return nil, err
		}
		for i := range testWorkflowsList {
			testWorkflows = append(testWorkflows, *testworkflows.MapAPIToKube(&testWorkflowsList[i]))
		}
	}
	return testWorkflows, nil
}
