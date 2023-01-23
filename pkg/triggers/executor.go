package triggers

import (
	"context"
	"strconv"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pkg/errors"

	testsv3 "github.com/kubeshop/testkube-operator/apis/tests/v3"
	testsuitesv2 "github.com/kubeshop/testkube-operator/apis/testsuite/v2"
	testtriggersv1 "github.com/kubeshop/testkube-operator/apis/testtriggers/v1"
	v1 "github.com/kubeshop/testkube/internal/app/api/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/workerpool"
)

type Execution string

const (
	ExecutionTest      = "test"
	ExecutionTestSuite = "testsuite"
)

type ExecutorF func(context.Context, *testtriggersv1.TestTrigger) error

func (s *Service) execute(ctx context.Context, t *testtriggersv1.TestTrigger) error {
	status := s.getStatusForTrigger(t)

	concurrencyLevel, err := strconv.Atoi(v1.DefaultConcurrencyLevel)
	if err != nil {
		return errors.Wrap(err, "error parsing default concurrency level")
	}

	switch t.Spec.Execution {
	case ExecutionTest:
		var results []testkube.Execution

		tests, err := s.getTests(t)
		if err != nil {
			return err
		}

		request := testkube.ExecutionRequest{}

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
			results = append(results, r.Result)
		}
	case ExecutionTestSuite:
		var results []testkube.TestSuiteExecution

		testSuites, err := s.getTestSuites(t)
		if err != nil {
			return err
		}

		request := testkube.TestSuiteExecutionRequest{}

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
			results = append(results, r.Result)
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

func (s *Service) getTestSuites(t *testtriggersv1.TestTrigger) ([]testsuitesv2.TestSuite, error) {
	var testSuites []testsuitesv2.TestSuite
	if t.Spec.TestSelector.Name != "" {
		s.logger.Debugf("trigger service: executor component: fetching testsuitesv2.TestSuite with name %s", t.Spec.TestSelector.Name)
		testSuite, err := s.testSuitesClient.Get(t.Spec.TestSelector.Name)
		if err != nil {
			return nil, err
		}
		testSuites = append(testSuites, *testSuite)
	}
	if t.Spec.TestSelector.LabelSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(t.Spec.TestSelector.LabelSelector)
		if err != nil {
			return nil, errors.WithMessagef(err, "error creating selector from test resource label selector")
		}
		stringifiedSelector := selector.String()
		s.logger.Debugf("trigger service: executor component: fetching testsuitesv2.TestSuite with label %s", stringifiedSelector)
		testSuitesList, err := s.testSuitesClient.List(stringifiedSelector)
		if err != nil {
			return nil, err
		}
		testSuites = append(testSuites, testSuitesList.Items...)
	}
	return testSuites, nil
}
