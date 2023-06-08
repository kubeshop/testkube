package testsuites

import (
	"testing"

	testsuitesv3 "github.com/kubeshop/testkube-operator/apis/testsuite/v3"

	"github.com/stretchr/testify/assert"
)

func TestMapTestSuiteListKubeToAPI(t *testing.T) {

	openAPITest := MapCRToAPI(
		testsuitesv3.TestSuite{
			Spec: testsuitesv3.TestSuiteSpec{
				Before: []testsuitesv3.TestSuiteBatchStep{
					{
						Execute: []testsuitesv3.TestSuiteStepSpec{
							{
								Delay: &testsuitesv3.TestSuiteStepDelay{
									Duration: "1s",
								},
							},
						},
					},
				},

				Steps: []testsuitesv3.TestSuiteBatchStep{
					{
						Execute: []testsuitesv3.TestSuiteStepSpec{
							{
								Test: &testsuitesv3.TestSuiteStepExecute{
									Namespace: "testkube",
									Name:      "some-test-name",
								},
							},
						},
					},
				},

				After: []testsuitesv3.TestSuiteBatchStep{
					{
						Execute: []testsuitesv3.TestSuiteStepSpec{
							{
								Delay: &testsuitesv3.TestSuiteStepDelay{
									Duration: "1s",
								},
							},
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
