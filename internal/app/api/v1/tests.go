package v1

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/kubeshop/testkube/pkg/repository/result"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/mongo"

	testsv3 "github.com/kubeshop/testkube-operator/apis/tests/v3"
	"github.com/kubeshop/testkube-operator/client/tests/v3"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/crd"
	"github.com/kubeshop/testkube/pkg/executor/client"
	executionsmapper "github.com/kubeshop/testkube/pkg/mapper/executions"
	testsmapper "github.com/kubeshop/testkube/pkg/mapper/tests"

	"k8s.io/apimachinery/pkg/api/errors"
)

// GetTestHandler is method for getting an existing test
func (s TestkubeAPI) GetTestHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("id")
		crTest, err := s.TestsClient.Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, err)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		test := testsmapper.MapTestCRToAPI(*crTest)
		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			test.QuoteTestTextFields()
			data, err := crd.GenerateYAML(crd.TemplateTest, []testkube.Test{test})
			return s.getCRDs(c, data, err)
		}

		return c.JSON(test)
	}
}

// GetTestWithExecutionHandler is method for getting an existing test with execution
func (s TestkubeAPI) GetTestWithExecutionHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("id")
		crTest, err := s.TestsClient.Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, err)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		test := testsmapper.MapTestCRToAPI(*crTest)
		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			test.QuoteTestTextFields()
			data, err := crd.GenerateYAML(crd.TemplateTest, []testkube.Test{test})
			return s.getCRDs(c, data, err)
		}

		ctx := c.Context()
		startExecution, startErr := s.ExecutionResults.GetLatestByTest(ctx, name, "starttime")
		if startErr != nil && startErr != mongo.ErrNoDocuments {
			return s.Error(c, http.StatusInternalServerError, startErr)
		}

		endExecution, endErr := s.ExecutionResults.GetLatestByTest(ctx, name, "endtime")
		if endErr != nil && endErr != mongo.ErrNoDocuments {
			return s.Error(c, http.StatusInternalServerError, endErr)
		}

		testWithExecution := testkube.TestWithExecution{
			Test: &test,
		}
		if startErr == nil && endErr == nil {
			if startExecution.StartTime.After(endExecution.EndTime) {
				testWithExecution.LatestExecution = &startExecution
			} else {
				testWithExecution.LatestExecution = &endExecution
			}
		} else if startErr == nil {
			testWithExecution.LatestExecution = &startExecution
		} else if endErr == nil {
			testWithExecution.LatestExecution = &endExecution
		}

		return c.JSON(testWithExecution)
	}
}

func (s TestkubeAPI) getFilteredTestList(c *fiber.Ctx) (*testsv3.TestList, error) {

	crTests, err := s.TestsClient.List(c.Query("selector"))
	if err != nil {
		return nil, err
	}

	search := c.Query("textSearch")
	if search != "" {
		// filter items array
		for i := len(crTests.Items) - 1; i >= 0; i-- {
			if !strings.Contains(crTests.Items[i].Name, search) {
				crTests.Items = append(crTests.Items[:i], crTests.Items[i+1:]...)
			}
		}
	}

	testType := c.Query("type")
	if testType != "" {
		// filter items array
		for i := len(crTests.Items) - 1; i >= 0; i-- {
			if !strings.Contains(crTests.Items[i].Spec.Type_, testType) {
				crTests.Items = append(crTests.Items[:i], crTests.Items[i+1:]...)
			}
		}
	}

	return crTests, nil
}

// ListTestsHandler is a method for getting list of all available tests
func (s TestkubeAPI) ListTestsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		crTests, err := s.getFilteredTestList(c)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		tests := testsmapper.MapTestListKubeToAPI(*crTests)
		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			for i := range tests {
				tests[i].QuoteTestTextFields()
			}

			data, err := crd.GenerateYAML(crd.TemplateTest, tests)
			return s.getCRDs(c, data, err)
		}

		return c.JSON(tests)
	}
}

// ListTestsHandler is a method for getting list of all available tests
func (s TestkubeAPI) TestMetricsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		testName := c.Params("id")

		const DefaultLimit = 0
		limit, err := strconv.Atoi(c.Query("limit", strconv.Itoa(DefaultLimit)))
		if err != nil {
			limit = DefaultLimit
		}

		const DefaultLastDays = 7
		last, err := strconv.Atoi(c.Query("last", strconv.Itoa(DefaultLastDays)))
		if err != nil {
			last = DefaultLastDays
		}

		metrics, err := s.ExecutionResults.GetTestMetrics(context.Background(), testName, limit, last)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		return c.JSON(metrics)
	}
}

