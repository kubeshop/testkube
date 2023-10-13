package testsuites

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testsuitesv3 "github.com/kubeshop/testkube-operator/api/testsuite/v3"
)

func TestMapTestSuiteListKubeToAPI(t *testing.T) {

	openAPITest := MapCRToAPI(
		testsuitesv3.TestSuite{
			Spec: testsuitesv3.TestSuiteSpec{
				Before: []testsuitesv3.TestSuiteBatchStep{
					{
						Execute: []testsuitesv3.TestSuiteStepSpec{
							{
								Delay: metav1.Duration{Duration: 2 * time.Second},
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
								Delay: metav1.Duration{Duration: time.Second},
							},
						},
					},
				},

				Repeats: 2,
			},
		},
	)

	assert.Equal(t, 1, len(openAPITest.Before))
	assert.Equal(t, "2s", openAPITest.Before[0].Execute[0].Delay)
	assert.Equal(t, 1, len(openAPITest.Steps))
	assert.Equal(t, "some-test-name", openAPITest.Steps[0].Execute[0].Test)
	assert.Equal(t, 1, len(openAPITest.After))
	assert.Equal(t, "1s", openAPITest.After[0].Execute[0].Delay)
}

func TestMapTestSuiteTestCRDToUpdateRequest(t *testing.T) {

	openAPITest := MapTestSuiteTestCRDToUpdateRequest(
		&testsuitesv3.TestSuite{
			Spec: testsuitesv3.TestSuiteSpec{
				Before: []testsuitesv3.TestSuiteBatchStep{
					{
						Execute: []testsuitesv3.TestSuiteStepSpec{
							{
								Delay: metav1.Duration{Duration: 2 * time.Second},
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
								Delay: metav1.Duration{Duration: time.Second},
							},
						},
					},
				},

				Repeats: 2,
			},
		},
	)

	assert.Equal(t, 1, len(*openAPITest.Before))
	assert.Equal(t, "2s", (*openAPITest.Before)[0].Execute[0].Delay)
	assert.Equal(t, 1, len(*openAPITest.Steps))
	assert.Equal(t, "some-test-name", (*openAPITest.Steps)[0].Execute[0].Test)
	assert.Equal(t, 1, len(*openAPITest.After))
	assert.Equal(t, "1s", (*openAPITest.After)[0].Execute[0].Delay)
}
