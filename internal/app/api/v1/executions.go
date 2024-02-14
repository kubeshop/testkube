package v1

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/kubeshop/testkube/pkg/logs/events"
	"github.com/kubeshop/testkube/pkg/repository/result"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"go.mongodb.org/mongo-driver/mongo"

	testsv3 "github.com/kubeshop/testkube-operator/api/tests/v3"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/scheduler"
	"github.com/kubeshop/testkube/pkg/storage"
	"github.com/kubeshop/testkube/pkg/storage/minio"
	"github.com/kubeshop/testkube/pkg/types"
	"github.com/kubeshop/testkube/pkg/workerpool"
)

const (
	// latestExecutionNo defines the number of relevant latest executions
	latestExecutions = 5

	containerType = "container"
)

// ExecuteTestsHandler calls particular executor based on execution request content and type
func (s *TestkubeAPI) ExecuteTestsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		errPrefix := "failed to execute test"

		var request testkube.ExecutionRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: test request body invalid: %w", errPrefix, err))
		}

		if request.Args != nil {
			request.Args, err = testkube.PrepareExecutorArgs(request.Args)
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not prepare executor args: %w", errPrefix, err))
			}
		}

		id := c.Params("id")

		var tests []testsv3.Test
		if id != "" {
			test, err := s.TestsClient.Get(id)
			if err != nil {
				if errors.IsNotFound(err) {
					return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: client found no test: %w", errPrefix, err))
				}
				return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: can't get test: %w", errPrefix, err))
			}

			tests = append(tests, *test)
		} else {
			testList, err := s.TestsClient.List(c.Query("selector"))
			if err != nil {
				return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: can't get tests: %w", errPrefix, err))
			}

			tests = append(tests, testList.Items...)
		}

		l := s.Log.With("testID", id)

		if len(tests) != 0 {
			l.Infow("executing test", "test", tests[0])
		}
		var results []testkube.Execution
		if len(tests) != 0 {
			request.TestExecutionName = strings.Clone(c.Query("testExecutionName"))
			concurrencyLevel, err := strconv.Atoi(c.Query("concurrency", strconv.Itoa(scheduler.DefaultConcurrencyLevel)))
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: can't detect concurrency level: %w", errPrefix, err))
			}

			workerpoolService := workerpool.New[testkube.Test, testkube.ExecutionRequest, testkube.Execution](concurrencyLevel)

			go workerpoolService.SendRequests(s.scheduler.PrepareTestRequests(tests, request))
			go workerpoolService.Run(ctx)

			for r := range workerpoolService.GetResponses() {
				results = append(results, r.Result)
			}
		}

		if id != "" && len(results) != 0 {
			if results[0].ExecutionResult.IsFailed() {
				return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: execution failed: %s", errPrefix, results[0].ExecutionResult.ErrorMessage))
			}

			c.Status(http.StatusCreated)
			return c.JSON(results[0])
		}

		c.Status(http.StatusCreated)
		return c.JSON(results)
	}
}

// ListExecutionsHandler returns array of available test executions
func (s *TestkubeAPI) ListExecutionsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to list executions"
		// TODO refactor into some Services (based on some abstraction for CRDs at least / CRUD)
		// should we split this to separate endpoint? currently this one handles
		// endpoints from /executions and from /tests/{id}/executions
		// or should id be a query string as it's some kind of filter?

		filter := getFilterFromRequest(c)

		executions, err := s.ExecutionResults.GetExecutions(c.Context(), filter)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: db found no execution results: %w", errPrefix, err))
			}
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: db client failed to get execution results: %w", errPrefix, err))
		}

		executionTotals, err := s.ExecutionResults.GetExecutionTotals(c.Context(), false, filter)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: db client found no total execution results: %w", errPrefix, err))
			}
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: db client failed to get total execution results: %w", errPrefix, err))
		}

		filteredTotals, err := s.ExecutionResults.GetExecutionTotals(c.Context(), true, filter)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: db found no total filtered execution results: %w", errPrefix, err))
			}
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: db client failed to get total filtered execution results: %w", errPrefix, err))
		}
		results := testkube.ExecutionsResult{
			Totals:   &executionTotals,
			Filtered: &filteredTotals,
			Results:  mapExecutionsToExecutionSummary(executions),
		}

		return c.JSON(results)
	}
}

