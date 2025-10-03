package deprecatedv1

import (
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	gomock "go.uber.org/mock/gomock"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	testsv3 "github.com/kubeshop/testkube-operator/api/tests/v3"
	testsclientv3 "github.com/kubeshop/testkube-operator/pkg/client/tests/v3"
	"github.com/kubeshop/testkube/cmd/api-server/commons"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
)

func TestDeprecatedTestkubeAPI_DeleteTest(t *testing.T) {
	app := fiber.New()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockDeprecatedClients := commons.NewMockDeprecatedClients(mockCtrl)
	mockDeprecatedClients.EXPECT().Tests().Return(getMockTestClient()).AnyTimes()

	s := &DeprecatedTestkubeAPI{
		Log:               log.DefaultLogger,
		DeprecatedClients: mockDeprecatedClients,
	}

	app.Delete("/tests/:id", s.DeleteTestHandler())

	req := httptest.NewRequest("DELETE", "http://localhost/tests/k6?skipDeleteExecutions=true", nil)
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	defer resp.Body.Close()

}

func TestTestWithExecutions_SortingLogic(t *testing.T) {
	t.Run("should sort by StartTime with EndTime fallback", func(t *testing.T) {
		baseTime := time.Now().Truncate(time.Minute)

		// Create test data with different execution times
		tests := []testkube.Test{
			{Name: "test-old", Created: baseTime.Add(-15 * time.Minute)},
			{Name: "test-recent", Created: baseTime.Add(-12 * time.Minute)},
			{Name: "test-queued", Created: baseTime.Add(-6 * time.Minute)},
		}

		// Create executions with different timing scenarios
		executions := map[string]testkube.ExecutionSummary{
			"test-old": {
				StartTime: baseTime.Add(-10 * time.Minute), // Started 10 minutes ago
				EndTime:   baseTime.Add(-8 * time.Minute),  // Ended 8 minutes ago
			},
			"test-recent": {
				StartTime: baseTime.Add(-5 * time.Minute), // Started 5 minutes ago (more recent)
				EndTime:   baseTime.Add(-3 * time.Minute), // Ended 3 minutes ago
			},
			"test-queued": {
				StartTime: time.Time{},                    // Zero time - not started
				EndTime:   baseTime.Add(-2 * time.Minute), // Ended 2 minutes ago (should use this)
			},
		}

		// Create results as would be done in the handler
		results := make([]testkube.TestWithExecutionSummary, len(tests))
		for i, test := range tests {
			if execution, ok := executions[test.Name]; ok {
				results[i] = testkube.TestWithExecutionSummary{
					Test:            &test,
					LatestExecution: &execution,
				}
			} else {
				results[i] = testkube.TestWithExecutionSummary{
					Test: &test,
				}
			}
		}

		// Apply the same sorting logic as in the handler
		sortTestExecutions(results)

		// Expected order based on effective sort times:
		// test-queued: EndTime -2 min (most recent)
		// test-recent: StartTime -5 min
		// test-old: StartTime -10 min (oldest)
		assert.Equal(t, "test-queued", results[0].Test.Name)
		assert.Equal(t, "test-recent", results[1].Test.Name)
		assert.Equal(t, "test-old", results[2].Test.Name)
	})

	t.Run("should fall back to test Created time when no execution", func(t *testing.T) {
		baseTime := time.Now().Truncate(time.Minute)

		tests := []testkube.Test{
			{Name: "test-with-execution", Created: baseTime.Add(-15 * time.Minute)},
			{Name: "test-no-execution", Created: baseTime.Add(-5 * time.Minute)}, // More recent Created time
		}

		results := []testkube.TestWithExecutionSummary{
			{
				Test: &tests[0],
				LatestExecution: &testkube.ExecutionSummary{
					StartTime: baseTime.Add(-10 * time.Minute),
				},
			},
			{
				Test: &tests[1],
				// No execution
			},
		}

		sortTestExecutions(results)

		// test-no-execution should be first (Created -5 min > StartTime -10 min)
		assert.Equal(t, "test-no-execution", results[0].Test.Name)
		assert.Equal(t, "test-with-execution", results[1].Test.Name)
	})

	t.Run("should prioritize StartTime over EndTime", func(t *testing.T) {
		baseTime := time.Now().Truncate(time.Minute)

		tests := []testkube.Test{
			{Name: "test-a", Created: baseTime.Add(-15 * time.Minute)},
			{Name: "test-b", Created: baseTime.Add(-15 * time.Minute)},
		}

		results := []testkube.TestWithExecutionSummary{
			{
				Test: &tests[0],
				LatestExecution: &testkube.ExecutionSummary{
					StartTime: baseTime.Add(-2 * time.Minute), // More recent StartTime
					EndTime:   baseTime.Add(-8 * time.Minute), // Older EndTime
				},
			},
			{
				Test: &tests[1],
				LatestExecution: &testkube.ExecutionSummary{
					StartTime: baseTime.Add(-5 * time.Minute), // Older StartTime
					EndTime:   baseTime.Add(-1 * time.Minute), // More recent EndTime
				},
			},
		}

		sortTestExecutions(results)

		// test-a should be first because StartTime takes precedence (-2 min > -5 min)
		assert.Equal(t, "test-a", results[0].Test.Name)
		assert.Equal(t, "test-b", results[1].Test.Name)
	})

	t.Run("should use EndTime when StartTime is zero", func(t *testing.T) {
		baseTime := time.Now().Truncate(time.Minute)

		tests := []testkube.Test{
			{Name: "test-a", Created: baseTime.Add(-15 * time.Minute)},
			{Name: "test-b", Created: baseTime.Add(-15 * time.Minute)},
		}

		results := []testkube.TestWithExecutionSummary{
			{
				Test: &tests[0],
				LatestExecution: &testkube.ExecutionSummary{
					StartTime: time.Time{},                    // Zero time
					EndTime:   baseTime.Add(-2 * time.Minute), // More recent EndTime
				},
			},
			{
				Test: &tests[1],
				LatestExecution: &testkube.ExecutionSummary{
					StartTime: time.Time{},                    // Zero time
					EndTime:   baseTime.Add(-5 * time.Minute), // Older EndTime
				},
			},
		}

		sortTestExecutions(results)

		// test-a should be first because its EndTime is more recent
		assert.Equal(t, "test-a", results[0].Test.Name)
		assert.Equal(t, "test-b", results[1].Test.Name)
	})
}