// getLatestExecutions return latest executions either by starttime or endtime for tests
func (s TestkubeAPI) getLatestExecutions(ctx context.Context, testNames []string) (map[string]testkube.Execution, error) {
	executions, err := s.ExecutionResults.GetLatestByTests(ctx, testNames, "starttime")
	if err != nil && err != mongo.ErrNoDocuments {
		return nil, err
	}

	startExecutionMap := make(map[string]testkube.Execution, len(executions))
	for i := range executions {
		startExecutionMap[executions[i].TestName] = executions[i]
	}

	executions, err = s.ExecutionResults.GetLatestByTests(ctx, testNames, "endtime")
	if err != nil && err != mongo.ErrNoDocuments {
		return nil, err
	}

	endExecutionMap := make(map[string]testkube.Execution, len(executions))
	for i := range executions {
		endExecutionMap[executions[i].TestName] = executions[i]
	}

	executionMap := make(map[string]testkube.Execution)
	for _, testName := range testNames {
		startExecution, okStart := startExecutionMap[testName]
		endExecution, okEnd := endExecutionMap[testName]
		if !okStart && !okEnd {
			continue
		}

		if okStart && !okEnd {
			executionMap[testName] = startExecution
			continue
		}

		if !okStart && okEnd {
			executionMap[testName] = endExecution
			continue
		}

		if startExecution.StartTime.After(endExecution.EndTime) {
			executionMap[testName] = startExecution
		} else {
			executionMap[testName] = endExecution
		}
	}

	return executionMap, nil
}

// ListTestWithExecutionsHandler is a method for getting list of all available test with latest executions
func (s TestkubeAPI) ListTestWithExecutionsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		crTests, err := s.getFilteredTestList(c)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		tests := testsmapper.MapTestListKubeToAPI(*crTests)
		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			for i := range tests {
				tests[i].QuoteTestTextFields()
			}

			data, err := crd.GenerateYAML(crd.TemplateTest, tests)
			return s.getCRDs(c, data, err)
		}

		ctx := c.Context()
		results := make([]testkube.TestWithExecutionSummary, 0, len(tests))
		testNames := make([]string, len(tests))
		for i := range tests {
			testNames[i] = tests[i].Name
		}

		executionMap, err := s.getLatestExecutions(ctx, testNames)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, err)
		}

		for i := range tests {
			if execution, ok := executionMap[tests[i].Name]; ok {
				results = append(results, testkube.TestWithExecutionSummary{
					Test:            &tests[i],
					LatestExecution: executionsmapper.MapToSummary(&execution),
				})
			} else {
				results = append(results, testkube.TestWithExecutionSummary{
					Test: &tests[i],
				})
			}
		}

		sort.Slice(results, func(i, j int) bool {
			iTime := results[i].Test.Created
			if results[i].LatestExecution != nil {
				iTime = results[i].LatestExecution.EndTime
				if results[i].LatestExecution.StartTime.After(results[i].LatestExecution.EndTime) {
					iTime = results[i].LatestExecution.StartTime
				}
			}

			jTime := results[j].Test.Created
			if results[j].LatestExecution != nil {
				jTime = results[j].LatestExecution.EndTime
				if results[j].LatestExecution.StartTime.After(results[j].LatestExecution.EndTime) {
					jTime = results[j].LatestExecution.StartTime
				}
			}

			return iTime.After(jTime)
		})

		status := c.Query("status")
		if status != "" {
			statusList, err := testkube.ParseExecutionStatusList(status, ",")
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("execution status filter invalid: %w", err))
			}

			statusMap := statusList.ToMap()
			// filter items array
			for i := len(results) - 1; i >= 0; i-- {
				if results[i].LatestExecution != nil && results[i].LatestExecution.Status != nil {
					if _, ok := statusMap[*results[i].LatestExecution.Status]; ok {
						continue
					}
				}

				results = append(results[:i], results[i+1:]...)
			}
		}

		var page, pageSize int
		pageParam := c.Query("page", "")
		if pageParam != "" {
			pageSize = result.PageDefaultLimit
			page, err = strconv.Atoi(pageParam)
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("test page filter invalid: %w", err))
			}
		}

		pageSizeParam := c.Query("pageSize", "")
		if pageSizeParam != "" {
			pageSize, err = strconv.Atoi(pageSizeParam)
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("test page size filter invalid: %w", err))
			}
		}

		if pageParam != "" || pageSizeParam != "" {
			startPos := page * pageSize
			endPos := (page + 1) * pageSize
			if startPos < len(results) {
				if endPos > len(results) {
					endPos = len(results)
				}

				results = results[startPos:endPos]
			}
		}

		return c.JSON(results)
	}
}

