package v1

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/agent"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/scheduler"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowresolver"
	"github.com/kubeshop/testkube/pkg/workerpool"
)

func (s *TestkubeAPI) ListTestWorkflowsHandler() fiber.Handler {
	errPrefix := "failed to list test workflows"
	return func(c *fiber.Ctx) (err error) {
		workflows, err := s.getFilteredTestWorkflowList(c)
		if err != nil {
			return s.BadGateway(c, errPrefix, "client problem", err)
		}
		err = SendResourceList(c, "TestWorkflow", testworkflowsv1.GroupVersion, testworkflows.MapTestWorkflowKubeToAPI, workflows.Items...)
		if err != nil {
			return s.InternalError(c, errPrefix, "serialization problem", err)
		}
		return
	}
}

func (s *TestkubeAPI) GetTestWorkflowHandler() fiber.Handler {
	return func(c *fiber.Ctx) (err error) {
		name := c.Params("id")
		errPrefix := fmt.Sprintf("failed to get test workflow '%s'", name)
		workflow, err := s.TestWorkflowsClient.Get(name)
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}
		err = SendResource(c, "TestWorkflow", testworkflowsv1.GroupVersion, testworkflows.MapKubeToAPI, workflow)
		if err != nil {
			return s.InternalError(c, errPrefix, "serialization problem", err)
		}
		return
	}
}

func (s *TestkubeAPI) DeleteTestWorkflowHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("id")
		errPrefix := fmt.Sprintf("failed to delete test workflow '%s'", name)
		skipCRD := c.Query("skipDeleteCRD", "")
		if skipCRD != "true" {
			err := s.TestWorkflowsClient.Delete(name)
			s.Metrics.IncDeleteTestWorkflow(err)
			if err != nil {
				return s.ClientError(c, errPrefix, err)
			}
		}
		skipExecutions := c.Query("skipDeleteExecutions", "")
		if skipExecutions != "true" {
			err := s.TestWorkflowOutput.DeleteOutputByTestWorkflow(context.Background(), name)
			if err != nil {
				return s.ClientError(c, "deleting executions output", err)
			}
			err = s.TestWorkflowResults.DeleteByTestWorkflow(context.Background(), name)
			if err != nil {
				return s.ClientError(c, "deleting executions", err)
			}
		}
		return c.SendStatus(http.StatusNoContent)
	}
}

func (s *TestkubeAPI) DeleteTestWorkflowsHandler() fiber.Handler {
	errPrefix := "failed to delete test workflows"
	return func(c *fiber.Ctx) error {
		selector := c.Query("selector")

		var (
			workflows *testworkflowsv1.TestWorkflowList
			err       error
		)
		testWorkflowNames := c.Query("testWorkflowNames")
		if testWorkflowNames != "" {
			names := strings.Split(testWorkflowNames, ",")
			workflows = &testworkflowsv1.TestWorkflowList{}
			for _, name := range names {
				workflow, err := s.TestWorkflowsClient.Get(name)
				if err != nil {
					return s.ClientError(c, errPrefix, err)
				}
				workflows.Items = append(workflows.Items, *workflow)
			}
		} else {
			workflows, err = s.TestWorkflowsClient.List(selector)
			if err != nil {
				return s.BadGateway(c, errPrefix, "client problem", err)
			}
		}

		// Delete
		err = s.TestWorkflowsClient.DeleteByLabels(selector)
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}

		// Mark as deleted
		for range workflows.Items {
			s.Metrics.IncDeleteTestWorkflow(err)
		}

		// Delete the executions
		skipExecutions := c.Query("skipDeleteExecutions", "")
		if skipExecutions != "true" {
			names := common.MapSlice(workflows.Items, func(t testworkflowsv1.TestWorkflow) string {
				return t.Name
			})

			err = s.TestWorkflowOutput.DeleteOutputForTestWorkflows(context.Background(), names)
			if err != nil {
				return s.ClientError(c, "deleting executions output", err)
			}
			err = s.TestWorkflowResults.DeleteByTestWorkflows(context.Background(), names)
			if err != nil {
				return s.ClientError(c, "deleting executions", err)
			}
		}

		return c.SendStatus(http.StatusNoContent)
	}
}

