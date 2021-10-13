package v1

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	scriptsv1 "github.com/kubeshop/testkube-operator/apis/script/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/client"
	scriptsMapper "github.com/kubeshop/testkube/pkg/mapper/scripts"

	"github.com/kubeshop/testkube/pkg/rand"
	"go.mongodb.org/mongo-driver/mongo"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

		var request testkube.ScriptCreateRequest
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
				Name:      request.Name,
				Namespace: request.Namespace,
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

func (s testkubeAPI) GetExecuteOptions(namespace, scriptID string, request testkube.ExecutionRequest) (options client.ExecuteOptions, err error) {
	// get script content from kubernetes CRs
	scriptCR, err := s.ScriptsClient.Get(namespace, scriptID)
	fmt.Printf("SCRIPT CR %+v\n", scriptCR)

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

		// get executor
		executor, err := s.Executors.Get(options.ScriptSpec.Type_)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("can't get executor: %w", err))
		}

		// call executor rest or job based and update execution object after queueing execution
		s.Log.Infow("calling executor with options", "options", options)
		result, err := executor.Execute(options)
		if uerr := s.Repository.UpdateResult(ctx, execution.Id, result); uerr != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("update execution error: %w", uerr))
		}

		// set execution from one created
		execution.ExecutionResult = &result
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("script execution failed: %w, called with options %+v", err, options))
		}

		// watch for changes run listener in async mode
		s.Log.Infow("running execution of script", "execution", execution, "request", request)
		go s.ExecutionListener(ctx, execution, executor)

		// metrics increase
		s.Metrics.IncExecution(execution)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		return c.JSON(execution)
	}
}

func (s testkubeAPI) ExecutionListener(ctx context.Context, execution testkube.Execution, executor client.ExecutorClient) {
	for event := range executor.Watch(execution.Id) {
		result := event.Result
		l := s.Log.With("executionID", execution.Id, "duration", result.Duration().String(), "scriptName", execution.ScriptName)
		l.Infow("got result event", "result", result)

		// if something changed during execution
		if event.Error != nil || result.Status != execution.ExecutionResult.Status || result.Output != execution.ExecutionResult.Output {
			l.Infow("watch - saving script execution", "oldStatus", execution.ExecutionResult.Status, "newStatus", result.Status, "result", result)
			l.Debugw("watch - saving script execution - debug", "execution", execution)

			err := s.Repository.UpdateResult(ctx, execution.Id, result)
			if err != nil {
				s.Log.Errorw("update execution error", err.Error())
			}
		}
	}

	s.Log.Infow("watch execution completed", "executionID", execution.Id, "status", execution.ExecutionResult.Status)
}

// ListExecutions returns array of available script executions
func (s testkubeAPI) ListExecutions() fiber.Handler {
	return func(c *fiber.Ctx) error {

		scriptID := c.Params("id", "-")
		pageSize, err := strconv.Atoi(c.Query("pageSize", "100"))
		if err != nil {
			pageSize = 100
		} else if pageSize < 1 || pageSize > 1000 {
			pageSize = 1000
		}

		page, err := strconv.Atoi(c.Query("page", "0"))
		if err != nil {
			page = 0
		}

		statusFilter := c.Query("status", "")

		dFilter := NewDateFilter(c.Query("startDate", ""), c.Query("endDate", ""))

		ctx := c.Context()

		var executions []testkube.Execution

		// TODO should we split this to separate endpoint? currently this one handles
		// endpoints from /executions and from /scripts/{id}/executions
		// or should scriptID be a query string as it's some kind of filter?
		if scriptID == "-" {
			s.Log.Infow("Getting script executions (no id passed)")
			executions, err = s.Repository.GetNewestExecutions(ctx, 10000)
		} else {
			s.Log.Infow("Getting script executions", "id", scriptID)
			executions, err = s.Repository.GetExecutions(ctx, scriptID)
		}
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, err)
		}

		results := createListExecutionsResult(executions, statusFilter, dFilter, page, pageSize)

		return c.JSON(results)
	}
}

func createListExecutionsResult(executions []testkube.Execution, statusFilter string, dFilter DateFilter, page int, pageSize int) testkube.ExecutionsResult {
	totals := testkube.ExecutionsTotals{
		Results: int32(len(executions)),
		Passed:  0,
		Failed:  0,
		Queued:  0,
		Pending: 0,
	}

	executionResults := make([]testkube.ExecutionSummary, pageSize)
	addedToResultCount := 0
	filteredCount := 0

	for _, s := range executions {

		// TODO move it to mapper with valid error handling
		// it could kill api server with panic in case of empty
		// Execution result - for now omit failed result
		if s.ExecutionResult == nil || s.ExecutionResult.Status == nil {
			continue
		}

		switch *s.ExecutionResult.Status {
		case testkube.QUEUED_ExecutionStatus:
			totals.Queued++
		case testkube.SUCCESS_ExecutionStatus:
			totals.Passed++
		case testkube.ERROR__ExecutionStatus:
			totals.Failed++
		case testkube.PENDING_ExecutionStatus:
			totals.Pending++
		}

		isPassingStatusFilter := (statusFilter == "" || string(*s.ExecutionResult.Status) == statusFilter)
		if addedToResultCount < pageSize &&
			isPassingStatusFilter &&
			dFilter.IsPassing(s.ExecutionResult.StartTime) {
			if filteredCount == page*pageSize {
				executionResults[addedToResultCount] = testkube.ExecutionSummary{
					Id:         s.Id,
					Name:       s.Name,
					ScriptName: s.ScriptName,
					ScriptType: s.ScriptType,
					Status:     s.ExecutionResult.Status,
					StartTime:  s.ExecutionResult.StartTime,
					EndTime:    s.ExecutionResult.EndTime,
				}
				addedToResultCount++
			} else {
				filteredCount++
			}
		}
	}

	return testkube.ExecutionsResult{
		Totals:  &totals,
		Results: executionResults[0:addedToResultCount],
	}
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

		s.Log.Infow("get script execution request", "id", scriptID, "executionID", executionID, "execution", execution)

		return c.JSON(execution)
	}
}

func (s testkubeAPI) AbortExecution() fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")

		// get script execution by id to get executor type
		execution, err := s.Repository.Get(c.Context(), id)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("can't get execution id:%s, error:%w", id, err))
		}

		executor, err := s.Executors.Get(execution.ScriptType)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("can't get executor: %w", err))
		}

		return executor.Abort(id)
	}
}

func NewExecutionFromExecutionOptions(options client.ExecuteOptions) testkube.Execution {
	execution := testkube.NewExecution(
		options.ScriptName,
		options.Request.Name,
		options.ScriptSpec.Type_,
		testkube.NewResult(),
		options.Request.Params,
	)

	return execution
}
