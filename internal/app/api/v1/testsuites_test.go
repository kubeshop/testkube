package v1

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	testsuitesv2 "github.com/kubeshop/testkube-operator/apis/testsuite/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/testkube/internal/pkg/api/mock"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/server"
	"github.com/stretchr/testify/assert"
)

func TestTestkubeAPI_GetTestSuiteWithExecutionHandler(t *testing.T) {
	app := fiber.New()
	route := "/test-suite-with-executions"
	resultRepo := mock.TestExecutionRepository{}
	testSuitesClient := &mock.TestSuitesClient{}
	s := &TestkubeAPI{
		HTTPServer: server.HTTPServer{
			Mux: app,
			Log: log.DefaultLogger,
		},
		TestExecutionResults: &resultRepo,
		TestsSuitesClient:    testSuitesClient,
	}
	app.Get(route, s.ListTestSuiteWithExecutionsHandler())

	tests := []struct {
		name           string
		expectedCode   int
		testSuitesList *testsuitesv2.TestSuiteList
		executionsList []testkube.TestSuiteExecution
		want           []testkube.TestSuiteWithExecution
	}{
		{
			name:           "No test suites should return empty list",
			expectedCode:   200,
			testSuitesList: &testsuitesv2.TestSuiteList{},
			executionsList: []testkube.TestSuiteExecution{},
			want:           []testkube.TestSuiteWithExecution{},
		},
		{
			name:           "Test suites with executions should be returned fully",
			expectedCode:   200,
			testSuitesList: getExampleTestSuiteList(),
			executionsList: getExampleTestSuiteExecution(),
			want:           getExampleTestSuiteWithExecutionList(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testSuitesClient.ListFn = func(selector string) (*testsuitesv2.TestSuiteList, error) {
				return tt.testSuitesList, nil
			}
			resultRepo.GetLatestByTestSuitesFn = func(ctx context.Context, testSuiteNames []string, sortField string) (executions []testkube.TestSuiteExecution, err error) {
				return tt.executionsList, nil
			}

			req := httptest.NewRequest("GET", route, nil)
			resp, err := app.Test(req, -1)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedCode, resp.StatusCode, tt.name)

			var res []testkube.TestSuiteWithExecution
			err = json.NewDecoder(resp.Body).Decode(&res)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, res)
		})
	}
}

func getExampleTestSuiteList() *testsuitesv2.TestSuiteList {
	return &testsuitesv2.TestSuiteList{
		Items: []testsuitesv2.TestSuite{
			{
				TypeMeta: metav1.TypeMeta{
					Kind:       "test-suite-crd",
					APIVersion: "test.0.0",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-suite-example",
					Namespace: "testkube",
				},
				Spec: testsuitesv2.TestSuiteSpec{
					Before:           []testsuitesv2.TestSuiteStepSpec{},
					Steps:            []testsuitesv2.TestSuiteStepSpec{},
					After:            []testsuitesv2.TestSuiteStepSpec{},
					Repeats:          0,
					Description:      "test description",
					Schedule:         "* * * * *",
					ExecutionRequest: &testsuitesv2.TestSuiteExecutionRequest{},
				},
				Status: testsuitesv2.TestSuiteStatus{},
			},
		},
	}
}

func getExampleTestSuiteExecution() []testkube.TestSuiteExecution {
	return []testkube.TestSuiteExecution{
		{
			Id:         "123",
			Name:       "test-suite-test-execution",
			TestSuite:  &testkube.ObjectRef{Namespace: "testkube", Name: "test-suite-example"},
			Status:     testkube.TestSuiteExecutionStatusPassed,
			Envs:       map[string]string{},
			Variables:  map[string]testkube.Variable{},
			SecretUUID: "",
			StartTime:  time.Time{},
			EndTime:    time.Time{},
			Duration:   "10s",
			StepResults: []testkube.TestSuiteStepExecutionResult{
				{
					Step: &testkube.TestSuiteStep{
						StopTestOnFailure: false,
						Execute: &testkube.TestSuiteStepExecuteTest{
							Namespace: "testkube",
							Name:      "test1",
						},
					},
					Execution: &testkube.Execution{
						Id:            "123",
						TestName:      "test1",
						TestSuiteName: "test-suite-example",
						TestType:      "postman/collection",
						Name:          "test-suite-example-test1-1",
						Number:        1,
						StartTime:     time.Date(2021, time.Month(2), 21, 1, 10, 30, 0, time.UTC),
						EndTime:       time.Date(2021, time.Month(2), 22, 1, 10, 30, 0, time.UTC),
					},
				},
			},
			Labels: map[string]string{},
		},
	}
}

func getExampleTestSuiteWithExecutionList() []testkube.TestSuiteWithExecution {
	return []testkube.TestSuiteWithExecution{
		{
			TestSuite: &testkube.TestSuite{
				Name:             "test-suite-example",
				Namespace:        "testkube",
				Description:      "test description",
				Schedule:         "* * * * *",
				Repeats:          0,
				ExecutionRequest: &testkube.TestSuiteExecutionRequest{},
			},
			LatestExecution: &testkube.TestSuiteExecution{
				Id:   "123",
				Name: "test-suite-test-execution",
				TestSuite: &testkube.ObjectRef{
					Namespace: "testkube",
					Name:      "test-suite-example",
				},
				Status:   testkube.TestSuiteExecutionStatusPassed,
				Duration: "10s",
				StepResults: []testkube.TestSuiteStepExecutionResult{
					{
						Step: &testkube.TestSuiteStep{
							StopTestOnFailure: false,
							Execute: &testkube.TestSuiteStepExecuteTest{
								Namespace: "testkube",
								Name:      "test1",
							},
						},
						Execution: &testkube.Execution{
							Id:            "123",
							TestName:      "test1",
							TestSuiteName: "test-suite-example",
							TestType:      "postman/collection",
							Name:          "test-suite-example-test1-1",
							Number:        1,
							StartTime:     time.Date(2021, time.Month(2), 21, 1, 10, 30, 0, time.UTC),
							EndTime:       time.Date(2021, time.Month(2), 22, 1, 10, 30, 0, time.UTC),
						},
					},
				},
			},
		},
	}
}
