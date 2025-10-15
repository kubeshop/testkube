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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/jsonpath"

	commonv1 "github.com/kubeshop/testkube/api/common/v1"
	testsv3 "github.com/kubeshop/testkube/api/tests/v3"
	testsuitesv3 "github.com/kubeshop/testkube/api/testsuite/v3"
	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/scheduler"
	triggerstcl "github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/triggers"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
	"github.com/kubeshop/testkube/pkg/utils"
	"github.com/kubeshop/testkube/pkg/workerpool"
)

type Execution string

const (
	ExecutionTest         = "test"
	ExecutionTestSuite    = "testsuite"
	ExecutionTestWorkflow = "testworkflow"
	JsonPathPrefix        = "jsonpath="
)

type ExecutorF func(context.Context, *watcherEvent, *testtriggersv1.TestTrigger) error

func (s *Service) execute(ctx context.Context, e *watcherEvent, t *testtriggersv1.TestTrigger) error {
	status := s.getStatusForTrigger(t)

	concurrencyLevel := scheduler.DefaultConcurrencyLevel
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

	switch t.Spec.Execution {
	case ExecutionTest:
		tests, err := s.getTests(t)
		if err != nil {
			return err
		}

		request := testkube.ExecutionRequest{
			Variables: variables,
			RunningContext: &testkube.RunningContext{
				Type_:   string(testkube.RunningContextTypeTestTrigger),
				Context: t.Name,
			},
		}

		wp := workerpool.New[testkube.Test, testkube.ExecutionRequest, testkube.Execution](concurrencyLevel)
		go func() {
			isDelayDefined := t.Spec.Delay != nil
			if isDelayDefined {
				s.logger.Infof(
					"trigger service: executor component: trigger %s/%s has delayed test execution configured for %f seconds",
					t.Namespace, t.Name, t.Spec.Delay.Seconds(),
				)
				time.Sleep(t.Spec.Delay.Duration)
			}
			s.logger.Infof(
				"trigger service: executor component: scheduling test executions for trigger %s/%s",
				t.Namespace, t.Name,
			)
			go wp.SendRequests(s.deprecatedSystem.Scheduler.PrepareTestRequests(tests, request))
			go wp.Run(ctx)
		}()

		for r := range wp.GetResponses() {
			status.addExecutionID(r.Result.Id)
		}
	case ExecutionTestSuite:
		testSuites, err := s.getTestSuites(t)
		if err != nil {
			return err
		}

		request := testkube.TestSuiteExecutionRequest{
			Variables: variables,
			RunningContext: &testkube.RunningContext{
				Type_:   string(testkube.RunningContextTypeTestTrigger),
				Context: t.Name,
			},
		}

		wp := workerpool.New[testkube.TestSuite, testkube.TestSuiteExecutionRequest, testkube.TestSuiteExecution](concurrencyLevel)
		go func() {
			isDelayDefined := t.Spec.Delay != nil
			if isDelayDefined {
				s.logger.Infof(
					"trigger service: executor component: trigger %s/%s has delayed testsuite execution configured for %f seconds",
					t.Namespace, t.Name, t.Spec.Delay.Seconds(),
				)
				time.Sleep(t.Spec.Delay.Duration)
			}
			s.logger.Infof(
				"trigger service: executor component: scheduling testsuite executions for trigger %s/%s",
				t.Namespace, t.Name,
			)
			go wp.SendRequests(s.deprecatedSystem.Scheduler.PrepareTestSuiteRequests(testSuites, request))
			go wp.Run(ctx)
		}()

		for r := range wp.GetResponses() {
			status.addTestSuiteExecutionID(r.Result.Id)
		}

	case ExecutionTestWorkflow:
		testWorkflows, err := s.getTestWorkflows(t)
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

				if t.Spec.ActionParameters != nil {
					if len(t.Spec.ActionParameters.Tags) > 0 && execution.Tags == nil {
						execution.Tags = make(map[string]string)
					}

					var parameters = []struct {
						name string
						s    *map[string]string
						d    *map[string]string
					}{
						{
							"config",
							&t.Spec.ActionParameters.Config,
							&execution.Config,
						},
						{
							"tag",
							&t.Spec.ActionParameters.Tags,
							&execution.Tags,
						},
					}

					for _, parameter := range parameters {
						for key, value := range *parameter.s {
							if strings.HasPrefix(value, JsonPathPrefix) {
								s.logger.Debugf("trigger service: executor component: trigger %s/%s parsing jsonpath %s for %s %s",
									t.Namespace, t.Name, key, parameter.name, value)

								data, err := s.getJsonPathData(e, strings.TrimPrefix(value, JsonPathPrefix))

								if err != nil {
									s.logger.Errorf("trigger service: executor component: trigger %s/%s parsing jsonpath %s for %s %s error %v",
										t.Namespace, t.Name, key, value, parameter.name, err)
									continue
								}

								(*parameter.d)[key] = data
							} else {
								s.logger.Debugf("trigger service: executor component: trigger %s/%s parsing template %s for %s %s",
									t.Namespace, t.Name, key, parameter.name, value)
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
				}

				if t.Spec.ActionParameters != nil && t.Spec.ActionParameters.Target != nil {
					if target := s.mapTargetKubeToGrpcWithEvent(e, t, t.Spec.ActionParameters.Target); target != nil {
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

		isDelayDefined := t.Spec.Delay != nil
		if isDelayDefined {
			s.logger.Infof(
				"trigger service: executor component: trigger %s/%s has delayed testworkflow execution configured for %f seconds",
				t.Namespace, t.Name, t.Spec.Delay.Seconds(),
			)
			time.Sleep(t.Spec.Delay.Duration)
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

	default:
		return errors.Errorf("invalid execution: %s", t.Spec.Execution)
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

// mapTargetKubeToGrpcWithEvent converts a K8s Target into a gRPC ExecutionTarget,
// resolving JSONPath (jsonpath=...) and Go templates using the watcher event object.
func (s *Service) mapTargetKubeToGrpcWithEvent(e *watcherEvent, t *testtriggersv1.TestTrigger, src *commonv1.Target) *cloud.ExecutionTarget {
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

func (s *Service) getTests(t *testtriggersv1.TestTrigger) ([]testsv3.Test, error) {
	var tests []testsv3.Test
	if t.Spec.TestSelector.Name != "" {
		s.logger.Debugf("trigger service: executor component: fetching testsv3.Test with name %s", t.Spec.TestSelector.Name)
		test, err := s.deprecatedSystem.Clients.Tests().Get(t.Spec.TestSelector.Name)
		if err != nil {
			return nil, err
		}
		tests = append(tests, *test)
	}

	if t.Spec.TestSelector.NameRegex != "" {
		s.logger.Debugf("trigger service: executor component: fetching testsv3.Test with name regex %s", t.Spec.TestSelector.NameRegex)
		testList, err := s.deprecatedSystem.Clients.Tests().List("")
		if err != nil {
			return nil, err
		}

		re, err := regexp.Compile(t.Spec.TestSelector.NameRegex)
		if err != nil {
			return nil, err
		}

		for i := range testList.Items {
			if re.MatchString(testList.Items[i].Name) {
				tests = append(tests, testList.Items[i])
			}
		}
	}

	if t.Spec.TestSelector.LabelSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(t.Spec.TestSelector.LabelSelector)
		if err != nil {
			return nil, errors.WithMessagef(err, "error creating selector from test resource label selector")
		}
		stringifiedSelector := selector.String()
		s.logger.Debugf("trigger service: executor component: fetching testsv3.Test with labels %s", stringifiedSelector)
		testList, err := s.deprecatedSystem.Clients.Tests().List(stringifiedSelector)
		if err != nil {
			return nil, err
		}
		tests = append(tests, testList.Items...)
	}
	return tests, nil
}

func (s *Service) getTestSuites(t *testtriggersv1.TestTrigger) ([]testsuitesv3.TestSuite, error) {
	var testSuites []testsuitesv3.TestSuite
	if t.Spec.TestSelector.Name != "" {
		s.logger.Debugf("trigger service: executor component: fetching testsuitesv3.TestSuite with name %s", t.Spec.TestSelector.Name)
		testSuite, err := s.deprecatedSystem.Clients.TestSuites().Get(t.Spec.TestSelector.Name)
		if err != nil {
			return nil, err
		}
		testSuites = append(testSuites, *testSuite)
	}

	if t.Spec.TestSelector.NameRegex != "" {
		s.logger.Debugf("trigger service: executor component: fetching testsuitesv3.TestSuite with name regex %s", t.Spec.TestSelector.NameRegex)
		testSuitesList, err := s.deprecatedSystem.Clients.TestSuites().List("")
		if err != nil {
			return nil, err
		}

		re, err := regexp.Compile(t.Spec.TestSelector.NameRegex)
		if err != nil {
			return nil, err
		}

		for i := range testSuitesList.Items {
			if re.MatchString(testSuitesList.Items[i].Name) {
				testSuites = append(testSuites, testSuitesList.Items[i])
			}
		}
	}

	if t.Spec.TestSelector.LabelSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(t.Spec.TestSelector.LabelSelector)
		if err != nil {
			return nil, errors.WithMessagef(err, "error creating selector from test resource label selector")
		}
		stringifiedSelector := selector.String()
		s.logger.Debugf("trigger service: executor component: fetching testsuitesv3.TestSuite with label %s", stringifiedSelector)
		testSuitesList, err := s.deprecatedSystem.Clients.TestSuites().List(stringifiedSelector)
		if err != nil {
			return nil, err
		}
		testSuites = append(testSuites, testSuitesList.Items...)
	}
	return testSuites, nil
}

func (s *Service) getEnvironmentId() string {
	if s.proContext != nil {
		return s.proContext.EnvID
	}
	return ""
}

func (s *Service) getTestWorkflows(t *testtriggersv1.TestTrigger) ([]testworkflowsv1.TestWorkflow, error) {
	var testWorkflows []testworkflowsv1.TestWorkflow
	if t.Spec.TestSelector.Name != "" {
		s.logger.Debugf("trigger service: executor component: fetching testworkflowsv3.TestWorkflow with name %s", t.Spec.TestSelector.Name)

		testWorkflow, err := s.testWorkflowsClient.Get(context.Background(), s.getEnvironmentId(), t.Spec.TestSelector.Name)
		if err != nil {
			return nil, err
		}
		testWorkflows = append(testWorkflows, *testworkflows.MapAPIToKube(testWorkflow))
	}

	if t.Spec.TestSelector.NameRegex != "" {
		s.logger.Debugf("trigger service: executor component: fetching testworkflosv1.TestWorkflow with name regex %s", t.Spec.TestSelector.NameRegex)
		testWorkflowsList, err := s.testWorkflowsClient.List(context.Background(), s.getEnvironmentId(), testworkflowclient.ListOptions{})
		if err != nil {
			return nil, err
		}

		re, err := regexp.Compile(t.Spec.TestSelector.NameRegex)
		if err != nil {
			return nil, err
		}

		for i := range testWorkflowsList {
			if re.MatchString(testWorkflowsList[i].Name) {
				testWorkflows = append(testWorkflows, *testworkflows.MapAPIToKube(&testWorkflowsList[i]))
			}
		}
	}

	if t.Spec.TestSelector.LabelSelector != nil {
		if len(t.Spec.TestSelector.LabelSelector.MatchExpressions) > 0 {
			return nil, errors.New("error creating selector from test resource label selector: MatchExpressions not supported")
		}
		s.logger.Debugf("trigger service: executor component: fetching testworkflowsv1.TestWorkflow with label %s", t.Spec.TestSelector.LabelSelector.MatchLabels)
		testWorkflowsList, err := s.testWorkflowsClient.List(context.Background(), s.getEnvironmentId(), testworkflowclient.ListOptions{
			Labels: t.Spec.TestSelector.LabelSelector.MatchLabels,
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
