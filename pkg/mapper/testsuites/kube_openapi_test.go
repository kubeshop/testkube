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
						Batch: []testsuitesv3.TestSuiteStepSpec{
							{
								Delay: &testsuitesv3.TestSuiteStepDelay{
									Duration: 1000,
								},
							},
						},
					},
				},

				Steps: []testsuitesv3.TestSuiteBatchStep{
					{
						Batch: []testsuitesv3.TestSuiteStepSpec{
							{
								Execute: &testsuitesv3.TestSuiteStepExecute{
									Namespace: "testkube",
									Name:      "some-test-name",
								},
							},
						},
					},
				},

				After: []testsuitesv3.TestSuiteBatchStep{
					{
						Batch: []testsuitesv3.TestSuiteStepSpec{
							{
								Delay: &testsuitesv3.TestSuiteStepDelay{
									Duration: 1000,
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
