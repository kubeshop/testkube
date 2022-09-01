package v1

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
	"go.mongodb.org/mongo-driver/mongo"
	"k8s.io/apimachinery/pkg/api/errors"

	testsv3 "github.com/kubeshop/testkube-operator/apis/tests/v3"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/internal/pkg/api/repository/result"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/kubeshop/testkube/pkg/executor/output"
	testsmapper "github.com/kubeshop/testkube/pkg/mapper/tests"
	"github.com/kubeshop/testkube/pkg/secret"
	"github.com/kubeshop/testkube/pkg/types"
	"github.com/kubeshop/testkube/pkg/workerpool"
)

const (
	// testResourceURI is test resource uri for cron job call
	testResourceURI = "tests"
	// testSuiteResourceURI is test suite resource uri for cron job call
	testSuiteResourceURI = "test-suites"
	// defaultConcurrencyLevel is a default concurrency level for worker pool
	defaultConcurrencyLevel = "10"
	// latestExecutionNo defines the number of relevant latest executions
	latestExecutions = 5
)

// ExecuteTestsHandler calls particular executor based on execution request content and type
func (s TestkubeAPI) ExecuteTestsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()

		var request testkube.ExecutionRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("test request body invalid: %w", err))
		}

		id := c.Params("id")

		var tests []testsv3.Test
		if id != "" {
			test, err := s.TestsClient.Get(id)
			if err != nil {
				return s.Error(c, http.StatusInternalServerError, fmt.Errorf("can't get test: %w", err))
			}

			tests = append(tests, *test)
		} else {
			testList, err := s.TestsClient.List(c.Query("selector"))
			if err != nil {
				return s.Error(c, http.StatusInternalServerError, fmt.Errorf("can't get tests: %w", err))
			}

			tests = append(tests, testList.Items...)
		}

		var results []testkube.Execution
		if len(tests) != 0 {
			concurrencyLevel, err := strconv.Atoi(c.Query("concurrency", defaultConcurrencyLevel))
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("can't detect concurrency level: %w", err))
			}

			workerpoolService := workerpool.New[testkube.Test, testkube.ExecutionRequest, testkube.Execution](concurrencyLevel)

			go workerpoolService.SendRequests(s.prepareTestRequests(tests, request))
			go workerpoolService.Run(ctx)

			for r := range workerpoolService.GetResponses() {
				results = append(results, r.Result)
			}
		}

		if id != "" && len(results) != 0 {
			if results[0].ExecutionResult.IsFailed() {
				return s.Error(c, http.StatusInternalServerError, fmt.Errorf(results[0].ExecutionResult.ErrorMessage))
			}

			c.Status(http.StatusCreated)
			return c.JSON(results[0])
		}

		c.Status(http.StatusCreated)
		return c.JSON(results)
	}
}

func (s TestkubeAPI) prepareTestRequests(work []testsv3.Test, request testkube.ExecutionRequest) []workerpool.Request[
	testkube.Test, testkube.ExecutionRequest, testkube.Execution] {
	requests := make([]workerpool.Request[testkube.Test, testkube.ExecutionRequest, testkube.Execution], len(work))
	for i := range work {
		requests[i] = workerpool.Request[testkube.Test, testkube.ExecutionRequest, testkube.Execution]{
			Object:  testsmapper.MapTestCRToAPI(work[i]),
			Options: request,
			ExecFn:  s.executeTest,
		}
	}
	return requests
}

