package v1

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/mongo"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/yaml"

	testsv3 "github.com/kubeshop/testkube-operator/api/tests/v3"
	"github.com/kubeshop/testkube-operator/pkg/client/tests/v3"
	testsclientv3 "github.com/kubeshop/testkube-operator/pkg/client/tests/v3"
	"github.com/kubeshop/testkube-operator/pkg/secret"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/crd"
	"github.com/kubeshop/testkube/pkg/executor/client"
	executionsmapper "github.com/kubeshop/testkube/pkg/mapper/executions"
	testsmapper "github.com/kubeshop/testkube/pkg/mapper/tests"
	"github.com/kubeshop/testkube/pkg/repository/result"
)

// GetTestHandler is method for getting an existing test
func (s TestkubeAPI) GetTestHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("id")
		if name == "" {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("failed to get test: id cannot be empty"))
		}
		errPrefix := fmt.Sprintf("failed to get test %s", name)
		crTest, err := s.TestsClient.Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: client was unable to find test: %w", errPrefix, err))
			}

			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client failed to find test: %w", errPrefix, err))
		}

		test := testsmapper.MapTestCRToAPI(*crTest)
		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			test.QuoteTestTextFields()
			data, err := crd.GenerateYAML(crd.TemplateTest, []testkube.Test{test})
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not build CRD: %w", errPrefix, err))
			}
			return s.getCRDs(c, data, err)
		}

		return c.JSON(test)
	}
}

// GetTestWithExecutionHandler is method for getting an existing test with execution
func (s TestkubeAPI) GetTestWithExecutionHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("id")
		if name == "" {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("failed to get test with execution: id cannot be empty"))
		}
		errPrefix := fmt.Sprintf("failed to get test %s with execution", name)
		crTest, err := s.TestsClient.Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: client failed to find test: %w", errPrefix, err))
			}

			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client failed to find test: %w", errPrefix, err))
		}

		test := testsmapper.MapTestCRToAPI(*crTest)
		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			test.QuoteTestTextFields()
			data, err := crd.GenerateYAML(crd.TemplateTest, []testkube.Test{test})
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not build CRD: %w", errPrefix, err))
			}
			return s.getCRDs(c, data, err)
		}

		ctx := c.Context()
		execution, err := s.ExecutionResults.GetLatestByTest(ctx, name)
		if err != nil && err != mongo.ErrNoDocuments {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: failed to get execution: %w", errPrefix, err))
		}

		return c.JSON(testkube.TestWithExecution{
			Test:            &test,
			LatestExecution: execution,
		})
	}
}

func (s TestkubeAPI) getFilteredTestList(c *fiber.Ctx) (*testsv3.TestList, error) {

	crTests, err := s.TestsClient.List(c.Query("selector"))
	if err != nil {
		return nil, fmt.Errorf("client failed to list tests: %w", err)
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
		errPrefix := "failed to list tests"
		crTests, err := s.getFilteredTestList(c)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: unable to get filtered tests: %w", errPrefix, err))
		}

		tests := testsmapper.MapTestListKubeToAPI(*crTests)
		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			for i := range tests {
				tests[i].QuoteTestTextFields()
			}

			data, err := crd.GenerateYAML(crd.TemplateTest, tests)
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not build CRD: %w", errPrefix, err))
			}
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
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("failed to get metrics for test %s: %w", testName, err))
		}

		return c.JSON(metrics)
	}
}

// getLatestExecutions return latest executions either by starttime or endtime for tests
func (s TestkubeAPI) getLatestExecutions(ctx context.Context, testNames []string) (map[string]testkube.Execution, error) {
	executions, err := s.ExecutionResults.GetLatestByTests(ctx, testNames)
	if err != nil && err != mongo.ErrNoDocuments {
		return nil, fmt.Errorf("could not get latest executions for tests %s sorted by start time: %w", testNames, err)
	}

	executionMap := make(map[string]testkube.Execution, len(executions))
	for i := range executions {
		executionMap[executions[i].TestName] = executions[i]
	}
	return executionMap, nil
}

