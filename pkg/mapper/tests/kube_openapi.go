package tests

import (
	"fmt"

	testsv1 "github.com/kubeshop/testkube-operator/apis/tests/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func MapTestListKubeToAPI(cr testsv1.TestList) (tests []testkube.Test) {
	tests = make([]testkube.Test, len(cr.Items))
	for i, item := range cr.Items {
		tests[i] = MapCRToAPI(item)
	}

	return
}

func MapCRToAPI(cr testsv1.Test) (test testkube.Test) {
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

func mapCRStepToAPI(crstep testsv1.TestStepSpec) (teststep testkube.TestStep) {

	switch true {
	case crstep.Execute != nil:
		teststep = testkube.TestStep{
			Type: testkube.EXECUTE_SCRIPT_TestStepType,
			TestStepExecuteScript: testkube.TestStepExecuteScript{
				Name:              crstep.Execute.Name,
				Namespace:         crstep.Execute.Namespace,
				StopTestOnFailure: crstep.Execute.StopOnFailure,
			},
		}

	case crstep.Delay != nil:
		fmt.Printf("DURATION: %+v\n", crstep.Delay.Duration)

		teststep = testkube.TestStep{
			Type: testkube.DELAY_TestStepType,
			TestStepDelay: testkube.TestStepDelay{
				Name:     fmt.Sprintf("Delay %dms", crstep.Delay.Duration),
				Duration: crstep.Delay.Duration,
			},
		}
	}

	return
}
