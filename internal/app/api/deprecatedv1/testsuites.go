package deprecatedv1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/mongo"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/yaml"

	testsuitesv3 "github.com/kubeshop/testkube-operator/api/testsuite/v3"
	"github.com/kubeshop/testkube/internal/app/api/apiutils"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/crd"
	"github.com/kubeshop/testkube/pkg/datefilter"
	"github.com/kubeshop/testkube/pkg/event/bus"
	testsmapper "github.com/kubeshop/testkube/pkg/mapper/tests"
	testsuiteexecutionsmapper "github.com/kubeshop/testkube/pkg/mapper/testsuiteexecutions"
	testsuitesmapper "github.com/kubeshop/testkube/pkg/mapper/testsuites"
	"github.com/kubeshop/testkube/pkg/repository/testresult"
	"github.com/kubeshop/testkube/pkg/scheduler"
	"github.com/kubeshop/testkube/pkg/types"
	"github.com/kubeshop/testkube/pkg/utils"
	"github.com/kubeshop/testkube/pkg/workerpool"
)

// CreateTestSuiteHandler for getting test object
func (s *DeprecatedTestkubeAPI) CreateTestSuiteHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to create test suite"
		var testSuite testsuitesv3.TestSuite
		if string(c.Request().Header.ContentType()) == mediaTypeYAML {
			testSuiteSpec := string(c.Body())
			decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewBufferString(testSuiteSpec), len(testSuiteSpec))
			if err := decoder.Decode(&testSuite); err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse yaml request: %w", errPrefix, err))
			}
			errPrefix = errPrefix + " " + testSuite.Name
		} else {
			var request testkube.TestSuiteUpsertRequest
			data := c.Body()
			if string(c.Request().Header.ContentType()) != mediaTypeJSON {
				return s.Error(c, http.StatusBadRequest, fiber.ErrUnprocessableEntity)
			}

			err := json.Unmarshal(data, &request)
			if err != nil {
				s.Log.Warnw("could not parse json request", "error", err)
			}
			errPrefix = errPrefix + " " + request.Name

			emptyBatch := true
			for _, step := range request.Steps {
				if len(step.Execute) != 0 {
					emptyBatch = false
					break
				}
			}

			if emptyBatch {
				var requestV2 testkube.TestSuiteUpsertRequestV2
				if err := json.Unmarshal(data, &requestV2); err != nil {
					return s.Error(c, http.StatusBadRequest, err)
				}

				request = *requestV2.ToTestSuiteUpsertRequest()
			}

			if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
				request.QuoteTestSuiteTextFields()
				data, err := crd.GenerateYAML(crd.TemplateTestSuite, []testkube.TestSuiteUpsertRequest{request})
				return apiutils.SendLegacyCRDs(c, data, err)
			}

			testSuite, err = testsuitesmapper.MapTestSuiteUpsertRequestToTestCRD(request)
			if err != nil {
				return s.Error(c, http.StatusBadRequest, err)
			}

			testSuite.Namespace = s.Namespace
		}

		s.Log.Infow("creating test suite", "testSuite", testSuite)

		created, err := s.DeprecatedClients.TestSuites().Create(&testSuite, !s.secretConfig.AutoCreate)

		s.Metrics.IncCreateTestSuite(err)

		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not create test suite: %w", errPrefix, err))
		}

		c.Status(http.StatusCreated)
		return c.JSON(created)
	}
}

