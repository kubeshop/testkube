package testsuites

import (
	"testing"

	"github.com/stretchr/testify/assert"

	testsuitesv2 "github.com/kubeshop/testkube-operator/apis/testsuite/v2"
)

func TestMapTestSuiteListKubeToAPI(t *testing.T) {

	openAPITest := MapCRToAPI(
		testsuitesv2.TestSuite{
			Spec: testsuitesv2.TestSuiteSpec{
				Before: []testsuitesv2.TestSuiteStepSpec{
					{
						Delay: &testsuitesv2.TestSuiteStepDelay{
							Duration: 1000,
						},
					},
				},

				Steps: []testsuitesv2.TestSuiteStepSpec{
					{
						Execute: &testsuitesv2.TestSuiteStepExecute{
							Namespace: "testkube",
							Name:      "some-test-name",
						},
					},
				},

				After: []testsuitesv2.TestSuiteStepSpec{
					{
						Delay: &testsuitesv2.TestSuiteStepDelay{
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
