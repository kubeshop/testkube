package triggers

import (
	"context"
	"fmt"
	testsv3 "github.com/kubeshop/testkube-operator/apis/tests/v3"
	testsuitesv2 "github.com/kubeshop/testkube-operator/apis/testsuite/v2"
	testtriggersv1 "github.com/kubeshop/testkube-operator/apis/testtriggers/v1"
	v1 "github.com/kubeshop/testkube/internal/app/api/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/workerpool"
	"github.com/pkg/errors"
	"strconv"
)

const (
	ExecutionTest      = "test"
	ExecutionTestSuite = "testsuite"
)

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
		go wp.SendRequests(s.tk.PrepareTestRequests(tests, request))
		go wp.Run(ctx)

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
		go wp.SendRequests(s.tk.PrepareTestSuiteRequests(testSuites, request))
		go wp.Run(ctx)

		for r := range wp.GetResponses() {
			status.addTestSuiteExecutionID(r.Result.Id)
			results = append(results, r.Result)
		}
	default:
		return errors.Errorf("invalid execution: %s", t.Spec.Execution)
	}

	status.start()
	s.l.Debugf("trigger service: executor component: started test execution for trigger %s/%s", t.Namespace, t.Name)

	return nil
}

func (s *Service) getTests(t *testtriggersv1.TestTrigger) ([]testsv3.Test, error) {
	var tests []testsv3.Test
	if t.Spec.TestSelector.Name != "" {
		s.l.Debugf("trigger service: executor component: fetching testsv3.Test with name %s", t.Spec.TestSelector.Name)
		test, err := s.tc.Get(t.Spec.TestSelector.Name)
		if err != nil {
			return nil, err
		}
		tests = append(tests, *test)
	}
	if len(t.Spec.TestSelector.Labels) > 0 {
		s.l.Debugf("trigger service: executor component: fetching testsv3.Test with labels %v", t.Spec.TestSelector.Labels)
		selectors := labelsToSelector(t.Spec.TestSelector.Labels)
		testList, err := s.tc.List(selectors)
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
		s.l.Debugf("trigger service: executor component: fetching testsuitesv2.TestSuite with name %s", t.Spec.TestSelector.Name)
		testSuite, err := s.tsc.Get(t.Spec.TestSelector.Name)
		if err != nil {
			return nil, err
		}
		testSuites = append(testSuites, *testSuite)
	}
	if len(t.Spec.TestSelector.Labels) > 0 {
		s.l.Debugf("trigger service: executor component: fetching testsuitesv2.TestSuite with label %v", t.Spec.TestSelector.Labels)
		selectors := labelsToSelector(t.Spec.TestSelector.Labels)
		testSuitesList, err := s.tsc.List(selectors)
		if err != nil {
			return nil, err
		}
		testSuites = append(testSuites, testSuitesList.Items...)
	}
	return testSuites, nil
}

func labelsToSelector(labels map[string]string) string {
	stringified := ""
	for k, v := range labels {
		labelAsString := fmt.Sprintf("%s=%s", k, v)
		if stringified == "" {
			stringified += labelAsString
		} else {
			stringified += fmt.Sprintf(",%s", labelAsString)
		}
	}
	return stringified
}