func (s TestkubeAPI) executeTest(ctx context.Context, test testkube.Test, request testkube.ExecutionRequest) (
	execution testkube.Execution, err error) {
	// generate random execution name in case there is no one set
	// like for docker images
	if request.Name == "" && test.ExecutionRequest != nil && test.ExecutionRequest.Name != "" {
		request.Name = test.ExecutionRequest.Name
	}

	if request.Name == "" {
		request.Name = test.Name
	}

	request.Number = s.getNextExecutionNumber(test.Name)
	request.Name = fmt.Sprintf("%s-%d", request.Name, request.Number)

	// test name + test execution name should be unique
	execution, _ = s.ExecutionResults.GetByNameAndTest(ctx, request.Name, test.Name)
	if execution.Name == request.Name {
		return execution.Err(fmt.Errorf("test execution with name %s already exists", request.Name)), nil
	}

	secretUUID, err := s.TestsClient.GetCurrentSecretUUID(test.Name)
	if err != nil {
		return execution.Errw("can't get current secret uuid: %w", err), nil
	}

	request.TestSecretUUID = secretUUID
	// merge available data into execution options test spec, executor spec, request, test id
	options, err := s.GetExecuteOptions(test.Namespace, test.Name, request)
	if err != nil {
		return execution.Errw("can't create valid execution options: %w", err), nil
	}

	// store execution in storage, can be get from API now
	execution = newExecutionFromExecutionOptions(options)
	options.ID = execution.Id

	if err := s.createSecretsReferences(&execution); err != nil {
		return execution.Errw("can't create secret variables `Secret` references: %w", err), nil
	}

	err = s.ExecutionResults.Insert(ctx, execution)
	if err != nil {
		return execution.Errw("can't create new test execution, can't insert into storage: %w", err), nil
	}

	s.Log.Infow("calling executor with options", "options", options.Request)
	execution.Start()

	s.Events.Notify(testkube.NewEventStartTest(&execution))

	// update storage with current execution status
	err = s.ExecutionResults.StartExecution(ctx, execution.Id, execution.StartTime)
	if err != nil {
		s.Events.Notify(testkube.NewEventEndTest(&execution))
		return execution.Errw("can't execute test, can't insert into storage error: %w", err), nil
	}

	options.HasSecrets = true
	if _, err = s.SecretClient.Get(secret.GetMetadataName(execution.TestName)); err != nil {
		if !errors.IsNotFound(err) {
			s.Events.Notify(testkube.NewEventEndTest(&execution))
			return execution.Errw("can't get secrets: %w", err), nil
		}

		options.HasSecrets = false
	}

	var result testkube.ExecutionResult

	// sync/async test execution
	if options.Sync {
		result, err = s.Executor.ExecuteSync(&execution, options)
	} else {
		result, err = s.Executor.Execute(&execution, options)
	}

	// set execution result to one created
	execution.ExecutionResult = &result

	// update storage with current execution status
	if uerr := s.ExecutionResults.UpdateResult(ctx, execution.Id, result); uerr != nil {
		s.Events.Notify(testkube.NewEventEndTest(&execution))
		return execution.Errw("update execution error: %w", uerr), nil
	}

	if err != nil {
		s.Events.Notify(testkube.NewEventEndTest(&execution))
		return execution.Errw("test execution failed: %w", err), nil
	}

	s.Log.Infow("test executed", "executionId", execution.Id, "status", execution.ExecutionResult.Status)

	if execution.ExecutionResult != nil && *execution.ExecutionResult.Status != testkube.RUNNING_ExecutionStatus {
		s.Events.Notify(testkube.NewEventEndTest(&execution))
	}

	return execution, nil
}

// createSecretsReferences strips secrets from text and store it inside model as reference to secret
func (s TestkubeAPI) createSecretsReferences(execution *testkube.Execution) (err error) {
	secrets := map[string]string{}
	secretName := execution.Id + "-vars"

	for k, v := range execution.Variables {
		if v.IsSecret() {
			obfuscated := execution.Variables[k]
			if v.SecretRef != nil {
				obfuscated.SecretRef = &testkube.SecretRef{
					Namespace: execution.TestNamespace,
					Name:      v.SecretRef.Name,
					Key:       v.SecretRef.Key,
				}
			} else {
				obfuscated.Value = ""
				obfuscated.SecretRef = &testkube.SecretRef{
					Namespace: execution.TestNamespace,
					Name:      secretName,
					Key:       v.Name,
				}

				secrets[v.Name] = v.Value
			}

			execution.Variables[k] = obfuscated
		}
	}

	labels := map[string]string{"executionID": execution.Id, "testName": execution.TestName}

	if len(secrets) > 0 {
		return s.SecretClient.Create(
			secretName,
			labels,
			secrets,
		)
	}

	return nil
}

// ListExecutionsHandler returns array of available test executions
func (s TestkubeAPI) ListExecutionsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// TODO refactor into some Services (based on some abstraction for CRDs at least / CRUD)
		// should we split this to separate endpoint? currently this one handles
		// endpoints from /executions and from /tests/{id}/executions
		// or should id be a query string as it's some kind of filter?

		filter := getFilterFromRequest(c)

		executions, err := s.ExecutionResults.GetExecutions(c.Context(), filter)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, err)
		}

		executionTotals, err := s.ExecutionResults.GetExecutionTotals(c.Context(), false, filter)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, err)
		}

		filteredTotals, err := s.ExecutionResults.GetExecutionTotals(c.Context(), true, filter)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, err)
		}
		results := testkube.ExecutionsResult{
			Totals:   &executionTotals,
			Filtered: &filteredTotals,
			Results:  mapExecutionsToExecutionSummary(executions),
		}

		return c.JSON(results)
	}
}

