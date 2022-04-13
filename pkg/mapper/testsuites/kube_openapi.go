package testsuites

import (
	testsuitesv1 "github.com/kubeshop/testkube-operator/apis/testsuite/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// MapTestSuiteListKubeToAPI maps TestSuiteList CRD to list of OpenAPI spec TestSuite
func MapTestSuiteListKubeToAPI(cr testsuitesv1.TestSuiteList) (tests []testkube.TestSuite) {
	tests = make([]testkube.TestSuite, len(cr.Items))
	for i, item := range cr.Items {
		tests[i] = MapCRToAPI(item)
	}

	return
}

// MapCRToAPI maps TestSuite CRD to OpenAPI spec TestSuite
func MapCRToAPI(cr testsuitesv1.TestSuite) (test testkube.TestSuite) {
	test.Name = cr.Name
	test.Namespace = cr.Namespace
	test.Description = cr.Spec.Description

	for _, s := range cr.Spec.Before {
		test.Before = append(test.Before, mapCRStepToAPI(s))
	}
	for _, s := range cr.Spec.Steps {
		test.Steps = append(test.Steps, mapCRStepToAPI(s))
	}
	for _, s := range cr.Spec.After {
		test.After = append(test.After, mapCRStepToAPI(s))
	}

	test.Description = cr.Spec.Description
	test.Repeats = int32(cr.Spec.Repeats)
	test.Labels = cr.Labels
	test.Schedule = cr.Spec.Schedule
	test.Params = cr.Spec.Params

	return
}

// mapCRStepToAPI maps CRD TestSuiteStepSpec to OpenAPI spec TestSuiteStep
func mapCRStepToAPI(crstep testsuitesv1.TestSuiteStepSpec) (teststep testkube.TestSuiteStep) {

	switch true {
	case crstep.Execute != nil:
		teststep = testkube.TestSuiteStep{
			StopTestOnFailure: crstep.Execute.StopOnFailure,
			Execute: &testkube.TestSuiteStepExecuteTest{
				Name:      crstep.Execute.Name,
				Namespace: crstep.Execute.Namespace,
			},
		}

	case crstep.Delay != nil:
		teststep = testkube.TestSuiteStep{
			Delay: &testkube.TestSuiteStepDelay{
				Duration: crstep.Delay.Duration,
			},
		}
	}

	return
}