// UpdateTestSuiteHandler updates an existing TestSuite CR based on TestSuite content
func (s *DeprecatedTestkubeAPI) UpdateTestSuiteHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to update test suite"
		var request testkube.TestSuiteUpdateRequest
		if string(c.Request().Header.ContentType()) == mediaTypeYAML {
			var testSuite testsuitesv3.TestSuite
			testSuiteSpec := string(c.Body())
			decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewBufferString(testSuiteSpec), len(testSuiteSpec))
			if err := decoder.Decode(&testSuite); err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse yaml request: %w", errPrefix, err))
			}
			request = testsuitesmapper.MapTestSuiteTestCRDToUpdateRequest(&testSuite)
		} else {
			data := c.Body()
			if string(c.Request().Header.ContentType()) != mediaTypeJSON {
				return s.Error(c, http.StatusBadRequest, fiber.ErrUnprocessableEntity)
			}

			err := json.Unmarshal(data, &request)
			if err != nil {
				s.Log.Warnw("could not parse json request", "error", err)
			}

			if request.Steps != nil {
				emptyBatch := true
				for _, step := range *request.Steps {
					if len(step.Execute) != 0 {
						emptyBatch = false
						break
					}
				}

				if emptyBatch {
					var requestV2 testkube.TestSuiteUpdateRequestV2
					if err := json.Unmarshal(data, &requestV2); err != nil {
						return s.Error(c, http.StatusBadRequest, err)
					}

					request = *requestV2.ToTestSuiteUpdateRequest()
				}
			}
		}

		var name string
		if request.Name != nil {
			name = *request.Name
		}
		errPrefix = errPrefix + " " + name

		// we need to get resource first and load its metadata.ResourceVersion
		testSuite, err := s.DeprecatedClients.TestSuites().Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: test suite not found: %w", errPrefix, err))
			}

			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not get test suite: %w", errPrefix, err))
		}

		// map TestSuite but load spec only to not override metadata.ResourceVersion
		testSuiteSpec, err := testsuitesmapper.MapTestSuiteUpdateRequestToTestCRD(request, testSuite)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		updatedTestSuite, err := s.DeprecatedClients.TestSuites().Update(testSuiteSpec, !s.secretConfig.AutoCreate)

		s.Metrics.IncUpdateTestSuite(err)

		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not update test suite: %w", errPrefix, err))
		}

		return c.JSON(updatedTestSuite)
	}
}

// GetTestSuiteHandler for getting TestSuite object
func (s *DeprecatedTestkubeAPI) GetTestSuiteHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("id")
		errPrefix := "failed to get test suite " + name

		crTestSuite, err := s.DeprecatedClients.TestSuites().Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Warn(c, http.StatusNotFound, fmt.Errorf("%s: test suite not found: %w", errPrefix, err))
			}

			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not get test suite: %w", errPrefix, err))
		}

		testSuite := testsuitesmapper.MapCRToAPI(*crTestSuite)
		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			testSuite.QuoteTestSuiteTextFields()
			data, err := crd.GenerateYAML(crd.TemplateTestSuite, []testkube.TestSuite{testSuite})
			return apiutils.SendLegacyCRDs(c, data, err)
		}

		return c.JSON(testSuite)
	}
}

// GetTestSuiteWithExecutionHandler for getting TestSuite object with execution
func (s *DeprecatedTestkubeAPI) GetTestSuiteWithExecutionHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("id")
		errPrefix := fmt.Sprintf("failed to get test suite %s with execution", name)
		crTestSuite, err := s.DeprecatedClients.TestSuites().Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Warn(c, http.StatusNotFound, fmt.Errorf("%s: test suite not found: %w", errPrefix, err))
			}

			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not get test suite: %w", errPrefix, err))
		}

		testSuite := testsuitesmapper.MapCRToAPI(*crTestSuite)
		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			testSuite.QuoteTestSuiteTextFields()
			data, err := crd.GenerateYAML(crd.TemplateTestSuite, []testkube.TestSuite{testSuite})
			return apiutils.SendLegacyCRDs(c, data, err)
		}

		ctx := c.Context()
		execution, err := s.DeprecatedRepositories.TestSuiteResults().GetLatestByTestSuite(ctx, name)
		if err != nil && err != mongo.ErrNoDocuments {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: could not get execution: %w", errPrefix, err))
		}

		return c.JSON(testkube.TestSuiteWithExecution{
			TestSuite:       &testSuite,
			LatestExecution: execution,
		})
	}
}

// DeleteTestSuiteHandler for deleting a TestSuite with id
func (s *DeprecatedTestkubeAPI) DeleteTestSuiteHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("id")
		errPrefix := fmt.Sprintf("failed to delete test suite %s", name)
		skipCRD := c.Query("skipDeleteCRD", "")
		if skipCRD != "true" {
			err := s.DeprecatedClients.TestSuites().Delete(name)
			if err != nil {
				if errors.IsNotFound(err) {
					return s.Warn(c, http.StatusNotFound, fmt.Errorf("%s: test suite not found: %w", errPrefix, err))
				}

				return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not delete test suite: %w", errPrefix, err))
			}
		}

		// delete executions for test
		if err := s.DeprecatedRepositories.TestResults().DeleteByTestSuite(c.Context(), name); err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not delete test suite test executions: %w", errPrefix, err))
		}

		// delete executions for test suite
		if err := s.DeprecatedRepositories.TestSuiteResults().DeleteByTestSuite(c.Context(), name); err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not delete test suite executions: %w", errPrefix, err))
		}

		return c.SendStatus(http.StatusNoContent)
	}
}