// ExecutionLogsHandler streams the logs from a test execution
func (s *TestkubeAPI) ExecutionLogsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		executionID := c.Params("executionID")

		s.Log.Debug("getting logs", "executionID", executionID)

		ctx := c.Context()

		ctx.SetContentType("text/event-stream")
		ctx.Response.Header.Set("Cache-Control", "no-cache")
		ctx.Response.Header.Set("Connection", "keep-alive")
		ctx.Response.Header.Set("Transfer-Encoding", "chunked")

		ctx.SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
			s.Log.Debug("starting stream writer")
			w.Flush()

			execution, err := s.ExecutionResults.Get(ctx, executionID)
			if err != nil {
				output.PrintError(os.Stdout, fmt.Errorf("could not get execution result for ID %s: %w", executionID, err))
				s.Log.Errorw("getting execution error", "error", err)
				w.Flush()
				return
			}

			if execution.ExecutionResult.IsCompleted() {
				err := s.streamLogsFromResult(execution.ExecutionResult, w)
				if err != nil {
					output.PrintError(os.Stdout, fmt.Errorf("could not get execution result for ID %s: %w", executionID, err))
					s.Log.Errorw("getting execution error", "error", err)
					w.Flush()
				}
				return
			}

			s.streamLogsFromJob(executionID, w)
		}))

		return nil
	}
}

// GetExecutionHandler returns test execution object for given test and execution id/name
func (s TestkubeAPI) GetExecutionHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		id := c.Params("id", "")
		executionID := c.Params("executionID")

		var execution testkube.Execution
		var err error

		if id == "" {
			execution, err = s.ExecutionResults.Get(ctx, executionID)
			if err == mongo.ErrNoDocuments {
				execution, err = s.ExecutionResults.GetByName(ctx, executionID)
				if err == mongo.ErrNoDocuments {
					return s.Error(c, http.StatusNotFound, fmt.Errorf("test with execution id/name %s not found", executionID))
				}
			}
			if err != nil {
				return s.Error(c, http.StatusInternalServerError, err)
			}
		} else {
			execution, err = s.ExecutionResults.GetByNameAndTest(ctx, executionID, id)
			if err == mongo.ErrNoDocuments {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("test %s/%s not found", id, executionID))
			}
			if err != nil {
				return s.Error(c, http.StatusInternalServerError, err)
			}
		}

		execution.Duration = types.FormatDuration(execution.Duration)

		testSecretMap := make(map[string]string)
		if execution.TestSecretUUID != "" {
			testSecretMap, err = s.TestsClient.GetSecretTestVars(execution.TestName, execution.TestSecretUUID)
			if err != nil {
				return s.Error(c, http.StatusInternalServerError, err)
			}
		}

		testSuiteSecretMap := make(map[string]string)
		if execution.TestSuiteSecretUUID != "" {
			testSuiteSecretMap, err = s.TestsSuitesClient.GetSecretTestSuiteVars(execution.TestSuiteName, execution.TestSuiteSecretUUID)
			if err != nil {
				return s.Error(c, http.StatusInternalServerError, err)
			}
		}

		for key, value := range testSuiteSecretMap {
			testSecretMap[key] = value
		}

		for key, value := range testSecretMap {
			if variable, ok := execution.Variables[key]; ok && value != "" {
				variable.Value = string(value)
				variable.SecretRef = nil
				execution.Variables[key] = variable
			}
		}

		s.Log.Debugw("get test execution request - debug", "execution", execution)

		return c.JSON(execution)
	}
}

func (s TestkubeAPI) AbortExecutionHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		executionID := c.Params("executionID")
		execution, err := s.ExecutionResults.Get(ctx, executionID)
		if err == mongo.ErrNoDocuments {
			return s.Error(c, http.StatusNotFound, fmt.Errorf("test with execution id %s not found", executionID))
		}

		if err != nil {
			return s.Error(c, http.StatusInternalServerError, err)
		}

		result := s.Executor.Abort(executionID)
		s.Metrics.IncAbortTest(execution.TestType, result.IsFailed())

		return err
	}
}

