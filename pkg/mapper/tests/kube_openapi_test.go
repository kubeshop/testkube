package tests

import (
	"fmt"
	"testing"
	"time"

	testsv1 "github.com/kubeshop/testkube-operator/apis/tests/v1"
)

func TestMapTestListKubeToAPI(t *testing.T) {

	openAPITest := MapCRToAPI(testsv1.Test{Spec: testsv1.TestSpec{
		Before: []testsv1.TestStepSpec{
			testsv1.TestStepSpec{
				DelayStep: &testsv1.TestStepDelay{
					Duration: time.Second,
				},
			},
		},

		Steps: []testsv1.TestStepSpec{
			testsv1.TestStepSpec{
				ScriptStep: &testsv1.TestStepExecuteScript{
					Namespace: "testkube",
					Name:      "some-test-name",
				},
			},
		},

		After: []testsv1.TestStepSpec{
			testsv1.TestStepSpec{
				DelayStep: &testsv1.TestStepDelay{
					Duration: time.Second,
				},
			},
		},

		Repeats: 2,
	},
	})

	fmt.Printf("%+v\n", openAPITest)

	t.Fail()

}