func (s *TestkubeAPI) GetLogsStream(ctx context.Context, executionID string) (chan output.Output, error) {
	execution, err := s.ExecutionResults.Get(ctx, executionID)
	if err != nil {
		return nil, fmt.Errorf("can't find execution %s: %w", executionID, err)
	}
	executor, err := s.getExecutorByTestType(execution.TestType)
	if err != nil {
		return nil, fmt.Errorf("can't get executor for test type %s: %w", execution.TestType, err)
	}

	logs, err := executor.Logs(ctx, executionID, execution.TestNamespace)
	if err != nil {
		return nil, fmt.Errorf("can't get executor logs: %w", err)
	}

	return logs, nil
}

func (s *TestkubeAPI) ExecutionLogsStreamHandler() fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		if s.featureFlags.LogsV2 {
			return
		}

		executionID := c.Params("executionID")
		l := s.Log.With("executionID", executionID)

		l.Debugw("getting pod logs and passing to websocket", "id", c.Params("id"), "locals", c.Locals, "remoteAddr", c.RemoteAddr(), "localAddr", c.LocalAddr())

		defer c.Conn.Close()

		logs, err := s.GetLogsStream(context.Background(), executionID)
		if err != nil {
			l.Errorw("can't get pod logs", "error", err)
			return
		}
		for logLine := range logs {
			l.Debugw("sending log line to websocket", "line", logLine)
			_ = c.WriteJSON(logLine)
		}
	})
}

func (s *TestkubeAPI) ExecutionLogsStreamHandlerV2() fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		if !s.featureFlags.LogsV2 {
			return
		}

		executionID := c.Params("executionID")
		l := s.Log.With("executionID", executionID)

		l.Debugw("getting logs from grpc log server and passing to websocket",
			"id", c.Params("id"), "locals", c.Locals, "remoteAddr", c.RemoteAddr(), "localAddr", c.LocalAddr())

		defer c.Conn.Close()

		logs, err := s.logGrpcClient.Get(context.Background(), executionID)
		if err != nil {
			l.Errorw("can't get logs fom grpc", "error", err)
			return
		}

		for logLine := range logs {
			if logLine.Error != nil {
				l.Errorw("can't get log line", "error", logLine.Error)
				continue
			}

			l.Debugw("sending log line to websocket", "line", logLine.Log)
			_ = c.WriteJSON(logLine.Log)
		}

		l.Debug("stream stopped in v2 logs handler")
	})
}

// ExecutionLogsHandler streams the logs from a test execution
func (s *TestkubeAPI) ExecutionLogsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if s.featureFlags.LogsV2 {
			return nil
		}

		executionID := c.Params("executionID")

		s.Log.Debug("getting logs", "executionID", executionID)

		ctx := c.Context()

		ctx.SetContentType("text/event-stream")
		ctx.Response.Header.Set("Cache-Control", "no-cache")
		ctx.Response.Header.Set("Connection", "keep-alive")
		ctx.Response.Header.Set("Transfer-Encoding", "chunked")

		ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
			s.Log.Debug("start streaming logs")
			_ = w.Flush()

			execution, err := s.ExecutionResults.Get(ctx, executionID)
			if err != nil {
				output.PrintError(os.Stdout, fmt.Errorf("could not get execution result for ID %s: %w", executionID, err))
				s.Log.Errorw("getting execution error", "error", err)
				_ = w.Flush()
				return
			}

			if execution.ExecutionResult.IsCompleted() {
				err := s.streamLogsFromResult(execution.ExecutionResult, w)
				if err != nil {
					output.PrintError(os.Stdout, fmt.Errorf("could not get execution result for ID %s: %w", executionID, err))
					s.Log.Errorw("getting execution error", "error", err)
					_ = w.Flush()
				}
				return
			}

			s.streamLogsFromJob(ctx, executionID, execution.TestType, execution.TestNamespace, w)
		})

		return nil
	}
}

// ExecutionLogsHandlerV2 streams the logs from a test execution version 2
func (s *TestkubeAPI) ExecutionLogsHandlerV2() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !s.featureFlags.LogsV2 {
			return nil
		}

		executionID := c.Params("executionID")

		s.Log.Debugw("getting logs", "executionID", executionID)

		ctx := c.Context()

		ctx.SetContentType("text/event-stream")
		ctx.Response.Header.Set("Cache-Control", "no-cache")
		ctx.Response.Header.Set("Connection", "keep-alive")
		ctx.Response.Header.Set("Transfer-Encoding", "chunked")

		ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
			s.Log.Debug("start streaming logs")
			_ = w.Flush()

			s.Log.Infow("getting logs from grpc log server")
			logs, err := s.logGrpcClient.Get(ctx, executionID)
			if err != nil {
				s.Log.Errorw("can't get logs from grpc", "error", err)
				return
			}

			s.streamLogsFromLogServer(logs, w)
		})

		return nil
	}
}

