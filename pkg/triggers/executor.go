package triggers

import (
	"context"
	"regexp"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testsv3 "github.com/kubeshop/testkube-operator/api/tests/v3"
	testsuitesv3 "github.com/kubeshop/testkube-operator/api/testsuite/v3"
	testtriggersv1 "github.com/kubeshop/testkube-operator/api/testtriggers/v1"
	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/scheduler"
	"github.com/kubeshop/testkube/pkg/workerpool"
)

type Execution string

const (
	ExecutionTest         = "test"
	ExecutionTestSuite    = "testsuite"
	ExecutionTestWorkflow = "testworkflow"
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
			Value: e.namespace,
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
			go wp.SendRequests(s.scheduler.PrepareTestRequests(tests, request))
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
			go wp.SendRequests(s.scheduler.PrepareTestSuiteRequests(testSuites, request))
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

		request := testkube.TestWorkflowExecutionRequest{
			Config: make(map[string]string, len(variables)),
		}

		for _, variable := range variables {
			request.Config[variable.Name] = variable.Value
		}

		wp := workerpool.New[testworkflowsv1.TestWorkflow, testkube.TestWorkflowExecutionRequest, testkube.TestWorkflowExecution](concurrencyLevel)
		go func() {
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

			requests := make([]workerpool.Request[testworkflowsv1.TestWorkflow, testkube.TestWorkflowExecutionRequest,
				testkube.TestWorkflowExecution], len(testWorkflows))
			for i := range testWorkflows {
				requests[i] = workerpool.Request[testworkflowsv1.TestWorkflow, testkube.TestWorkflowExecutionRequest,
					testkube.TestWorkflowExecution]{
					Object:  testWorkflows[i],
					Options: request,
					// Pro edition only (tcl protected code)
					ExecFn: s.testWorkflowExecutor.Execute,
				}
			}

			go wp.SendRequests(requests)
			go wp.Run(ctx)
		}()

		for r := range wp.GetResponses() {
			status.addTestWorkflowExecutionID(r.Result.Id)
		}

	default:
		return errors.Errorf("invalid execution: %s", t.Spec.Execution)
	}

	status.start()
	s.logger.Debugf("trigger service: executor component: started test execution for trigger %s/%s", t.Namespace, t.Name)

	return nil
}

func (s *Service) getTests(t *testtriggersv1.TestTrigger) ([]testsv3.Test, error) {
	var tests []testsv3.Test
	if t.Spec.TestSelector.Name != "" {
		s.logger.Debugf("trigger service: executor component: fetching testsv3.Test with name %s", t.Spec.TestSelector.Name)
		test, err := s.testsClient.Get(t.Spec.TestSelector.Name)
		if err != nil {
			return nil, err
		}
		tests = append(tests, *test)
	}

	if t.Spec.TestSelector.NameRegex != "" {
		s.logger.Debugf("trigger service: executor component: fetching testsv3.Test with name regex %s", t.Spec.TestSelector.NameRegex)
		testList, err := s.testsClient.List("")
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
		testList, err := s.testsClient.List(stringifiedSelector)
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
		testSuite, err := s.testSuitesClient.Get(t.Spec.TestSelector.Name)
		if err != nil {
			return nil, err
		}
		testSuites = append(testSuites, *testSuite)
	}

	if t.Spec.TestSelector.NameRegex != "" {
		s.logger.Debugf("trigger service: executor component: fetching testsuitesv3.TestSuite with name regex %s", t.Spec.TestSelector.NameRegex)
		testSuitesList, err := s.testSuitesClient.List("")
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
		testSuitesList, err := s.testSuitesClient.List(stringifiedSelector)
		if err != nil {
			return nil, err
		}
		testSuites = append(testSuites, testSuitesList.Items...)
	}
	return testSuites, nil
}

func (s *Service) getTestWorkflows(t *testtriggersv1.TestTrigger) ([]testworkflowsv1.TestWorkflow, error) {
	var testWorkflows []testworkflowsv1.TestWorkflow
	if t.Spec.TestSelector.Name != "" {
		s.logger.Debugf("trigger service: executor component: fetching testworkflowsv3.TestWorkflow with name %s", t.Spec.TestSelector.Name)
		testWorkflow, err := s.testWorkflowsClient.Get(t.Spec.TestSelector.Name)
		if err != nil {
			return nil, err
		}
		testWorkflows = append(testWorkflows, *testWorkflow)
	}

	if t.Spec.TestSelector.NameRegex != "" {
		s.logger.Debugf("trigger service: executor component: fetching testworkflosv1.TestWorkflow with name regex %s", t.Spec.TestSelector.NameRegex)
		testWorkflowsList, err := s.testWorkflowsClient.List("")
		if err != nil {
			return nil, err
		}

		re, err := regexp.Compile(t.Spec.TestSelector.NameRegex)
		if err != nil {
			return nil, err
		}

		for i := range testWorkflowsList.Items {
			if re.MatchString(testWorkflowsList.Items[i].Name) {
				testWorkflows = append(testWorkflows, testWorkflowsList.Items[i])
			}
		}
	}

	if t.Spec.TestSelector.LabelSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(t.Spec.TestSelector.LabelSelector)
		if err != nil {
			return nil, errors.WithMessagef(err, "error creating selector from test resource label selector")
		}
		stringifiedSelector := selector.String()
		s.logger.Debugf("trigger service: executor component: fetching testworkflowsv1.TestWorkflow with label %s", stringifiedSelector)
		testWorkflowsList, err := s.testWorkflowsClient.List(stringifiedSelector)
		if err != nil {
			return nil, err
		}
		testWorkflows = append(testWorkflows, testWorkflowsList.Items...)
	}
	return testWorkflows, nil
}