// DeleteTestSuitesHandler for deleting all TestSuites
func (s *DeprecatedTestkubeAPI) DeleteTestSuitesHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to delete test suites"

		var err error
		var testSuiteNames []string
		selector := c.Query("selector")
		if selector == "" {
			err = s.DeprecatedClients.TestSuites().DeleteAll()
		} else {
			var testSuiteList *testsuitesv3.TestSuiteList
			testSuiteList, err = s.DeprecatedClients.TestSuites().List(selector)
			if err != nil {
				if !errors.IsNotFound(err) {
					return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not list test suites: %w", errPrefix, err))
				}
			} else {
				for _, item := range testSuiteList.Items {
					testSuiteNames = append(testSuiteNames, item.Name)
				}
			}

			err = s.DeprecatedClients.TestSuites().DeleteByLabels(selector)
		}

		if err != nil {
			if errors.IsNotFound(err) {
				return s.Warn(c, http.StatusNotFound, fmt.Errorf("%s: test suite not found: %w", errPrefix, err))
			}

			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not delete test suites: %w", errPrefix, err))
		}

		// delete all executions for tests
		if selector == "" {
			err = s.DeprecatedRepositories.TestResults().DeleteForAllTestSuites(c.Context())
		} else {
			err = s.DeprecatedRepositories.TestResults().DeleteByTestSuites(c.Context(), testSuiteNames)
		}

		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not list test suite test executions: %w", errPrefix, err))
		}

		// delete all executions for test suites
		if selector == "" {
			err = s.DeprecatedRepositories.TestSuiteResults().DeleteAll(c.Context())
		} else {
			err = s.DeprecatedRepositories.TestSuiteResults().DeleteByTestSuites(c.Context(), testSuiteNames)
		}

		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not list test suite executions: %w", errPrefix, err))
		}

		return c.SendStatus(http.StatusNoContent)
	}
}

func (s *DeprecatedTestkubeAPI) getFilteredTestSuitesList(c *fiber.Ctx) (*testsuitesv3.TestSuiteList, error) {
	crTestSuites, err := s.DeprecatedClients.TestSuites().List(c.Query("selector"))
	if err != nil {
		return nil, err
	}

	search := c.Query("textSearch")
	if search != "" {
		// filter items array
		for i := len(crTestSuites.Items) - 1; i >= 0; i-- {
			if !strings.Contains(crTestSuites.Items[i].Name, search) {
				crTestSuites.Items = append(crTestSuites.Items[:i], crTestSuites.Items[i+1:]...)
			}
		}
	}

	return crTestSuites, nil
}

// ListTestSuitesHandler for getting list of all available TestSuites
func (s *DeprecatedTestkubeAPI) ListTestSuitesHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to list test suites"

		crTestSuites, err := s.getFilteredTestSuitesList(c)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not list test suites: %w", errPrefix, err))
		}

		testSuites := testsuitesmapper.MapTestSuiteListKubeToAPI(*crTestSuites)
		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			for i := range testSuites {
				testSuites[i].QuoteTestSuiteTextFields()
			}

			data, err := crd.GenerateYAML(crd.TemplateTestSuite, testSuites)
			return apiutils.SendLegacyCRDs(c, data, err)
		}

		return c.JSON(testSuites)
	}
}

// TestSuiteMetricsHandler returns basic metrics for given testsuite
func (s *DeprecatedTestkubeAPI) TestSuiteMetricsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to get test suite metrics"
		const (
			DefaultLastDays = 0
			DefaultLimit    = 0
		)

		testSuiteName := c.Params("id")

		limit, err := strconv.Atoi(c.Query("limit", strconv.Itoa(DefaultLimit)))
		if err != nil {
			limit = DefaultLimit
		}

		last, err := strconv.Atoi(c.Query("last", strconv.Itoa(DefaultLastDays)))
		if err != nil {
			last = DefaultLastDays
		}

		metrics, err := s.DeprecatedRepositories.TestSuiteResults().GetTestSuiteMetrics(context.Background(), testSuiteName, limit, last)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: failed to get metrics from client: %w", errPrefix, err))
		}

		return c.JSON(metrics)
	}
}