// GetExecutionHandler returns test execution object for given test and execution id/name
func (s *TestkubeAPI) GetExecutionHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		id := c.Params("id", "")
		executionID := c.Params("executionID")

		var execution testkube.Execution
		var err error

		if id == "" {
			execution, err = s.ExecutionResults.Get(ctx, executionID)
			if err == mongo.ErrNoDocuments {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("execution %s not found (test:%s)", executionID, id))
			}
			if err != nil {
				return s.Error(c, http.StatusInternalServerError, fmt.Errorf("db client was unable to get execution %s (test:%s): %w", executionID, id, err))
			}
		} else {
			execution, err = s.ExecutionResults.GetByNameAndTest(ctx, executionID, id)
			if err == mongo.ErrNoDocuments {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("test %s not found for execution %s", id, executionID))
			}
			if err != nil {
				return s.Error(c, http.StatusInternalServerError, fmt.Errorf("can't get test (%s) for execution %s: %w", id, executionID, err))
			}
		}

		execution.Duration = types.FormatDuration(execution.Duration)

		testSecretMap := make(map[string]string)
		if execution.TestSecretUUID != "" {
			testSecretMap, err = s.TestsClient.GetSecretTestVars(execution.TestName, execution.TestSecretUUID)
			if err != nil {
				return s.Error(c, http.StatusBadGateway, fmt.Errorf("client was unable to get test secrets: %w", err))
			}
		}

		testSuiteSecretMap := make(map[string]string)
		if execution.TestSuiteSecretUUID != "" {
			testSuiteSecretMap, err = s.TestsSuitesClient.GetSecretTestSuiteVars(execution.TestSuiteName, execution.TestSuiteSecretUUID)
			if err != nil {
				return s.Error(c, http.StatusBadGateway, fmt.Errorf("client was unable to get test suite secrets: %w", err))
			}
		}

		for key, value := range testSuiteSecretMap {
			testSecretMap[key] = value
		}

		for key, value := range testSecretMap {
			if variable, ok := execution.Variables[key]; ok && value != "" {
				variable.Value = value
				variable.SecretRef = nil
				execution.Variables[key] = variable
			}
		}

		s.Log.Debugw("get test execution request - debug", "execution", execution)

		return c.JSON(execution)
	}
}

func (s *TestkubeAPI) AbortExecutionHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		executionID := c.Params("executionID")
		errPrefix := "failed to abort execution %s"

		s.Log.Infow("aborting execution", "executionID", executionID)
		execution, err := s.ExecutionResults.Get(ctx, executionID)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: test with execution id %s not found", errPrefix, executionID))
			}
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: could not get test %v", errPrefix, err))
		}

		res, err := s.Executor.Abort(ctx, &execution)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: could not abort execution: %v", errPrefix, err))
		}
		s.Metrics.IncAbortTest(execution.TestType, res.IsFailed())

		return c.JSON(res)
	}
}

func (s *TestkubeAPI) GetArtifactHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		executionID := c.Params("executionID")
		fileName := c.Params("filename")
		errPrefix := fmt.Sprintf("failed to get artifact %s for execution %s", fileName, executionID)

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

		execution, err := s.ExecutionResults.Get(c.Context(), executionID)
		if err == mongo.ErrNoDocuments {
			return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: test with execution id/name %s not found", errPrefix, executionID))
		}
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: db could not get execution result: %w", errPrefix, err))
		}

		var file io.Reader
		var bucket string
		artifactsStorage := s.ArtifactsStorage
		folder := execution.Id
		if execution.ArtifactRequest != nil {
			bucket = execution.ArtifactRequest.StorageBucket
			if execution.ArtifactRequest.OmitFolderPerExecution {
				folder = ""
			}
		}

		if bucket != "" {
			artifactsStorage, err = s.getArtifactStorage(bucket)
			if err != nil {
				return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: could not get artifact storage: %w", errPrefix, err))
			}
		}

		file, err = artifactsStorage.DownloadFile(c.Context(), fileName, folder, execution.TestName, execution.TestSuiteName, "")
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: could not download file: %w", errPrefix, err))
		}

		// SendStream promises to close file using io.Close() method
		return c.SendStream(file)
	}
}