func (s TestkubeAPI) GetArtifactHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		executionID := c.Params("executionID")
		fileName := c.Params("filename")

		// TODO fix this someday :) we don't know 15 mins before release why it's working this way
		// remember about CLI client and Dashboard client too!
		unescaped, err := url.QueryUnescape(fileName)
		if err == nil {
			fileName = unescaped
		}

		unescaped, err = url.QueryUnescape(fileName)
		if err == nil {
			fileName = unescaped
		}

		//// quickfix end

		file, err := s.Storage.DownloadFile(executionID, fileName)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, err)
		}
		defer file.Close()

		return c.SendStream(file)
	}
}

// GetArtifacts returns list of files in the given bucket
func (s TestkubeAPI) ListArtifactsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {

		executionID := c.Params("executionID")
		files, err := s.Storage.ListFiles(executionID)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, err)
		}

		return c.JSON(files)
	}
}

func (s TestkubeAPI) GetExecuteOptions(namespace, id string, request testkube.ExecutionRequest) (options client.ExecuteOptions, err error) {
	// get test content from kubernetes CRs
	testCR, err := s.TestsClient.Get(id)
	if err != nil {
		return options, fmt.Errorf("can't get test custom resource %w", err)
	}

	test := testsmapper.MapTestCRToAPI(*testCR)

	if test.ExecutionRequest != nil {
		// Test variables lowest priority, then test suite, then test suite execution / test execution
		request.Variables = mergeVariables(test.ExecutionRequest.Variables, request.Variables)
		// Combine test executor args with execution args
		request.Args = append(request.Args, test.ExecutionRequest.Args...)
		request.Envs = mergeEnvs(request.Envs, test.ExecutionRequest.Envs)
		request.SecretEnvs = mergeEnvs(request.SecretEnvs, test.ExecutionRequest.SecretEnvs)
		if request.HttpProxy == "" && test.ExecutionRequest.HttpProxy != "" {
			request.HttpProxy = test.ExecutionRequest.HttpProxy
		}

		if request.HttpsProxy == "" && test.ExecutionRequest.HttpsProxy != "" {
			request.HttpsProxy = test.ExecutionRequest.HttpsProxy
		}
	}

	// get executor from kubernetes CRs
	executorCR, err := s.ExecutorsClient.GetByType(testCR.Spec.Type_)
	if err != nil {
		return options, fmt.Errorf("can't get executor spec: %w", err)
	}

	var usernameSecret, tokenSecret *testkube.SecretRef
	if test.Content != nil && test.Content.Repository != nil {
		usernameSecret = test.Content.Repository.UsernameSecret
		tokenSecret = test.Content.Repository.TokenSecret
	}

	return client.ExecuteOptions{
		TestName:       id,
		Namespace:      namespace,
		TestSpec:       testCR.Spec,
		ExecutorName:   executorCR.ObjectMeta.Name,
		ExecutorSpec:   executorCR.Spec,
		Request:        request,
		Sync:           request.Sync,
		Labels:         testCR.Labels,
		UsernameSecret: usernameSecret,
		TokenSecret:    tokenSecret,
		ImageOverride:  request.Image,
	}, nil
}

// streamLogsFromResult writes logs from the output of executionResult to the writer
func (s *TestkubeAPI) streamLogsFromResult(executionResult *testkube.ExecutionResult, w *bufio.Writer) error {
	enc := json.NewEncoder(w)
	fmt.Fprintf(w, "data: ")
	s.Log.Debug("using logs from result")
	output := testkube.ExecutorOutput{
		Type_:   output.TypeResult,
		Content: executionResult.Output,
		Result:  executionResult,
	}
	err := enc.Encode(output)
	if err != nil {
		s.Log.Infow("Encode", "error", err)
		return err
	}
	fmt.Fprintf(w, "\n")
	w.Flush()
	return nil
}

// streamLogsFromJob streams logs in chunks to writer from the running execution
func (s *TestkubeAPI) streamLogsFromJob(executionID string, w *bufio.Writer) {
	enc := json.NewEncoder(w)
	s.Log.Debug("getting logs from Kubernetes job")

	logs, err := s.Executor.Logs(executionID)
	s.Log.Debugw("waiting for jobs channel", "channelSize", len(logs))
	if err != nil {
		output.PrintError(os.Stdout, err)
		s.Log.Errorw("getting logs error", "error", err)
		w.Flush()
		return
	}

	// loop through pods log lines - it's blocking channel
	// and pass single log output as sse data chunk
	for out := range logs {
		s.Log.Debugw("got log", "out", out)
		fmt.Fprintf(w, "data: ")
		err = enc.Encode(out)
		if err != nil {
			s.Log.Infow("Encode", "error", err)
		}
		// enc.Encode adds \n and we need \n\n after `data: {}` chunk
		fmt.Fprintf(w, "\n")
		w.Flush()
	}
}