// getLatestTestSuiteExecutions return latest test suite executions either by starttime or endtine for tests
func (s *DeprecatedTestkubeAPI) getLatestTestSuiteExecutions(ctx context.Context, testSuiteNames []string) (map[string]testkube.TestSuiteExecution, error) {
	executions, err := s.DeprecatedRepositories.TestSuiteResults().GetLatestByTestSuites(ctx, testSuiteNames)
	if err != nil && err != mongo.ErrNoDocuments {
		return nil, err
	}

	executionMap := make(map[string]testkube.TestSuiteExecution, len(executions))
	for i := range executions {
		if executions[i].TestSuite == nil {
			continue
		}
		executionMap[executions[i].TestSuite.Name] = executions[i]
	}
	return executionMap, nil
}

// ListTestSuiteWithExecutionsHandler for getting list of all available TestSuite with latest executions
func (s *DeprecatedTestkubeAPI) ListTestSuiteWithExecutionsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to list test suites with executions"

		crTestSuites, err := s.getFilteredTestSuitesList(c)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not list test suites: %w", errPrefix, err))
		}

		testSuites := testsuitesmapper.MapTestSuiteListKubeToAPI(*crTestSuites)
		if c.Accepts(mediaTypeJSON, mediaTypeYAML) == mediaTypeYAML {
			for i := range testSuites {
				testSuites[i].QuoteTestSuiteTextFields()
			}

			data, err := crd.GenerateYAML(crd.TemplateTestSuite, testSuites)
			return apiutils.SendLegacyCRDs(c, data, err)
		}

		results := make([]testkube.TestSuiteWithExecutionSummary, 0, len(testSuites))
		testSuiteNames := make([]string, len(testSuites))
		for i := range testSuites {
			testSuiteNames[i] = testSuites[i].Name
		}

		executionMap, err := s.getLatestTestSuiteExecutions(c.Context(), testSuiteNames)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: could not list test suite executions from db: %w", errPrefix, err))
		}

		for i := range testSuites {
			if execution, ok := executionMap[testSuites[i].Name]; ok {
				results = append(results, testkube.TestSuiteWithExecutionSummary{
					TestSuite:       &testSuites[i],
					LatestExecution: testsuiteexecutionsmapper.MapToSummary(&execution),
				})
			} else {
				results = append(results, testkube.TestSuiteWithExecutionSummary{
					TestSuite: &testSuites[i],
				})
			}
		}

		sort.Slice(results, func(i, j int) bool {
			iTime := results[i].TestSuite.Created
			if results[i].LatestExecution != nil {
				iTime = results[i].LatestExecution.StartTime
				// Fallback to EndTime if StartTime is not set (execution hasn't started yet)
				if iTime.IsZero() {
					iTime = results[i].LatestExecution.EndTime
				}
			}

			jTime := results[j].TestSuite.Created
			if results[j].LatestExecution != nil {
				jTime = results[j].LatestExecution.StartTime
				// Fallback to EndTime if StartTime is not set (execution hasn't started yet)
				if jTime.IsZero() {
					jTime = results[j].LatestExecution.EndTime
				}
			}

			return iTime.After(jTime)
		})

		status := c.Query("status")
		if status != "" {
			statusList, err := testkube.ParseTestSuiteExecutionStatusList(status, ",")
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: test suite execution status filter invalid: %w", errPrefix, err))
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
			pageSize = testresult.PageDefaultLimit
			page, err = strconv.Atoi(pageParam)
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: test suite page filter invalid: %w", errPrefix, err))
			}
		}

		pageSizeParam := c.Query("pageSize", "")
		if pageSizeParam != "" {
			pageSize, err = strconv.Atoi(pageSizeParam)
			if err != nil {
				s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: test suite page size filter invalid: %w", errPrefix, err))
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

func (s *DeprecatedTestkubeAPI) ExecuteTestSuitesHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to execute test suite"
		var request testkube.TestSuiteExecutionRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: test execution request body invalid: %w", errPrefix, err))
		}

		name := c.Params("id")
		selector := c.Query("selector")
		s.Log.Debugw("getting test suite", "name", name, "selector", selector)

		var testSuites []testsuitesv3.TestSuite
		if name != "" {
			errPrefix = errPrefix + " " + name
			testSuite, err := s.DeprecatedClients.TestSuites().Get(name)
			if err != nil {
				if errors.IsNotFound(err) {
					return s.Warn(c, http.StatusNotFound, fmt.Errorf("%s: test suite not found: %w", errPrefix, err))
				}

				return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could get test suite: %w", errPrefix, err))
			}
			testSuites = append(testSuites, *testSuite)
		} else {
			testSuiteList, err := s.DeprecatedClients.TestSuites().List(selector)
			if err != nil {
				return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: can't list test suites: %w", errPrefix, err))
			}

			testSuites = append(testSuites, testSuiteList.Items...)
		}

		var results []testkube.TestSuiteExecution
		if len(testSuites) != 0 {
			request.TestSuiteExecutionName = strings.Clone(c.Query("testSuiteExecutionName"))
			concurrencyLevel, err := strconv.Atoi(c.Query("concurrency", strconv.Itoa(scheduler.DefaultConcurrencyLevel)))
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: can't detect concurrency level: %w", errPrefix, err))
			}

			workerpoolService := workerpool.New[testkube.TestSuite, testkube.TestSuiteExecutionRequest, testkube.TestSuiteExecution](concurrencyLevel)

			go workerpoolService.SendRequests(s.scheduler.PrepareTestSuiteRequests(testSuites, request))
			go workerpoolService.Run(c.Context())

			for r := range workerpoolService.GetResponses() {
				results = append(results, r.Result)
			}
		}

		s.Log.Debugw("executing test", "name", name, "selector", selector)
		if name != "" && len(results) != 0 {
			if results[0].IsFailed() {
				return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: Test suite failed %v", errPrefix, name))
			}

			c.Status(http.StatusCreated)
			return c.JSON(results[0])
		}

		c.Status(http.StatusCreated)
		return c.JSON(results)
	}
}