// sortTestExecutions replicates the exact sorting logic from ListTestWithExecutionsHandler
func sortTestExecutions(results []testkube.TestWithExecutionSummary) {
	// Use sort.Slice to match the exact behavior in the handler
	sort.Slice(results, func(i, j int) bool {
		iTime := results[i].Test.Created
		if results[i].LatestExecution != nil {
			iTime = results[i].LatestExecution.StartTime
			// Fallback to EndTime if StartTime is not set (execution hasn't started yet)
			if iTime.IsZero() {
				iTime = results[i].LatestExecution.EndTime
			}
		}

		jTime := results[j].Test.Created
		if results[j].LatestExecution != nil {
			jTime = results[j].LatestExecution.StartTime
			// Fallback to EndTime if StartTime is not set (execution hasn't started yet)
			if jTime.IsZero() {
				jTime = results[j].LatestExecution.EndTime
			}
		}

		return iTime.After(jTime)
	})
}

func getMockTestClient() *testsclientv3.TestsClient {
	scheme := runtime.NewScheme()
	testsv3.AddToScheme(scheme)
	corev1.AddToScheme(scheme)
	initObjects := []k8sclient.Object{
		&testsv3.Test{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Test",
				APIVersion: "tests.testkube.io/v3",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "k6",
				Namespace: "",
			},
			Spec: testsv3.TestSpec{},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(initObjects...).
		Build()

	return testsclientv3.NewClient(fakeClient, "")
}
