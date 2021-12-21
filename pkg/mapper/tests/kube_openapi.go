package tests

import (
	"fmt"

	testsv1 "github.com/kubeshop/testkube-operator/apis/tests/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func MapTestListKubeToAPI(cr testsv1.TestList) (tests []testkube.Test) {
	for _, item := range cr.Items {
		tests = append(tests, MapCRToAPI(item))
	}

	return
}

func MapCRToAPI(cr testsv1.Test) (test testkube.Test) {
	test.Name = cr.Name
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

func mapCRStepToAPI(crstep testsv1.TestStepSpec) (teststep testkube.TestStep) {

	switch true {
	case crstep.Execute != nil:
		teststep = testkube.TestStepExecuteScript{
			Name:              crstep.Execute.Name,
			Namespace:         crstep.Execute.Namespace,
			StopTestOnFailure: crstep.Execute.StopOnFailure,
			Type_:             string(testkube.EXECUTE_SCRIPT_TestStepType),
		}

	case crstep.Delay != nil:
		teststep = testkube.TestStepDelay{
			Name:              fmt.Sprintf("Delay %dms", crstep.Delay.Duration),
			Duration:          crstep.Delay.Duration,
			Type_:             string(testkube.DELAY_TestStepType),
			StopTestOnFailure: false,
		}
	}

	return
}
