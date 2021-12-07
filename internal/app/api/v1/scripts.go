package v1

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gofiber/fiber/v2"
	scriptsv1 "github.com/kubeshop/testkube-operator/apis/script/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/client"
	scriptsMapper "github.com/kubeshop/testkube/pkg/mapper/scripts"
	"github.com/kubeshop/testkube/pkg/runner/output"
	"github.com/valyala/fasthttp"

	"github.com/kubeshop/testkube/pkg/rand"
	"go.mongodb.org/mongo-driver/mongo"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/testkube/internal/pkg/api"
	"github.com/kubeshop/testkube/internal/pkg/api/datefilter"
	"github.com/kubeshop/testkube/internal/pkg/api/repository/result"
)

// ListScripts for getting list of all available scripts
func (s testkubeAPI) GetScript() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("id")
		namespace := c.Query("namespace", "testkube")
		crScript, err := s.ScriptsClient.Get(namespace, name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, err)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		scripts := scriptsMapper.MapScriptCRToAPI(*crScript)

		return c.JSON(scripts)
	}
}

// ListScripts for getting list of all available scripts
func (s testkubeAPI) ListScripts() fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespace := c.Query("namespace", "testkube")
		crScripts, err := s.ScriptsClient.List(namespace)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		scripts := scriptsMapper.MapScriptListKubeToAPI(*crScripts)

		return c.JSON(scripts)
	}
}

// CreateScript creates new script CR based on script content
func (s testkubeAPI) CreateScript() fiber.Handler {
	return func(c *fiber.Ctx) error {

		var request testkube.ScriptUpsertRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		s.Log.Infow("creating script", "request", request)

		var repository *scriptsv1.Repository

		if request.Repository != nil {
			repository = &scriptsv1.Repository{
				Type_:  "git",
				Uri:    request.Repository.Uri,
				Branch: request.Repository.Branch,
				Path:   request.Repository.Path,
			}
		}

		script, err := s.ScriptsClient.Create(&scriptsv1.Script{
			ObjectMeta: metav1.ObjectMeta{
				Name:            request.Name,
				Namespace:       request.Namespace,
				ResourceVersion: "1",
			},
			Spec: scriptsv1.ScriptSpec{
				Type_:      request.Type_,
				InputType:  request.InputType,
				Content:    request.Content,
				Repository: repository,
			},
		})

		s.Metrics.IncCreateScript(script.Spec.Type_, err)

		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		return c.JSON(script)
	}
}

// UpdateScript creates new script CR based on script content
func (s testkubeAPI) UpdateScript() fiber.Handler {
	return func(c *fiber.Ctx) error {

		var request testkube.ScriptUpsertRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		s.Log.Infow("updating script", "request", request)

		var repository *scriptsv1.Repository

		if request.Repository != nil {
			repository = &scriptsv1.Repository{
				Type_:  "git",
				Uri:    request.Repository.Uri,
				Branch: request.Repository.Branch,
				Path:   request.Repository.Path,
			}
		}

		// we need to get resouece first and load its metadata.ResourceVersion
		script, err := s.ScriptsClient.Get(request.Namespace, request.Name)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		script.Spec = scriptsv1.ScriptSpec{
			Type_:      request.Type_,
			InputType:  request.InputType,
			Content:    request.Content,
			Repository: repository,
		}

		script, err = s.ScriptsClient.Update(script)

		s.Metrics.IncUpdateScript(script.Spec.Type_, err)

		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		return c.JSON(script)
	}
}

// DeleteScript for deleting a script with id
func (s testkubeAPI) DeleteScript() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("id")
		namespace := c.Query("namespace", "testkube")
		err := s.ScriptsClient.Delete(namespace, name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, err)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}

// DeleteScripts for deleting all scripts
func (s testkubeAPI) DeleteScripts() fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespace := c.Query("namespace", "testkube")
		err := s.ScriptsClient.DeleteAll(namespace)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, err)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}

