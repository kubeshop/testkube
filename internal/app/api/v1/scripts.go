package v1

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	scriptsv1 "github.com/kubeshop/kubetest-operator/apis/script/v1"
	"github.com/kubeshop/kubetest/pkg/api/kubetest"
	scriptsMapper "github.com/kubeshop/kubetest/pkg/mapper/scripts"
	"github.com/kubeshop/kubetest/pkg/rand"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListScripts for getting list of all available scripts
func (s KubetestAPI) GetScript() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("id")
		namespace := c.Query("ns", "default")
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
func (s KubetestAPI) ListScripts() fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespace := c.Query("ns", "default")
		crScripts, err := s.ScriptsClient.List(namespace)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		scripts := scriptsMapper.MapScriptListKubeToAPI(*crScripts)

		return c.JSON(scripts)
	}
}

// CreateScript creates new script CR based on script content
func (s KubetestAPI) CreateScript() fiber.Handler {
	return func(c *fiber.Ctx) error {

		var request kubetest.ScriptCreateRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		script, err := s.ScriptsClient.Create(&scriptsv1.Script{
			ObjectMeta: metav1.ObjectMeta{
				Name:      request.Name,
				Namespace: request.Namespace,
			},
			Spec: scriptsv1.ScriptSpec{
				Type:    request.Type_,
				Content: request.Content,
			},
		})

		s.Metrics.IncCreateScript(script.Spec.Type, err)

		if err != nil {
			return s.Error(c, http.StatusBadGateway, err)
		}

		return c.JSON(script)
	}
}

// ExecuteScript calls particular executor based on execution request content and type
func (s KubetestAPI) ExecuteScript() fiber.Handler {
	// TODO use kube API to get registered executor details - for now it'll be fixed
	// we need to choose client based on script type in future for now there is only
	// one client postman-collection newman based executor
	// should be done on top level from some kind of available clients poll
	// for - s.ExecutorClient calls
	return func(c *fiber.Ctx) error {
		scriptID := c.Params("id")

		var request kubetest.ScriptExecutionRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("script request body invalid: %w", err))
		}

		s.Log.Infow("running execution of script", "script", request)

		// generate random execution name in case there is no one set
		// like for docker images
		if request.Name == "" {
			request.Name = rand.Name()
		}

		scriptExecution, err := s.Repository.GetByName(context.Background(), request.Name)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("can't get existing script execution: %w", err))
		}
		if scriptExecution.Name == request.Name {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("script execution with name %s already exists", request.Name))
		}

		scriptCR, err := s.ScriptsClient.Get(request.Namespace, scriptID)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("getting script CR error: %w", err))
		}

		execution, err := s.ExecutorClient.Execute(scriptCR.Spec.Content, request.Params)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, err)
		}

		ctx := context.Background()
		scriptExecution = kubetest.NewScriptExecution(
			scriptID,
			request.Name,
			execution,
			request.Params,
		)
		s.Repository.Insert(ctx, scriptExecution)

		execution, err = s.ExecutorClient.Watch(scriptExecution.Execution.Id, func(e kubetest.Execution) error {
			s.Log.Infow("saving", "status", e.Status, "scriptExecution", scriptExecution)
			scriptExecution.Execution = &e
			return s.Repository.Update(ctx, scriptExecution)
		})

		s.Metrics.IncExecution(scriptExecution)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		return c.JSON(scriptExecution)
	}
}

// ListExecutions returns array of available script executions
func (s KubetestAPI) ListExecutions() fiber.Handler {
	return func(c *fiber.Ctx) error {
		scriptID := c.Params("id")
		s.Log.Infow("Getting script executions", "id", scriptID)
		executions, err := s.Repository.GetScriptExecutions(context.Background(), scriptID)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, err)
		}

		return c.JSON(executions)
	}
}

// GetScriptExecution returns script execution object for given script and execution id
func (s KubetestAPI) GetScriptExecution() fiber.Handler {
	return func(c *fiber.Ctx) error {

		// TODO do we need scriptID here? consider removing it from API
		// It would be needed only for grouping purposes. executionID will be unique for scriptExecution
		// in API
		scriptID := c.Params("id")
		executionID := c.Params("executionID")

		s.Log.Infow("GET execution request", "id", scriptID, "executionID", executionID)

		scriptExecution, err := s.Repository.Get(context.Background(), executionID)
		if err != nil {
			return s.Error(c, http.StatusInternalServerError, err)
		}
		return c.JSON(scriptExecution)
	}
}

func (s KubetestAPI) AbortScriptExecution() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// TODO fill valid values when abort will be implemented
		s.Metrics.IncAbortScript("type", nil)
		return s.Error(c, http.StatusNotImplemented, fmt.Errorf("not implemented"))
	}
}