func (s *TestkubeAPI) CreateTestWorkflowHandler() fiber.Handler {
	errPrefix := "failed to create test workflow"
	return func(c *fiber.Ctx) (err error) {
		// Deserialize resource
		obj := new(testworkflowsv1.TestWorkflow)
		if HasYAML(c) {
			err = common.DeserializeCRD(obj, c.Body())
			if err != nil {
				return s.BadRequest(c, errPrefix, "invalid body", err)
			}
		} else {
			var v *testkube.TestWorkflow
			err = c.BodyParser(&v)
			if err != nil {
				return s.BadRequest(c, errPrefix, "invalid body", err)
			}
			obj = testworkflows.MapAPIToKube(v)
		}

		// Validate resource
		if obj == nil || obj.Name == "" {
			return s.BadRequest(c, errPrefix, "invalid body", errors.New("name is required"))
		}
		obj.Namespace = s.Namespace

		// Get information about execution namespace
		// TODO: Think what to do when it is dynamic - create in all execution namespaces?
		execNamespace := obj.Namespace
		if obj.Spec.Job != nil && obj.Spec.Job.Namespace != "" {
			execNamespace = obj.Spec.Job.Namespace
		}

		// Handle secrets auto-creation
		secrets := s.SecretManager.Batch(execNamespace, "tw-", obj.Name)
		err = testworkflowresolver.ExtractCredentialsInWorkflow(obj, secrets.Append)
		if err != nil {
			return s.BadRequest(c, errPrefix, "auto-creating secrets", err)
		}

		// Create the resource
		obj, err = s.TestWorkflowsClient.Create(obj)
		if err != nil {
			s.Metrics.IncCreateTestWorkflow(err)
			return s.BadRequest(c, errPrefix, "client error", err)
		}

		// Create secrets
		err = secrets.Create(c.Context(), &metav1.OwnerReference{
			APIVersion: testworkflowsv1.GroupVersion.String(),
			Kind:       testworkflowsv1.Resource,
			Name:       obj.Name,
			UID:        obj.UID,
		})
		s.Metrics.IncCreateTestWorkflow(err)
		if err != nil {
			_ = s.TestWorkflowsClient.Delete(obj.Name)
			return s.BadRequest(c, errPrefix, "auto-creating secrets", err)
		}

		s.sendCreateWorkflowTelemetry(c.Context(), obj)

		err = SendResource(c, "TestWorkflow", testworkflowsv1.GroupVersion, testworkflows.MapKubeToAPI, obj)
		if err != nil {
			return s.InternalError(c, errPrefix, "serialization problem", err)
		}
		return
	}
}

func (s *TestkubeAPI) UpdateTestWorkflowHandler() fiber.Handler {
	errPrefix := "failed to update test workflow"
	return func(c *fiber.Ctx) (err error) {
		name := c.Params("id")

		// Deserialize resource
		obj := new(testworkflowsv1.TestWorkflow)
		if HasYAML(c) {
			err = common.DeserializeCRD(obj, c.Body())
			if err != nil {
				return s.BadRequest(c, errPrefix, "invalid body", err)
			}
		} else {
			var v *testkube.TestWorkflow
			err = c.BodyParser(&v)
			if err != nil {
				return s.BadRequest(c, errPrefix, "invalid body", err)
			}
			obj = testworkflows.MapAPIToKube(v)
		}

		// Read existing resource
		workflow, err := s.TestWorkflowsClient.Get(name)
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}
		initial := workflow.DeepCopy()

		// Validate resource
		if obj == nil {
			return s.BadRequest(c, errPrefix, "invalid body", errors.New("body is required"))
		}
		obj.Namespace = workflow.Namespace
		obj.Name = workflow.Name
		obj.ResourceVersion = workflow.ResourceVersion

		// Get information about execution namespace
		// TODO: Think what to do when it is dynamic - create in all execution namespaces?
		execNamespace := obj.Namespace
		if obj.Spec.Job != nil && obj.Spec.Job.Namespace != "" {
			execNamespace = obj.Spec.Job.Namespace
		}

		// Handle secrets auto-creation
		secrets := s.SecretManager.Batch(execNamespace, "tw-", obj.Name)
		err = testworkflowresolver.ExtractCredentialsInWorkflow(obj, secrets.Append)
		if err != nil {
			return s.BadRequest(c, errPrefix, "auto-creating secrets", err)
		}

		// Update the resource
		obj, err = s.TestWorkflowsClient.Update(obj)
		if err != nil {
			s.Metrics.IncUpdateTestWorkflow(err)
			return s.BadRequest(c, errPrefix, "client error", err)
		}

		// Create secrets
		err = secrets.Create(c.Context(), &metav1.OwnerReference{
			APIVersion: testworkflowsv1.GroupVersion.String(),
			Kind:       testworkflowsv1.Resource,
			Name:       obj.Name,
			UID:        obj.UID,
		})
		s.Metrics.IncUpdateTestWorkflow(err)
		if err != nil {
			_, err = s.TestWorkflowsClient.Update(initial)
			if err != nil {
				s.Log.Errorf("failed to recover previous TestWorkflow state: %v", err)
			}
			return s.BadRequest(c, errPrefix, "auto-creating secrets", err)
		}

		err = SendResource(c, "TestWorkflow", testworkflowsv1.GroupVersion, testworkflows.MapKubeToAPI, obj)
		if err != nil {
			return s.InternalError(c, errPrefix, "serialization problem", err)
		}
		return
	}
}