func (s testkubeAPI) GetExecuteOptions(namespace, scriptID string, request testkube.ExecutionRequest) (options client.ExecuteOptions, err error) {
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

// ExecuteScript calls particular executor based on execution request content and type
func (s testkubeAPI) ExecuteScript() fiber.Handler {
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
		execution, _ := s.Repository.GetByNameAndScript(c.Context(), request.Name, scriptID)
		if execution.Name == request.Name {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("script execution with name %s already exists", request.Name))
		}

		// merge available data into execution options script spec, executor spec, request, script id
		options, err := s.GetExecuteOptions(namespace, scriptID, request)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("can't create valid execution options: %w", err))
		}

		// store execution in storage, can be get from API now
		execution = NewExecutionFromExecutionOptions(options)
		options.ID = execution.Id

		err = s.Repository.Insert(ctx, execution)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("can't create new script execution, can't insert into storage: %w", err))
		}

		// call executor rest or job based and update execution object after queueing execution
		s.Log.Infow("calling executor with options", "options", options.Request)
		execution.Start()
		err = s.Repository.StartExecution(ctx, execution.Id, execution.StartTime)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("can't create new script execution, can't insert into storage: %w", err))
		}

		result, err := s.Executor.Execute(execution, options)

		if uerr := s.Repository.UpdateResult(ctx, execution.Id, result); uerr != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("update execution error: %w", uerr))
		}

		// set execution result to one created
		execution.ExecutionResult = &result
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("script execution failed: %w", err), options)
		}

		// watch for changes run listener in async mode
		s.Log.Infow("running execution of script", "executionId", execution.Id, "request", request)

		// metrics increase
		s.Metrics.IncExecution(execution)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		return c.JSON(execution)
	}
}

func (s testkubeAPI) ExecutionLogs() fiber.Handler {
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

// ListExecutions returns array of available script executions
func (s testkubeAPI) ListExecutions() fiber.Handler {
	return func(c *fiber.Ctx) error {

		// TODO should we split this to separate endpoint? currently this one handles
		// endpoints from /executions and from /scripts/{id}/executions
		// or should scriptID be a query string as it's some kind of filter?

		filter := getFilterFromRequest(c)

		executions, err := s.Repository.GetExecutions(c.Context(), filter)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, err)
		}

		executionTotals, err := s.Repository.GetExecutionTotals(c.Context(), result.NewExecutionsFilter())
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, err)
		}

		filteredTotals, err := s.Repository.GetExecutionTotals(c.Context(), filter)
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

func getFilterFromRequest(c *fiber.Ctx) result.Filter {

	filter := result.NewExecutionsFilter()
	scriptName := c.Params("id", "-")
	if scriptName != "-" {
		filter = filter.WithScriptName(scriptName)
	}

	page, err := strconv.Atoi(c.Query("page", "-"))
	if err == nil {
		filter = filter.WithPage(page)
	}

	pageSize, err := strconv.Atoi(c.Query("pageSize", "-"))
	if err == nil && pageSize != 0 {
		filter = filter.WithPageSize(pageSize)
	}

	status := c.Query("status", "-")
	if status != "-" {
		filter = filter.WithStatus(testkube.ExecutionStatus(status))
	}

	dFilter := datefilter.NewDateFilter(c.Query("startDate", ""), c.Query("endDate", ""))
	if dFilter.IsStartValid {
		filter = filter.WithStartDate(dFilter.Start)
	}

	if dFilter.IsEndValid {
		filter = filter.WithEndDate(dFilter.End)
	}

	return filter
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

// GetExecution returns script execution object for given script and execution id
func (s testkubeAPI) GetExecution() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		scriptID := c.Params("id", "-")
		executionID := c.Params("executionID")

		var execution testkube.Execution
		var err error

		if scriptID == "-" {
			execution, err = s.Repository.Get(ctx, executionID)
			if err == mongo.ErrNoDocuments {
				return s.Error(c, http.StatusNotFound, fmt.Errorf("script with execution id %s not found", executionID))
			}
			if err != nil {
				return s.Error(c, http.StatusInternalServerError, err)
			}
		} else {
			execution, err = s.Repository.GetByNameAndScript(ctx, executionID, scriptID)
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

func (s testkubeAPI) AbortExecution() fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		return s.Executor.Abort(id)
	}
}

func (s testkubeAPI) Info() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(testkube.ServerInfo{
			Commit:  api.Commit,
			Version: api.Version,
		})
	}
}

func NewExecutionFromExecutionOptions(options client.ExecuteOptions) testkube.Execution {
	execution := testkube.NewExecution(
		options.ScriptName,
		options.Request.Name,
		options.ScriptSpec.Type_,
		options.ScriptSpec.Content,
		testkube.NewQueuedExecutionResult(),
		options.Request.Params,
	)

	execution.Repository = (*testkube.Repository)(options.ScriptSpec.Repository)

	return execution
}

// GetArtifacts returns list of files in the given bucket
func (s testkubeAPI) ListArtifacts() fiber.Handler {
	return func(c *fiber.Ctx) error {

		executionID := c.Params("executionID")
		files, err := s.Storage.ListFiles(executionID)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, err)
		}

		return c.JSON(files)
	}
}

func (s testkubeAPI) GetArtifact() fiber.Handler {
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