func (s *DeprecatedTestkubeAPI) ListTestSuiteExecutionsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {

		now := time.Now()
		var l = s.Log.With("handler", "ListTestSuiteExecutionsHandler", "id", utils.RandAlphanum(10))

		errPrefix := "failed to list test suite execution"
		filter := getExecutionsFilterFromRequest(c)

		ctx := c.Context()
		executionsTotals, err := s.DeprecatedRepositories.TestSuiteResults().GetExecutionsTotals(ctx, filter)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: client could not get executions totals: %w", errPrefix, err))
		}
		l.Debugw("got executions totals", "totals", executionsTotals, "time", time.Since(now))
		filterAllTotals := *filter.(*testresult.FilterImpl)
		filterAllTotals.WithPage(0).WithPageSize(math.MaxInt32)
		allExecutionsTotals, err := s.DeprecatedRepositories.TestSuiteResults().GetExecutionsTotals(ctx, &filterAllTotals)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: client could not get all executions totals: %w", errPrefix, err))
		}
		l.Debugw("got all executions totals", "totals", executionsTotals, "time", time.Since(now))

		executions, err := s.DeprecatedRepositories.TestSuiteResults().GetExecutions(ctx, filter)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: client could not get executions: %w", errPrefix, err))
		}
		l.Debugw("got executions", "time", time.Since(now))

		return c.JSON(testkube.TestSuiteExecutionsResult{
			Totals:   &allExecutionsTotals,
			Filtered: &executionsTotals,
			Results:  testsuitesmapper.MapToTestExecutionSummary(executions),
		})
	}
}