// ListTestWithExecutionsHandler is a method for getting list of all available test with latest executions
func (s TestkubeAPI) ListTestWithExecutionsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to list tests with executions"
		crTests, err := s.getFilteredTestList(c)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: unable to get filtered tests: %w", errPrefix, err))
		}

		tests := testsmapper.MapTestListKubeToAPI(*crTests)
		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			for i := range tests {
				tests[i].QuoteTestTextFields()
			}

			data, err := crd.GenerateYAML(crd.TemplateTest, tests)
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not build CRD: %w", errPrefix, err))
			}
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
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: could not get latest executions: %w", errPrefix, err))
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
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: execution status filter invalid: %w", errPrefix, err))
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
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: test page filter invalid: %w", errPrefix, err))
			}
		}

		pageSizeParam := c.Query("pageSize", "")
		if pageSizeParam != "" {
			pageSize, err = strconv.Atoi(pageSizeParam)
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: test page size filter invalid: %w", errPrefix, err))
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
		errPrefix := "failed to create test"
		var test *testsv3.Test
		var secrets map[string]string
		if string(c.Request().Header.ContentType()) == mediaTypeYAML {
			test = &testsv3.Test{}
			testSpec := string(c.Body())
			decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewBufferString(testSpec), len(testSpec))
			if err := decoder.Decode(&test); err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse yaml request: %w", errPrefix, err))
			}

			errPrefix = errPrefix + " " + test.Name
		} else {
			var request testkube.TestUpsertRequest
			err := c.BodyParser(&request)
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: failed to unmarshal request: %w", errPrefix, err))
			}

			err = testkube.ValidateUpsertTestRequest(request)
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: invalid test: %w", errPrefix, err))
			}

			errPrefix = errPrefix + " " + request.Name
			if request.ExecutionRequest != nil && request.ExecutionRequest.Args != nil {
				request.ExecutionRequest.Args, err = testkube.PrepareExecutorArgs(request.ExecutionRequest.Args)
				if err != nil {
					return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not prepare executor args: %w", errPrefix, err))
				}
			}

			if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
				request.QuoteTestTextFields()
				data, err := crd.GenerateYAML(crd.TemplateTest, []testkube.TestUpsertRequest{request})
				if err != nil {
					return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not build CRD: %w", errPrefix, err))
				}

				return s.getCRDs(c, data, err)
			}

			s.Log.Infow("creating test", "request", request)

			test = testsmapper.MapUpsertToSpec(request)
			test.Namespace = s.Namespace
			if request.Content != nil && request.Content.Repository != nil && !s.disableSecretCreation {
				secrets = createTestSecretsData(request.Content.Repository.Username, request.Content.Repository.Token)
			}
		}

		createdTest, err := s.TestsClient.Create(test, s.disableSecretCreation, tests.Option{Secrets: secrets})

		s.Metrics.IncCreateTest(test.Spec.Type_, err)

		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not create test: %w", errPrefix, err))
		}

		c.Status(http.StatusCreated)
		return c.JSON(createdTest)
	}
}

// UpdateTestHandler updates an existing test CR based on test content
func (s TestkubeAPI) UpdateTestHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to update test"
		var request testkube.TestUpdateRequest
		if string(c.Request().Header.ContentType()) == mediaTypeYAML {
			var test testsv3.Test
			testSpec := string(c.Body())
			decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewBufferString(testSpec), len(testSpec))
			if err := decoder.Decode(&test); err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse yaml request: %w", errPrefix, err))
			}

			request = testsmapper.MapSpecToUpdate(&test)
		} else {
			err := c.BodyParser(&request)
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse json request: %w", errPrefix, err))
			}
		}

		var name string
		if request.Name != nil {
			name = *request.Name
			errPrefix = errPrefix + " " + name
		}

		err := testkube.ValidateUpdateTestRequest(request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: invalid test: %w", errPrefix, err))
		}

		// we need to get resource first and load its metadata.ResourceVersion
		test, err := s.TestsClient.Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: client could not find test: %w", errPrefix, err))
			}

			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not get test: %w", errPrefix, err))
		}

		if request.ExecutionRequest != nil && *request.ExecutionRequest != nil && (*request.ExecutionRequest).Args != nil {
			*(*request.ExecutionRequest).Args, err = testkube.PrepareExecutorArgs(*(*request.ExecutionRequest).Args)
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not prepare executor arguments: %w", errPrefix, err))
			}
		}

		// map update test but load spec only to not override metadata.ResourceVersion
		testSpec := testsmapper.MapUpdateToSpec(request, test)

		s.Log.Infow("updating test", "request", request)

		var option *tests.Option
		if request.Content != nil && (*request.Content) != nil && (*request.Content).Repository != nil && *(*request.Content).Repository != nil {
			username := (*(*request.Content).Repository).Username
			token := (*(*request.Content).Repository).Token
			if (username != nil || token != nil) && !s.disableSecretCreation {
				data, err := s.SecretClient.Get(secret.GetMetadataName(name, client.SecretTest))
				if err != nil && !errors.IsNotFound(err) {
					return s.Error(c, http.StatusBadGateway, err)
				}

				option = &tests.Option{Secrets: updateTestSecretsData(data, username, token)}
			}
		}

		if isRepositoryEmpty(testSpec.Spec) {
			testSpec.Spec.Content.Repository = nil
		}

		var updatedTest *testsv3.Test
		if option != nil {
			updatedTest, err = s.TestsClient.Update(testSpec, s.disableSecretCreation, *option)
		} else {
			updatedTest, err = s.TestsClient.Update(testSpec, s.disableSecretCreation)
		}

		s.Metrics.IncUpdateTest(testSpec.Spec.Type_, err)

		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not update test %w", errPrefix, err))
		}

		return c.JSON(updatedTest)
	}
}

