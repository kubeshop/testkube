package tests

import (
	"testing"

	testsv1 "github.com/kubeshop/testkube-operator/apis/tests/v1"
	"github.com/stretchr/testify/assert"
)

func TestMapTestListKubeToAPI(t *testing.T) {

	openAPITest := MapCRToAPI(
		testsv1.Test{
			Spec: testsv1.TestSpec{
				Before: []testsv1.TestStepSpec{
					{
						Delay: &testsv1.TestStepDelay{
							Duration: 1000,
						},
					},
				},

				Steps: []testsv1.TestStepSpec{
					{
						Execute: &testsv1.TestStepExecute{
							Namespace: "testkube",
							Name:      "some-test-name",
						},
					},
				},

				After: []testsv1.TestStepSpec{
					{
						Delay: &testsv1.TestStepDelay{
							Duration: 1000,
						},
					},
				},

				Repeats: 2,
			},
		},
	)

	assert.Equal(t, 1, len(openAPITest.Steps))
	assert.Equal(t, 1, len(openAPITest.Before))
	assert.Equal(t, 1, len(openAPITest.After))
}
