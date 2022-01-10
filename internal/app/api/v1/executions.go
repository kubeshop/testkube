package v1

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/kubeshop/testkube/internal/pkg/api/repository/result"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/kubeshop/testkube/pkg/rand"
	"github.com/kubeshop/testkube/pkg/runner/output"
)

// ExecuteScriptHandler calls particular executor based on execution request content and type
func (s TestKubeAPI) ExecuteScriptHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()

		var request testkube.ExecutionRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("script request body invalid: %w", err))
		}

		scriptID := c.Params("id")
		namespace := request.Namespace

		// generate random execution name in case there is no one set
		// like for docker images
		if request.Name == "" {
			request.Name = rand.Name()
		}

		// script name + script execution name should be unique
		execution, _ := s.ExecutionResults.GetByNameAndScript(c.Context(), request.Name, scriptID)
		if execution.Name == request.Name {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("script execution with name %s already exists", request.Name))
		}

		// merge available data into execution options script spec, executor spec, request, script id
		options, err := s.GetExecuteOptions(namespace, scriptID, request)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("can't create valid execution options: %w", err))
		}

		execution = s.executeScript(ctx, options)
		if execution.ExecutionResult.IsFailed() {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf(execution.ExecutionResult.ErrorMessage))
		}

		return c.JSON(execution)
	}
}

func (s TestKubeAPI) executeScript(ctx context.Context, options client.ExecuteOptions) (execution testkube.Execution) {
	// store execution in storage, can be get from API now
	execution = newExecutionFromExecutionOptions(options)
	options.ID = execution.Id
	execution.Tags = options.ScriptSpec.Tags

	err := s.ExecutionResults.Insert(ctx, execution)
	if err != nil {
		return execution.Errw("can't create new script execution, can't insert into storage: %w", err)
	}

	// call executor rest or job based and update execution object after queueing execution
	s.Log.Infow("calling executor with options", "options", options.Request)
	execution.Start()
	err = s.ExecutionResults.StartExecution(ctx, execution.Id, execution.StartTime)
	if err != nil {
		return execution.Errw("can't execute script, rnto storage error: %w", err)
	}

	var result testkube.ExecutionResult

	// sync/async script execution
	if options.Sync {
		result, err = s.Executor.ExecuteSync(execution, options)
	} else {
		result, err = s.Executor.Execute(execution, options)
	}

	if uerr := s.ExecutionResults.UpdateResult(ctx, execution.Id, result); uerr != nil {
		return execution.Errw("update execution error: %w", uerr)
	}

	// set execution result to one created
	execution.ExecutionResult = &result

	// metrics increase
	s.Metrics.IncExecution(execution)

	if err != nil {
		return execution.Errw("script execution failed: %w", err)
	}

	s.Log.Infow("script executed", "executionId", execution.Id, "status", execution.ExecutionResult.Status)

	return
}

// ListExecutionsHandler returns array of available script executions
func (s TestKubeAPI) ListExecutionsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// TODO should we split this to separate endpoint? currently this one handles
		// endpoints from /executions and from /scripts/{id}/executions
		// or should scriptID be a query string as it's some kind of filter?

		filter := getFilterFromRequest(c)

		executions, err := s.ExecutionResults.GetExecutions(c.Context(), filter)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, err)
		}

		executionTotals, err := s.ExecutionResults.GetExecutionTotals(c.Context(), result.NewExecutionsFilter())
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, err)
		}

		filteredTotals, err := s.ExecutionResults.GetExecutionTotals(c.Context(), filter)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, err)
		}
		results := testkube.ExecutionsResult{
			Totals:   &executionTotals,
			Filtered: &filteredTotals,
			Results:  convertToExecutionSummary(executions),
		}

		return c.JSON(results)
	}
}