func isRepositoryEmpty(s testsv3.TestSpec) bool {
	return s.Content != nil &&
		s.Content.Repository != nil &&
		s.Content.Repository.Type_ == "" &&
		s.Content.Repository.Uri == "" &&
		s.Content.Repository.Branch == "" &&
		s.Content.Repository.Path == "" &&
		s.Content.Repository.Commit == "" &&
		s.Content.Repository.WorkingDir == ""
}

// DeleteTestHandler is a method for deleting a test with id
func (s TestkubeAPI) DeleteTestHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("id")
		if name == "" {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("failed to delete test: id cannot be empty"))
		}
		errPrefix := fmt.Sprintf("failed to delete test %s", name)
		err := s.TestsClient.Delete(name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Warn(c, http.StatusNotFound, fmt.Errorf("%s: client could not find test: %w", errPrefix, err))
			}

			if _, ok := err.(*testsclientv3.DeleteDependenciesError); ok {
				return s.Warn(c, http.StatusInternalServerError, fmt.Errorf("client deleted test %s but deleting test dependencies(secrets) returned errors: %w", name, err))
			}

			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: client could not delete test: %w", errPrefix, err))
		}

		skipExecutions := c.Query("skipDeleteExecutions", "")
		if skipExecutions != "true" {
			// delete executions for test
			if err = s.ExecutionResults.DeleteByTest(c.Context(), name); err != nil {
				return s.Warn(c, http.StatusInternalServerError, fmt.Errorf("test %s was deleted but deleting test executions returned error: %w", name, err))
			}
		}

		return c.SendStatus(http.StatusNoContent)
	}
}

// AbortTestHandler is a method for aborting a executions of a test with id
func (s TestkubeAPI) AbortTestHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		name := c.Params("id")
		if name == "" {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("failed to abort test: id cannot be empty"))
		}
		errPrefix := fmt.Sprintf("failed to abort test %s", name)
		filter := result.NewExecutionsFilter().WithTestName(name).WithStatus(string(testkube.RUNNING_ExecutionStatus))
		executions, err := s.ExecutionResults.GetExecutions(ctx, filter)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: executions with name %s not found", errPrefix, name))
			}
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: could not get executions: %w", errPrefix, err))
		}

		var results []testkube.ExecutionResult
		for _, execution := range executions {
			res, errAbort := s.Executor.Abort(ctx, &execution)
			if errAbort != nil {
				s.Log.Errorw("aborting execution failed", "execution", execution, "error", errAbort)
				err = errAbort
			}
			s.Metrics.IncAbortTest(execution.TestType, res.IsFailed())
			results = append(results, *res)
		}

		return c.JSON(results)
	}
}

// DeleteTestsHandler for deleting all tests
func (s TestkubeAPI) DeleteTestsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to delete tests"
		var err error
		var testNames []string
		selector := c.Query("selector")
		if selector == "" {
			err = s.TestsClient.DeleteAll()
		} else {
			var testList *testsv3.TestList
			testList, err = s.TestsClient.List(selector)
			if err != nil {
				if !errors.IsNotFound(err) {
					return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not list tests: %w", errPrefix, err))
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
				return s.Warn(c, http.StatusNotFound, fmt.Errorf("%s: could not find tests: %w", errPrefix, err))
			}

			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: could not delete tests: %w", errPrefix, err))
		}

		// delete all executions for tests
		if selector == "" {
			err = s.ExecutionResults.DeleteAll(c.Context())
		} else {
			err = s.ExecutionResults.DeleteByTests(c.Context(), testNames)
		}

		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: could not delete executions: %w", errPrefix, err))
		}

		return c.SendStatus(http.StatusNoContent)
	}
}

func createTestSecretsData(username, token string) map[string]string {
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

func updateTestSecretsData(data map[string]string, username, token *string) map[string]string {
	if data == nil {
		data = make(map[string]string)
	}

	if username != nil {
		if *username == "" {
			delete(data, client.GitUsernameSecretName)
		} else {
			data[client.GitUsernameSecretName] = *username
		}
	}

	if token != nil {
		if *token == "" {
			delete(data, client.GitTokenSecretName)
		} else {
			data[client.GitTokenSecretName] = *token
		}
	}

	return data
}