// GetArtifactArchiveHandler returns artifact archive
func (s *TestkubeAPI) GetArtifactArchiveHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		executionID := c.Params("executionID")
		query := c.Request().URI().QueryString()
		errPrefix := fmt.Sprintf("failed to get artifact archive for execution %s", executionID)

		values, err := url.ParseQuery(string(query))
		if err != nil {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse query string: %w", errPrefix, err))
		}

		execution, err := s.ExecutionResults.Get(c.Context(), executionID)
		if err == mongo.ErrNoDocuments {
			return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: test with execution id/name %s not found", errPrefix, executionID))
		}
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: db could not get execution result: %w", errPrefix, err))
		}

		var archive io.Reader
		var bucket string
		artifactsStorage := s.ArtifactsStorage
		folder := execution.Id
		if execution.ArtifactRequest != nil {
			bucket = execution.ArtifactRequest.StorageBucket
			if execution.ArtifactRequest.OmitFolderPerExecution {
				folder = ""
			}
		}

		if bucket != "" {
			artifactsStorage, err = s.getArtifactStorage(bucket)
			if err != nil {
				return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: could not get artifact storage: %w", errPrefix, err))
			}
		}

		archive, err = artifactsStorage.DownloadArchive(c.Context(), folder, values["mask"])
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: could not download artifact archive: %w", errPrefix, err))
		}

		// SendStream promises to close archive using io.Close() method
		return c.SendStream(archive)
	}
}

// ListArtifactsHandler returns list of files in the given bucket
func (s *TestkubeAPI) ListArtifactsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {

		executionID := c.Params("executionID")
		errPrefix := fmt.Sprintf("failed to list artifacts for execution %s", executionID)

		execution, err := s.ExecutionResults.Get(c.Context(), executionID)
		if err == mongo.ErrNoDocuments {
			return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: test with execution id/name %s not found", errPrefix, executionID))
		}
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: db could not get test with execution: %s", errPrefix, err))
		}

		var files []testkube.Artifact
		var bucket string
		artifactsStorage := s.ArtifactsStorage
		folder := execution.Id
		if execution.ArtifactRequest != nil {
			bucket = execution.ArtifactRequest.StorageBucket
			if execution.ArtifactRequest.OmitFolderPerExecution {
				folder = ""
			}
		}

		if bucket != "" {
			artifactsStorage, err = s.getArtifactStorage(bucket)
			if err != nil {
				return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: could not get artifact storage: %w", errPrefix, err))
			}
		}

		files, err = artifactsStorage.ListFiles(c.Context(), folder, execution.TestName, execution.TestSuiteName, "")
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: storage client could not list files %w", errPrefix, err))
		}

		return c.JSON(files)
	}
}

// streamLogsFromResult writes logs from the output of executionResult to the writer
func (s *TestkubeAPI) streamLogsFromResult(executionResult *testkube.ExecutionResult, w *bufio.Writer) error {
	enc := json.NewEncoder(w)
	_, _ = fmt.Fprintf(w, "data: ")
	s.Log.Debug("using logs from result")
	output := testkube.ExecutorOutput{
		Type_:   output.TypeResult,
		Content: executionResult.Output,
		Result:  executionResult,
	}

	if executionResult.ErrorMessage != "" {
		output.Content = output.Content + "\n" + executionResult.ErrorMessage
	}

	err := enc.Encode(output)
	if err != nil {
		s.Log.Infow("Encode", "error", err)
		return err
	}
	_, _ = fmt.Fprintf(w, "\n")
	_ = w.Flush()
	return nil
}

// streamLogsFromJob streams logs in chunks to writer from the running execution
func (s *TestkubeAPI) streamLogsFromJob(ctx context.Context, executionID, testType, namespace string, w *bufio.Writer) {
	enc := json.NewEncoder(w)
	s.Log.Infow("getting logs from Kubernetes job")

	executor, err := s.getExecutorByTestType(testType)
	if err != nil {
		output.PrintError(os.Stdout, err)
		s.Log.Errorw("getting logs error", "error", err)
		_ = w.Flush()
		return
	}

	logs, err := executor.Logs(ctx, executionID, namespace)
	s.Log.Debugw("waiting for jobs channel", "channelSize", len(logs))
	if err != nil {
		output.PrintError(os.Stdout, err)
		s.Log.Errorw("getting logs error", "error", err)
		_ = w.Flush()
		return
	}

	s.Log.Infow("looping through logs channel")
	// loop through pods log lines - it's blocking channel
	// and pass single log output as sse data chunk
	for out := range logs {
		s.Log.Debugw("got log line from pod", "out", out)
		_, _ = fmt.Fprintf(w, "data: ")
		err = enc.Encode(out)
		if err != nil {
			s.Log.Infow("Encode", "error", err)
		}
		// enc.Encode adds \n and we need \n\n after `data: {}` chunk
		_, _ = fmt.Fprintf(w, "\n")
		_ = w.Flush()
	}
}

