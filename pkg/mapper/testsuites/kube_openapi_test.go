package testsuites

import (
	"testing"

	testsuitesv1 "github.com/kubeshop/testkube-operator/apis/testsuite/v1"
	"github.com/stretchr/testify/assert"
)

func TestMapTestSuiteListKubeToAPI(t *testing.T) {

	openAPITest := MapCRToAPI(
		testsuitesv1.TestSuite{
			Spec: testsuitesv1.TestSuiteSpec{
				Before: []testsuitesv1.TestSuiteStepSpec{
					{
						Delay: &testsuitesv1.TestSuiteStepDelay{
							Duration: 1000,
						},
					},
				},

				Steps: []testsuitesv1.TestSuiteStepSpec{
					{
						Execute: &testsuitesv1.TestSuiteStepExecute{
							Namespace: "testkube",
							Name:      "some-test-name",
						},
					},
				},

				After: []testsuitesv1.TestSuiteStepSpec{
					{
						Delay: &testsuitesv1.TestSuiteStepDelay{
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
