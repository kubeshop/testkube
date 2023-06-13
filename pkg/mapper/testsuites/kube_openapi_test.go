package testsuites

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	testsuitesv3 "github.com/kubeshop/testkube-operator/apis/testsuite/v3"
)

func TestMapTestSuiteListKubeToAPI(t *testing.T) {

	openAPITest := MapCRToAPI(
		testsuitesv3.TestSuite{
			Spec: testsuitesv3.TestSuiteSpec{
				Before: []testsuitesv3.TestSuiteBatchStep{
					{
						Execute: []testsuitesv3.TestSuiteStepSpec{
							{
								Delay: time.Second.String(),
							},
						},
					},
				},

				Steps: []testsuitesv3.TestSuiteBatchStep{
					{
						Execute: []testsuitesv3.TestSuiteStepSpec{
							{
								Test: "some-test-name",
							},
						},
					},
				},

				After: []testsuitesv3.TestSuiteBatchStep{
					{
						Execute: []testsuitesv3.TestSuiteStepSpec{
							{
								Delay: time.Second.String(),
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