func (s TestkubeAPI) getNextExecutionNumber(testName string) int {
	number, err := s.ExecutionResults.GetNextExecutionNumber(context.Background(), testName)
	if err != nil {
		s.Log.Errorw("retrieving latest execution", "error", err)
		return number
	}
	return number
}

func mergeVariables(vars1 map[string]testkube.Variable, vars2 map[string]testkube.Variable) map[string]testkube.Variable {
	variables := map[string]testkube.Variable{}
	for k, v := range vars1 {
		variables[k] = v
	}
	for k, v := range vars2 {
		variables[k] = v
	}

	return variables
}

func mergeEnvs(envs1 map[string]string, envs2 map[string]string) map[string]string {
	envs := map[string]string{}
	for k, v := range envs1 {
		envs[k] = v
	}
	for k, v := range envs2 {
		envs[k] = v
	}

	return envs
}

func newExecutionFromExecutionOptions(options client.ExecuteOptions) testkube.Execution {
	execution := testkube.NewExecution(
		options.Namespace,
		options.TestName,
		options.Request.TestSuiteName,
		options.Request.Name,
		options.TestSpec.Type_,
		options.Request.Number,
		testsmapper.MapTestContentFromSpec(options.TestSpec.Content),
		testkube.NewRunningExecutionResult(),
		options.Request.Variables,
		options.Request.TestSecretUUID,
		options.Request.TestSuiteSecretUUID,
		common.MergeMaps(options.Labels, options.Request.ExecutionLabels),
	)

	execution.Envs = options.Request.Envs
	execution.Args = options.Request.Args
	execution.VariablesFile = options.Request.VariablesFile

	return execution
}

func mapExecutionsToExecutionSummary(executions []testkube.Execution) []testkube.ExecutionSummary {
	result := make([]testkube.ExecutionSummary, len(executions))

	for i, execution := range executions {
		result[i] = testkube.ExecutionSummary{
			Id:         execution.Id,
			Name:       execution.Name,
			Number:     execution.Number,
			TestName:   execution.TestName,
			TestType:   execution.TestType,
			Status:     execution.ExecutionResult.Status,
			StartTime:  execution.StartTime,
			EndTime:    execution.EndTime,
			Duration:   types.FormatDuration(execution.Duration),
			DurationMs: types.FormatDurationMs(execution.Duration),
			Labels:     execution.Labels,
		}
	}

	return result
}

// GetLatestExecutionLogs returns the latest executions' logs
func (s *TestkubeAPI) GetLatestExecutionLogs(c context.Context) (map[string][]string, error) {
	latestExecutions, err := s.getNewestExecutions(c)
	if err != nil {
		return nil, fmt.Errorf("could not list executions: %w", err)
	}

	executionLogs := map[string][]string{}
	for _, e := range latestExecutions {
		logs, err := s.getExecutionLogs(e)
		if err != nil {
			return nil, fmt.Errorf("could not get logs: %w", err)
		}
		executionLogs[e.Id] = logs
	}

	return executionLogs, nil
}

// getNewestExecutions returns the latest Testkube executions
func (s *TestkubeAPI) getNewestExecutions(c context.Context) ([]testkube.Execution, error) {
	f := result.NewExecutionsFilter().WithPage(1).WithPageSize(latestExecutions)
	executions, err := s.ExecutionResults.GetExecutions(c, f)
	if err != nil {
		return []testkube.Execution{}, fmt.Errorf("could not get executions from repo: %w", err)
	}
	return executions, nil
}

// getExecutionLogs returns logs from an execution
func (s *TestkubeAPI) getExecutionLogs(execution testkube.Execution) ([]string, error) {
	result := []string{}
	if execution.ExecutionResult.IsCompleted() {
		return append(result, execution.ExecutionResult.Output), nil
	}

	logs, err := s.Executor.Logs(execution.Id)
	if err != nil {
		return []string{}, fmt.Errorf("could not get logs for execution %s: %w", execution.Id, err)
	}

	for out := range logs {
		result = append(result, out.Result.Output)
	}

	return result, nil
}
