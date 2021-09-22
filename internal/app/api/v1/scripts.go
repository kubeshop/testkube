package v1

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	scriptsv1 "github.com/kubeshop/kubtest-operator/apis/script/v1"
	"github.com/kubeshop/kubtest/pkg/api/kubtest"
	"github.com/kubeshop/kubtest/pkg/executor/client"
	executionsMapper "github.com/kubeshop/kubtest/pkg/mapper/executions"
	scriptsMapper "github.com/kubeshop/kubtest/pkg/mapper/scripts"

	"github.com/kubeshop/kubtest/pkg/rand"
	"go.mongodb.org/mongo-driver/mongo"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListScripts for getting list of all available scripts
func (s kubtestAPI) GetScript() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("id")
		namespace := c.Query("namespace", "default")
		crScript, err := s.ScriptsClient.Get(namespace, name)
		if err != nil {
			if errors.IsNotFound(err) {
				return s.Error(c, http.StatusNotFound, err)
			}

			return s.Error(c, http.StatusBadGateway, err)
		}

		scripts := scriptsMapper.MapScriptKubeToAPI(*crScript)

		return c.JSON(scripts)
	}
}

// ListScripts for getting list of all available scripts
func (s kubtestAPI) ListScripts() fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespace := c.Query("namespace", "default")
		crScripts, err := s.ScriptsClient.List(namespace)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		scripts := scriptsMapper.MapScriptListKubeToAPI(*crScripts)

		return c.JSON(scripts)
	}
}

// CreateScript creates new script CR based on script content
func (s kubtestAPI) CreateScript() fiber.Handler {
	return func(c *fiber.Ctx) error {

		var request kubtest.ScriptCreateRequest
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

func (s kubtestAPI) GetExecuteOptions(namespace, scriptID string, request kubtest.ExecutionRequest) (options client.ExecuteOptions, err error) {
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
		ID:           scriptID,
		ScriptSpec:   scriptCR.Spec,
		ExecutorSpec: executorCR.Spec,
		Request:      request,
	}, nil
}

// ExecuteScript calls particular executor based on execution request content and type
func (s kubtestAPI) ExecuteScript() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()

		var request kubtest.ExecutionRequest
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
		execution = NewScriptExecutionFromExecutionOptions(options)
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
		execution.Result = &result
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

func (s kubtestAPI) ExecutionListener(ctx context.Context, execution kubtest.Execution, executor client.ExecutorClient) {
	for event := range executor.Watch(execution.Result.Id) {
		result := event.Result
		l := s.Log.With("executionID", execution.Id, "duration", result.Duration().String(), "scriptName", execution.ScriptName)
		l.Infow("got execution event", "event", result)

		// if something changed during execution
		if event.Error != nil || result.Status != execution.Result.Status || result.Result.Output != execution.Result.Result.Output {
			l.Infow("watch - saving script execution", "oldStatus", execution.Result.Status, "newStatus", result.Status, "result", result.Result)
			l.Debugw("watch - saving script execution - debug", "execution", execution)

			err := s.Repository.UpdateResult(ctx, execution.Id, result)
			if err != nil {
				s.Log.Errorw("update execution error", err.Error())
			}
		}
	}

	s.Log.Infow("watch execution completed", "executionID", execution.Id, "status", execution.Result.Status)
}

// ListExecutions returns array of available script executions
func (s kubtestAPI) ListExecutions() fiber.Handler {
	return func(c *fiber.Ctx) error {

		scriptID := c.Params("id", "-")
		pager := s.GetPager(c)
		l := s.Log.With("script", scriptID, "pager", pager)
		ctx := c.Context()

		var executions []kubtest.Execution
		var err error

		// TODO should we split this to separate endpoint? currently this one handles
		// endpoints from /executions and from /scripts/{id}/executions
		// or should scriptID be a query string as it's some kind of filter?
		if scriptID == "-" {
			l.Infow("Getting script executions (no id passed)")
			executions, err = s.Repository.GetNewestExecutions(ctx, pager.Limit)
		} else {
			l.Infow("Getting script executions")
			executions, err = s.Repository.GetScriptExecutions(ctx, scriptID)
		}
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, err)
		}

		// convert to summary
		result := executionsMapper.MapToSummary(executions)

		return c.JSON(result)
	}
}

// GetScriptExecution returns script execution object for given script and execution id
func (s kubtestAPI) GetScriptExecution() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		scriptID := c.Params("id", "-")
		executionID := c.Params("executionID")

		var execution kubtest.Execution
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

func (s kubtestAPI) AbortExecution() fiber.Handler {
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

func NewScriptExecutionFromExecutionOptions(options client.ExecuteOptions) kubtest.Execution {
	return kubtest.NewScriptExecution(
		options.ScriptSpec.Name,
		options.Request.Name,
		options.ScriptSpec.Type_,
		kubtest.NewResult(),
		options.Request.Params,
	)
}