func mapExecutionsToExecutionSummary(executions []testkube.Execution) []testkube.ExecutionSummary {
	res := make([]testkube.ExecutionSummary, len(executions))

	for i, execution := range executions {
		res[i] = testkube.ExecutionSummary{
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

	return res
}

// GetLatestExecutionLogs returns the latest executions' logs
func (s *TestkubeAPI) GetLatestExecutionLogs(ctx context.Context) (map[string][]string, error) {
	latestExecutions, err := s.getNewestExecutions(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not list executions: %w", err)
	}

	executionLogs := map[string][]string{}
	for _, e := range latestExecutions {
		logs, err := s.getExecutionLogs(ctx, e)
		if err != nil {
			return nil, fmt.Errorf("could not get logs: %w", err)
		}
		executionLogs[e.Id] = logs
	}

	return executionLogs, nil
}

// getNewestExecutions returns the latest Testkube executions
func (s *TestkubeAPI) getNewestExecutions(ctx context.Context) ([]testkube.Execution, error) {
	f := result.NewExecutionsFilter().WithPage(1).WithPageSize(latestExecutions)
	executions, err := s.ExecutionResults.GetExecutions(ctx, f)
	if err != nil {
		return []testkube.Execution{}, fmt.Errorf("could not get executions from repo: %w", err)
	}
	return executions, nil
}

// getExecutionLogs returns logs from an execution
func (s *TestkubeAPI) getExecutionLogs(ctx context.Context, execution testkube.Execution) ([]string, error) {
	var res []string

	if s.featureFlags.LogsV2 {
		logs, err := s.logGrpcClient.Get(ctx, execution.Id)
		if err != nil {
			return []string{}, fmt.Errorf("could not get logs for grpc %s: %w", execution.Id, err)
		}

		for out := range logs {
			if out.Error != nil {
				s.Log.Errorw("can't get log line", "error", out.Error)
				continue
			}

			res = append(res, out.Log.Content)
		}

		return res, nil
	}

	if execution.ExecutionResult.IsCompleted() {
		return append(res, execution.ExecutionResult.Output), nil
	}

	logs, err := s.Executor.Logs(ctx, execution.Id, execution.TestNamespace)
	if err != nil {
		return []string{}, fmt.Errorf("could not get logs for execution %s: %w", execution.Id, err)
	}

	for out := range logs {
		res = append(res, out.Result.Output)
	}

	return res, nil
}

func (s *TestkubeAPI) getExecutorByTestType(testType string) (client.Executor, error) {
	executorCR, err := s.ExecutorsClient.GetByType(testType)
	if err != nil {
		return nil, fmt.Errorf("can't get executor spec: %w", err)
	}
	switch executorCR.Spec.ExecutorType {
	case containerType:
		return s.ContainerExecutor, nil
	default:
		return s.Executor, nil
	}
}

func (s *TestkubeAPI) getArtifactStorage(bucket string) (storage.ArtifactsStorage, error) {
	if s.mode == common.ModeAgent {
		return s.ArtifactsStorage, nil
	}

	opts := minio.GetTLSOptions(s.storageParams.SSL, s.storageParams.SkipVerify, s.storageParams.CertFile, s.storageParams.KeyFile, s.storageParams.CAFile)
	minioClient := minio.NewClient(
		s.storageParams.Endpoint,
		s.storageParams.AccessKeyId,
		s.storageParams.SecretAccessKey,
		s.storageParams.Region,
		s.storageParams.Token,
		bucket,
		opts...,
	)
	if err := minioClient.Connect(); err != nil {
		return nil, err
	}

	return minio.NewMinIOArtifactClient(minioClient), nil
}

// streamLogsFromLogServer writes logs from the output of log server to the writer
func (s *TestkubeAPI) streamLogsFromLogServer(logs chan events.LogResponse, w *bufio.Writer) {
	enc := json.NewEncoder(w)
	s.Log.Infow("looping through logs channel")
	// loop through grpc server log lines - it's blocking channel
	// and pass single log output as sse data chunk
	for out := range logs {
		if out.Error != nil {
			s.Log.Errorw("can't get log line", "error", out.Error)
			continue
		}

		s.Log.Debugw("got log line from grpc log server", "out", out.Log)
		_, _ = fmt.Fprintf(w, "data: ")
		err := enc.Encode(out.Log)
		if err != nil {
			s.Log.Infow("Encode", "error", err)
		}
		// enc.Encode adds \n and we need \n\n after `data: {}` chunk
		_, _ = fmt.Fprintf(w, "\n")
		_ = w.Flush()
	}

	s.Log.Debugw("logs streaming stopped")
}