// ExecutionLogsHandler returns execution logs for given execution id
func (s TestKubeAPI) ExecutionLogsHandler() fiber.Handler {
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
			enc := json.NewEncoder(w)

			// get logs from job executor pods
			s.Log.Debug("getting logs")
			var logs chan output.Output
			var err error

			logs, err = s.Executor.Logs(executionID)
			s.Log.Debugw("waiting for jobs channel", "channelSize", len(logs))
			if err != nil {
				// TODO convert to some library for common output
				fmt.Fprintf(w, `data: {"type": "error","message": "%s"}\n\n`, err.Error())
				s.Log.Errorw("getting logs error", "error", err)
				w.Flush()
				return
			}

			// loop through pods log lines - it's blocking channel
			// and pass single log output as sse data chunk
			for out := range logs {
				s.Log.Debugw("got log", "out", out)
				fmt.Fprintf(w, "data: ")
				enc.Encode(out)
				// enc.Encode adds \n and we need \n\n after `data: {}` chunk
				fmt.Fprintf(w, "\n")
				w.Flush()
			}
		}))

		return nil
	}
}

// GetExecutionHandler returns script execution object for given script and execution id
func (s TestKubeAPI) GetExecutionHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		scriptID := c.Params("id", "-")
		executionID := c.Params("executionID")

		var execution testkube.Execution
		var err error

		if scriptID == "-" {
			execution, err = s.ExecutionResults.Get(ctx, executionID)
			if err == mongo.ErrNoDocuments {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("script with execution id %s not found", executionID))
			}
			if err != nil {
				return s.Error(c, http.StatusInternalServerError, err)
			}
		} else {
			execution, err = s.ExecutionResults.GetByNameAndScript(ctx, executionID, scriptID)
			if err == mongo.ErrNoDocuments {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("script %s/%s not found", scriptID, executionID))
			}
			if err != nil {
				return s.Error(c, http.StatusInternalServerError, err)
			}
		}

		s.Log.Infow("get script execution request", "id", scriptID, "executionID", executionID)
		s.Log.Debugw("get script execution request - debug", "execution", execution)

		return c.JSON(execution)
	}
}

// AbortExecutionHandler aborts script execution for given executor id
func (s TestKubeAPI) AbortExecutionHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		return s.Executor.Abort(id)
	}
}

// GetArtifactHandler returns execution result file for given execution id and filename
func (s TestKubeAPI) GetArtifactHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		executionID := c.Params("executionID")
		fileName := c.Params("filename")

		// TODO fix this someday :) we don't know 15 mins before release why it's working this way
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

// ListArtifactsHandler returns list of files for the given execution id
func (s TestKubeAPI) ListArtifactsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {

		executionID := c.Params("executionID")
		files, err := s.Storage.ListFiles(executionID)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, err)
		}

		return c.JSON(files)
	}
}

// GetExecuteOptions returns execute options for given namespace, script id and request
func (s TestKubeAPI) GetExecuteOptions(namespace, scriptID string, request testkube.ExecutionRequest) (options client.ExecuteOptions, err error) {
	// get script content from kubernetes CRs
	scriptCR, err := s.ScriptsClient.Get(namespace, scriptID)

	if err != nil {
		return options, fmt.Errorf("can't get script custom resource %w", err)
	}

	// get executor from kubernetes CRs
	executorCR, err := s.ExecutorsClient.GetByType(scriptCR.Spec.Type_)
	if err != nil {
		return options, fmt.Errorf("can't get executor spec: %w", err)
	}

	return client.ExecuteOptions{
		ScriptName:   scriptID,
		ScriptSpec:   scriptCR.Spec,
		ExecutorName: executorCR.ObjectMeta.Name,
		ExecutorSpec: executorCR.Spec,
		Request:      request,
	}, nil
}

func newExecutionFromExecutionOptions(options client.ExecuteOptions) testkube.Execution {
	execution := testkube.NewExecution(
		options.ScriptName,
		options.Request.Name,
		options.ScriptSpec.Type_,
		options.ScriptSpec.Content,
		testkube.NewQueuedExecutionResult(),
		options.Request.Params,
		options.Request.Tags,
	)

	execution.Repository = (*testkube.Repository)(options.ScriptSpec.Repository)

	return execution
}

func convertToExecutionSummary(executions []testkube.Execution) []testkube.ExecutionSummary {
	result := make([]testkube.ExecutionSummary, len(executions))

	for i, execution := range executions {
		result[i] = testkube.ExecutionSummary{
			Id:         execution.Id,
			Name:       execution.Name,
			ScriptName: execution.ScriptName,
			ScriptType: execution.ScriptType,
			Status:     execution.ExecutionResult.Status,
			StartTime:  execution.StartTime,
			EndTime:    execution.EndTime,
		}
	}

	return result
}