func (s *TestkubeAPI) PreviewTestWorkflowHandler() fiber.Handler {
	errPrefix := "failed to resolve test workflow"
	return func(c *fiber.Ctx) (err error) {
		// Check if it should inline templates
		inline, _ := strconv.ParseBool(c.Query("inline"))

		// Deserialize resource
		obj := new(testworkflowsv1.TestWorkflow)
		if HasYAML(c) {
			err = common.DeserializeCRD(obj, c.Body())
			if err != nil {
				return s.BadRequest(c, errPrefix, "invalid body", err)
			}
		} else {
			var v *testkube.TestWorkflow
			err = c.BodyParser(&v)
			if err != nil {
				return s.BadRequest(c, errPrefix, "invalid body", err)
			}
			obj = testworkflows.MapAPIToKube(v)
		}

		// Validate resource
		if obj == nil {
			return s.BadRequest(c, errPrefix, "invalid body", errors.New("name is required"))
		}
		obj.Namespace = s.Namespace

		if inline {
			// Fetch the templates
			tpls := testworkflowresolver.ListTemplates(obj)
			tplsMap := make(map[string]testworkflowsv1.TestWorkflowTemplate, len(tpls))
			for name := range tpls {
				tpl, err := s.TestWorkflowTemplatesClient.Get(name)
				if err != nil {
					return s.BadRequest(c, errPrefix, "fetching error", err)
				}
				tplsMap[name] = *tpl
			}

			// Resolve the TestWorkflow
			err = testworkflowresolver.ApplyTemplates(obj, tplsMap)
			if err != nil {
				return s.BadRequest(c, errPrefix, "resolving error", err)
			}
		}

		err = SendResource(c, "TestWorkflow", testworkflowsv1.GroupVersion, testworkflows.MapKubeToAPI, obj)
		if err != nil {
			return s.InternalError(c, errPrefix, "serialization problem", err)
		}
		return
	}
}

// TODO: Add metrics
func (s *TestkubeAPI) ExecuteTestWorkflowHandler() fiber.Handler {
	return func(c *fiber.Ctx) (err error) {
		// pass metadata to context
		ctx := agent.Context(c.Context(), *s.proContext)
		name := c.Params("id")
		errPrefix := fmt.Sprintf("failed to execute test workflow '%s'", name)
		workflow, err := s.TestWorkflowsClient.Get(name)
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}

		// Load the execution request
		var request testkube.TestWorkflowExecutionRequest
		err = c.BodyParser(&request)
		if err != nil && !errors.Is(err, fiber.ErrUnprocessableEntity) {
			return s.BadRequest(c, errPrefix, "invalid body", err)
		}

		log.DefaultLogger.Infow("TestWorkflow execution request", "name", name, "request", request)

		var results []testkube.TestWorkflowExecution
		var errs []error

		request.TestWorkflowExecutionName = strings.Clone(c.Query("testWorkflowExecutionName"))
		concurrencyLevel := scheduler.DefaultConcurrencyLevel
		workerpoolService := workerpool.New[testworkflowsv1.TestWorkflow, testkube.TestWorkflowExecutionRequest,
			testkube.TestWorkflowExecution](concurrencyLevel)
		requests := []workerpool.Request[testworkflowsv1.TestWorkflow, testkube.TestWorkflowExecutionRequest, testkube.TestWorkflowExecution]{
			{
				Object:  *workflow,
				Options: request,
				ExecFn:  s.TestWorkflowExecutor.Execute,
			},
		}

		go workerpoolService.SendRequests(requests)
		go workerpoolService.Run(ctx)

		for r := range workerpoolService.GetResponses() {
			results = append(results, r.Result)
			if r.Err != nil {
				errs = append(errs, r.Err)
			}
		}

		if len(errs) != 0 {
			return s.InternalError(c, errPrefix, "execution error", errs[0])
		}

		if len(results) != 0 {
			return c.JSON(results[0])
		}

		return s.InternalError(c, errPrefix, "error", errors.New("no execution results"))
	}
}

func (s *TestkubeAPI) getFilteredTestWorkflowList(c *fiber.Ctx) (*testworkflowsv1.TestWorkflowList, error) {
	crWorkflows, err := s.TestWorkflowsClient.List(c.Query("selector"))
	if err != nil {
		return nil, err
	}

	search := c.Query("textSearch")
	if search != "" {
		// filter items array
		for i := len(crWorkflows.Items) - 1; i >= 0; i-- {
			if !strings.Contains(crWorkflows.Items[i].Name, search) {
				crWorkflows.Items = append(crWorkflows.Items[:i], crWorkflows.Items[i+1:]...)
			}
		}
	}

	return crWorkflows, nil
}
