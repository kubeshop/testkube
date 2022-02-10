package testsuites

import (
	testsv1 "github.com/kubeshop/testkube-operator/apis/tests/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func MapTestListKubeToAPI(cr testsv1.TestList) (tests []testkube.TestSuite) {
	tests = make([]testkube.TestSuite, len(cr.Items))
	for i, item := range cr.Items {
		tests[i] = MapCRToAPI(item)
	}

	return
}

func MapCRToAPI(cr testsv1.Test) (test testkube.TestSuite) {
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

	return
}

func mapCRStepToAPI(crstep testsv1.TestStepSpec) (teststep testkube.TestSuiteStep) {

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