func (s *DeprecatedTestkubeAPI) GetTestSuiteExecutionHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("executionID")
		errPrefix := fmt.Sprintf("failed to get test suite execution %s", id)

		execution, err := s.DeprecatedRepositories.TestSuiteResults().Get(c.Context(), id)
		if err == mongo.ErrNoDocuments {
			return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: test suite with execution id/name %s not found", errPrefix, id))
		}
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: could not get test suite executions from db: %w", errPrefix, err))
		}

		execution.Duration = types.FormatDuration(execution.Duration)

		secretMap := make(map[string]string)
		if execution.SecretUUID != "" && execution.TestSuite != nil {
			secretMap, err = s.DeprecatedClients.TestSuites().GetSecretTestSuiteVars(execution.TestSuite.Name, execution.SecretUUID)
			if err != nil {
				return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not get test suite secrets: %w", errPrefix, err))
			}
		}

		for key, value := range secretMap {
			if variable, ok := execution.Variables[key]; ok && value != "" {
				variable.Value = value
				variable.SecretRef = nil
				execution.Variables[key] = variable
			}
		}

		return c.JSON(execution)
	}
}

func (s *DeprecatedTestkubeAPI) ListTestSuiteArtifactsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		s.Log.Infow("listing testsuite artifacts", "executionID", c.Params("executionID"))
		id := c.Params("executionID")
		errPrefix := fmt.Sprintf("failed to list test suite artifacts %s", id)
		execution, err := s.DeprecatedRepositories.TestSuiteResults().Get(c.Context(), id)
		if err == mongo.ErrNoDocuments {
			return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: test suite with execution id/name %s not found", errPrefix, id))
		}
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: could not get test suite execution from db: %w", errPrefix, err))
		}

		var artifacts []testkube.Artifact
		for _, stepResult := range execution.StepResults {
			if stepResult.Execution == nil || stepResult.Execution.Id == "" {
				continue
			}

			artifacts, err = s.getExecutionArtfacts(c.Context(), stepResult.Execution, artifacts)
			if err != nil {
				continue
			}
		}

		for _, stepResults := range execution.ExecuteStepResults {
			for _, stepResult := range stepResults.Execute {
				if stepResult.Execution == nil || stepResult.Execution.Id == "" {
					continue
				}

				artifacts, err = s.getExecutionArtfacts(c.Context(), stepResult.Execution, artifacts)
				if err != nil {
					continue
				}
			}
		}

		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: could not list artifacts: %w", errPrefix, err))
		}

		return c.JSON(artifacts)
	}
}

func (s *DeprecatedTestkubeAPI) getExecutionArtfacts(ctx context.Context, execution *testkube.Execution,
	artifacts []testkube.Artifact) ([]testkube.Artifact, error) {
	var stepArtifacts []testkube.Artifact
	var bucket string

	artifactsStorage := s.ArtifactsStorage
	folder := execution.Id
	if execution.ArtifactRequest != nil {
		bucket = execution.ArtifactRequest.StorageBucket
		if execution.ArtifactRequest.OmitFolderPerExecution {
			folder = ""
		}
	}

	var err error
	if bucket != "" {
		artifactsStorage, err = s.getArtifactStorage(bucket)
		if err != nil {
			s.Log.Warnw("can't get artifact storage", "executionID", execution.Id, "error", err)
			return artifacts, err
		}
	}

	stepArtifacts, err = artifactsStorage.ListFiles(ctx, folder, execution.TestName, execution.TestSuiteName, "")
	if err != nil {
		s.Log.Warnw("can't list artifacts", "executionID", execution.Id, "error", err)
		return artifacts, err
	}

	s.Log.Debugw("listing artifacts for step", "executionID", execution.Id, "artifacts", stepArtifacts)
	for i := range stepArtifacts {
		stepArtifacts[i].ExecutionName = execution.Name
		artifacts = append(artifacts, stepArtifacts[i])
	}

	return artifacts, nil
}

// AbortTestSuiteHandler for aborting a TestSuite with id
func (s *DeprecatedTestkubeAPI) AbortTestSuiteHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		name := c.Params("id")
		if name == "" {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("failed to abort test suite: id cannot be empty"))
		}
		errPrefix := fmt.Sprintf("failed to abort test suite %s", name)
		filter := testresult.NewExecutionsFilter().WithName(name).WithStatus(string(testkube.RUNNING_ExecutionStatus))
		executions, err := s.DeprecatedRepositories.TestSuiteResults().GetExecutions(ctx, filter)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: executions with test syute name %s not found", errPrefix, name))
			}
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: could not get executions: %w", errPrefix, err))
		}

		for _, execution := range executions {
			execution.Status = testkube.TestSuiteExecutionStatusAborting
			s.Log.Infow("aborting test suite execution", "executionID", execution.Id)
			err := s.eventsBus.PublishTopic(bus.InternalPublishTopic, testkube.NewEventEndTestSuiteAborted(&execution))
			if err != nil {
				return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: could not sent test suite abortion event: %w", errPrefix, err))
			}
			s.Metrics.IncAbortTestSuite()

			s.Log.Infow("test suite execution aborted, event sent", "executionID", c.Params("executionID"))
		}

		return c.Status(http.StatusNoContent).SendString("")
	}
}