// CreateTestHandler creates new test CR based on test content
func (s TestkubeAPI) CreateTestHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {

		var request testkube.TestUpsertRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		if request.ExecutionRequest != nil && request.ExecutionRequest.Args != nil {
			request.ExecutionRequest.Args, err = testkube.PrepareExecutorArgs(request.ExecutionRequest.Args)
			if err != nil {
				return s.Error(c, http.StatusBadRequest, err)
			}
		}

		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			request.QuoteTestTextFields()
			data, err := crd.GenerateYAML(crd.TemplateTest, []testkube.TestUpsertRequest{request})
			return s.getCRDs(c, data, err)
		}

		s.Log.Infow("creating test", "request", request)

		test := testsmapper.MapUpsertToSpec(request)
		test.Namespace = s.Namespace
		var secrets map[string]string
		if request.Content != nil && request.Content.Repository != nil {
			secrets = getTestSecretsData(request.Content.Repository.Username, request.Content.Repository.Token)
		}

		createdTest, err := s.TestsClient.Create(test, tests.Option{Secrets: secrets})

		s.Metrics.IncCreateTest(test.Spec.Type_, err)

		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		c.Status(http.StatusCreated)
		return c.JSON(createdTest)
	}
}

// UpdateTestHandler updates an existing test CR based on test content
func (s TestkubeAPI) UpdateTestHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var request testkube.TestUpdateRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		var name string
		if request.Name != nil {
			name = *request.Name
		}

		// we need to get resource first and load its metadata.ResourceVersion
		test, err := s.TestsClient.Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, err)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		if request.ExecutionRequest != nil && *request.ExecutionRequest != nil && (*request.ExecutionRequest).Args != nil {
			*(*request.ExecutionRequest).Args, err = testkube.PrepareExecutorArgs(*(*request.ExecutionRequest).Args)
			if err != nil {
				return s.Error(c, http.StatusBadRequest, err)
			}
		}

		// map update test but load spec only to not override metadata.ResourceVersion
		testSpec := testsmapper.MapUpdateToSpec(request, test)

		s.Log.Infow("updating test", "request", request)

		var option *tests.Option
		if request.Content != nil && (*request.Content) != nil && (*request.Content).Repository != nil && *(*request.Content).Repository != nil {
			username := (*(*request.Content).Repository).Username
			token := (*(*request.Content).Repository).Token
			if username != nil || token != nil {
				var uValue, tValue string
				if username != nil {
					uValue = *username
				}

				if token != nil {
					tValue = *token
				}

				option = &tests.Option{Secrets: getTestSecretsData(uValue, tValue)}
			}
		}

		var updatedTest *testsv3.Test
		if option != nil {
			updatedTest, err = s.TestsClient.Update(testSpec, *option)
		} else {
			updatedTest, err = s.TestsClient.Update(testSpec)
		}

		s.Metrics.IncUpdateTest(updatedTest.Spec.Type_, err)

		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		return c.JSON(updatedTest)
	}
}

// DeleteTestHandler is a method for deleting a test with id
func (s TestkubeAPI) DeleteTestHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("id")
		err := s.TestsClient.Delete(name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Warn(c, http.StatusNotFound, err)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		// delete executions for test
		if err = s.ExecutionResults.DeleteByTest(c.Context(), name); err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		return c.SendStatus(http.StatusNoContent)
	}
}

// DeleteTestsHandler for deleting all tests
func (s TestkubeAPI) DeleteTestsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var err error
		var testNames []string
		selector := c.Query("selector")
		if selector == "" {
			err = s.TestsClient.DeleteAll()
		} else {
			testList, err := s.TestsClient.List(selector)
			if err != nil {
				if !errors.IsNotFound(err) {
					return s.Error(c, http.StatusBadGateway, err)
				}
			} else {
				for _, item := range testList.Items {
					testNames = append(testNames, item.Name)
				}
			}

			err = s.TestsClient.DeleteByLabels(selector)
		}

		if err != nil {
			if errors.IsNotFound(err) {
				return s.Warn(c, http.StatusNotFound, err)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		// delete all executions for tests
		if selector == "" {
			err = s.ExecutionResults.DeleteAll(c.Context())
		} else {
			err = s.ExecutionResults.DeleteByTests(c.Context(), testNames)
		}

		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		return c.SendStatus(http.StatusNoContent)
	}
}

func getTestSecretsData(username, token string) map[string]string {
	if username == "" && token == "" {
		return nil
	}

	data := make(map[string]string, 0)
	if username != "" {
		data[client.GitUsernameSecretName] = username
	}

	if token != "" {
		data[client.GitTokenSecretName] = token
	}

	return data
}
