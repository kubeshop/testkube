package v1

import (
	"context"
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	scriptsv1 "github.com/kubeshop/kubetest-operator/apis/script/v1"
	"github.com/kubeshop/kubetest/pkg/api/kubetest"
	"github.com/kubeshop/kubetest/pkg/executor/client"
	scriptsMapper "github.com/kubeshop/kubetest/pkg/mapper/scripts"
	"github.com/kubeshop/kubetest/pkg/rand"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s Server) GetAllScripts() fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespace := c.Query("ns", "default")
		crScripts, err := s.ScriptsClient.List(namespace)
		if err != nil {
			return err
		}

		scripts := scriptsMapper.MapScriptListKubeToAPI(*crScripts)

		return c.JSON(scripts)
	}
}

func (s Server) CreateScript() fiber.Handler {
	return func(c *fiber.Ctx) error {

		request := CreateRequest{}
		err := c.BodyParser(&request)
		if err != nil {
			return err
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

		if err != nil {
			return err
		}

		return c.JSON(script)

	}
}

func (s Server) ExecuteScript() fiber.Handler {
	// TODO use kube API to get registered executor details - for now it'll be fixed
	// we need to choose client based on script type in future for now there is only
	// one client postman-collection newman based executor
	// should be done on top level from some kind of available clients poll
	// consider moving them to separate struct - and allow to choose by executor ID
	executorClient := client.NewHTTPExecutorClient(client.DefaultURI)

	return func(c *fiber.Ctx) error {
		namespace := c.Query("ns", "default")
		scriptID := c.Params("id")
		s.Log.Infow("running execution of script", "script", scriptID)

		var request kubetest.ScriptExecutionRequest
		c.BodyParser(&request)

		if request.Name == "" {
			request.Name = rand.Name()
		}

		scriptCR, err := s.ScriptsClient.Get(namespace, scriptID)
		if err != nil {
			return err
		}

		content, err := json.Marshal(scriptCR)
		if err != nil {
			return err
		}

		execution, err := executorClient.Execute(string(content), request.Params)
		if err != nil {
			return err
		}

		ctx := context.Background()
		scriptExecution := kubetest.NewScriptExecution(
			scriptID,
			request.Name,
			execution,
			request.Params,
		)
		s.Repository.Insert(ctx, scriptExecution)

		execution, err = executorClient.Watch(scriptExecution.Execution.Id, func(e kubetest.Execution) error {
			s.Log.Infow("saving", "status", e.Status, "scriptExecution", scriptExecution)
			scriptExecution.Execution = &e
			return s.Repository.Update(ctx, scriptExecution)
		})

		if err != nil {
			return err
		}

		return c.JSON(scriptExecution)
	}
}

func (s Server) GetScriptExecutions() fiber.Handler {
	return func(c *fiber.Ctx) error {
		scriptID := c.Params("id")
		s.Log.Infow("Getting script executions", "id", scriptID)
		executions, err := s.Repository.GetScriptExecutions(context.Background(), scriptID)
		if err != nil {
			return err
		}

		return c.JSON(executions)
	}
}

func (s Server) GetScriptExecution() fiber.Handler {
	// TODO use kube API to get registered executor details - for now it'll be fixed
	// we need to choose client based on script type in future for now there is only
	// one client postman-collection newman based executor
	// should be done on top level from some kind of available clients poll
	// consider moving them to separate struct - and allow to choose by executor ID
	return func(c *fiber.Ctx) error {

		scriptID := c.Params("id")
		executionID := c.Params("executionID")
		s.Log.Infow("GET execution request", "id", scriptID, "executionID", executionID)

		// TODO do we need scriptID here? consider removing it from API
		// It would be needed only for grouping purposes. executionID will be unique for scriptExecution
		// in API
		scriptExecution, err := s.Repository.Get(context.Background(), executionID)
		if err != nil {
			return err
		}
		return c.JSON(scriptExecution)
	}
}

func (s Server) AbortScriptExecution() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return nil
	}
}