func (s *DeprecatedTestkubeAPI) AbortTestSuiteExecutionHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		s.Log.Infow("aborting test suite execution", "executionID", c.Params("executionID"))
		id := c.Params("executionID")
		errPrefix := fmt.Sprintf("failed to abort test suite execution %s", id)
		execution, err := s.DeprecatedRepositories.TestSuiteResults().Get(c.Context(), id)
		if err == mongo.ErrNoDocuments {
			return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: test suite with execution id/name %s not found", errPrefix, id))
		}
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: could not abort test suite execution: %w", errPrefix, err))
		}

		execution.Status = testkube.TestSuiteExecutionStatusAborting

		err = s.eventsBus.PublishTopic(bus.InternalPublishTopic, testkube.NewEventEndTestSuiteAborted(&execution))

		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: could not sent test suite abortion event: %w", errPrefix, err))
		}
		s.Log.Debugw("test suite execution aborted, event sent", "executionID", c.Params("executionID"))
		s.Metrics.IncAbortTestSuite()

		return c.Status(http.StatusNoContent).SendString("")
	}
}

// ListTestSuiteTestsHandler for getting list of all available Tests for TestSuites
func (s *DeprecatedTestkubeAPI) ListTestSuiteTestsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("id")
		errPrefix := fmt.Sprintf("failed to list tests for test suite %s", name)
		crTestSuite, err := s.DeprecatedClients.TestSuites().Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Warn(c, http.StatusNotFound, fmt.Errorf("%s: test suite with id/name %s not found", errPrefix, name))
			}

			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could get test suite: %w", errPrefix, err))
		}

		testSuite := testsuitesmapper.MapCRToAPI(*crTestSuite)
		crTests, err := s.DeprecatedClients.Tests().ListByNames(testSuite.GetTestNames())
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Warn(c, http.StatusNotFound, fmt.Errorf("%s: test suite tests with id/name %s not found", errPrefix, testSuite.GetTestNames()))
			}

			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could get tests for test suite: %w", errPrefix, err))
		}

		return c.JSON(testsmapper.MapTestArrayKubeToAPI(crTests))
	}
}

func getExecutionsFilterFromRequest(c *fiber.Ctx) testresult.Filter {

	filter := testresult.NewExecutionsFilter()
	// id for /test-suites/ID/executions
	name := c.Params("id", "")
	if name == "" {
		// query param for /executions?id
		name = c.Query("id", "")
	}

	if name != "" {
		filter = filter.WithName(name)
	}

	textSearch := c.Query("textSearch", "")
	if textSearch != "" {
		filter = filter.WithTextSearch(textSearch)
	}

	page, err := strconv.Atoi(c.Query("page", ""))
	if err == nil {
		filter = filter.WithPage(page)
	}

	pageSize, err := strconv.Atoi(c.Query("pageSize", ""))
	if err == nil && pageSize != 0 {
		filter = filter.WithPageSize(pageSize)
	}

	status := c.Query("status", "")
	if status != "" {
		filter = filter.WithStatus(status)
	}

	last, err := strconv.Atoi(c.Query("last", "0"))
	if err == nil && last != 0 {
		filter = filter.WithLastNDays(last)
	}

	dFilter := datefilter.NewDateFilter(c.Query("startDate", ""), c.Query("endDate", ""))
	if dFilter.IsStartValid {
		filter = filter.WithStartDate(dFilter.Start)
	}

	if dFilter.IsEndValid {
		filter = filter.WithEndDate(dFilter.End)
	}

	selector := c.Query("selector")
	if selector != "" {
		filter = filter.WithSelector(selector)
	}

	return filter
}
