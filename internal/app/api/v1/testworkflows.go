package v1

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowresolver"
)

func (s *TestkubeAPI) ListTestWorkflowsHandler() fiber.Handler {
	errPrefix := "failed to list test workflows"
	return func(c *fiber.Ctx) (err error) {
		workflows, err := s.getFilteredTestWorkflowList(c)
		if err != nil {
			return s.BadGateway(c, errPrefix, "client problem", err)
		}
		crWorkflows := common.MapSlice(workflows, func(w testkube.TestWorkflow) testworkflowsv1.TestWorkflow {
			return *testworkflows.MapAPIToKube(&w)
		})
		err = SendResourceList(c, "TestWorkflow", testworkflowsv1.GroupVersion, testworkflows.MapTestWorkflowKubeToAPI, crWorkflows...)
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
		workflow, err := s.TestWorkflowsClient.Get(c.Context(), s.getEnvironmentId(), name)
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}
		err = SendResource(c, "TestWorkflow", testworkflowsv1.GroupVersion, testworkflows.MapKubeToAPI, testworkflows.MapAPIToKube(workflow))
		if err != nil {
			return s.InternalError(c, errPrefix, "serialization problem", err)
		}
		return
	}
}

func (s *TestkubeAPI) DeleteTestWorkflowHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		environmentId := s.getEnvironmentId()

		name := c.Params("id")
		errPrefix := fmt.Sprintf("failed to delete test workflow '%s'", name)
		skipCRD := c.Query("skipDeleteCRD", "")
		if skipCRD != "true" {
			err := s.TestWorkflowsClient.Delete(ctx, environmentId, name)
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
		ctx := c.Context()
		environmentId := s.getEnvironmentId()

		selector := c.Query("selector")
		labelSelector, err := metav1.ParseToLabelSelector(selector)
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}
		if len(labelSelector.MatchExpressions) > 0 {
			return s.ClientError(c, errPrefix, errors.New("matchExpressions are not supported"))
		}

		workflows := make([]testkube.TestWorkflow, 0)
		testWorkflowNames := c.Query("testWorkflowNames")
		if testWorkflowNames != "" {
			names := strings.Split(testWorkflowNames, ",")
			for _, name := range names {
				workflow, err := s.TestWorkflowsClient.Get(ctx, environmentId, name)
				if err != nil {
					return s.ClientError(c, errPrefix, err)
				}
				workflows = append(workflows, *workflow)
			}
		} else {
			workflows, err = s.TestWorkflowsClient.List(ctx, environmentId, testworkflowclient.ListOptions{
				Labels: labelSelector.MatchLabels,
			})
			if err != nil {
				return s.BadGateway(c, errPrefix, "client problem", err)
			}
		}

		// Delete
		_, err = s.TestWorkflowsClient.DeleteByLabels(ctx, environmentId, labelSelector.MatchLabels)
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}

		// Mark as deleted
		for range workflows {
			s.Metrics.IncDeleteTestWorkflow(err)
		}

		// Delete the executions
		skipExecutions := c.Query("skipDeleteExecutions", "")
		if skipExecutions != "true" {
			names := common.MapSlice(workflows, func(t testkube.TestWorkflow) string {
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
		ctx := c.Context()
		environmentId := s.getEnvironmentId()

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
		secrets := s.SecretManager.Batch("tw-", obj.Name)
		err = testworkflowresolver.ExtractCredentialsInWorkflow(obj, secrets.Append)
		if err != nil {
			return s.BadRequest(c, errPrefix, "auto-creating secrets", err)
		}

		// Create the resource
		err = s.TestWorkflowsClient.Create(ctx, environmentId, *testworkflows.MapKubeToAPI(obj))
		if err != nil {
			s.Metrics.IncCreateTestWorkflow(err)
			return s.BadRequest(c, errPrefix, "client error", err)
		}

		// Create secrets
		err = s.SecretManager.InsertBatch(ctx, execNamespace, secrets, &metav1.OwnerReference{
			APIVersion: testworkflowsv1.GroupVersion.String(),
			Kind:       testworkflowsv1.Resource,
			Name:       obj.Name,
			UID:        obj.UID,
		})
		s.Metrics.IncCreateTestWorkflow(err)
		if err != nil {
			_ = s.TestWorkflowsClient.Delete(context.Background(), environmentId, obj.Name)
			return s.BadRequest(c, errPrefix, "auto-creating secrets", err)
		}

		s.sendCreateWorkflowTelemetry(ctx, obj)

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
		ctx := c.Context()
		environmentId := s.getEnvironmentId()

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
		workflow, err := s.TestWorkflowsClient.Get(ctx, environmentId, name)
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

		// Get information about execution namespace
		// TODO: Think what to do when it is dynamic - create in all execution namespaces?
		execNamespace := obj.Namespace
		if obj.Spec.Job != nil && obj.Spec.Job.Namespace != "" {
			execNamespace = obj.Spec.Job.Namespace
		}

		// Handle secrets auto-creation
		secrets := s.SecretManager.Batch("tw-", obj.Name)
		err = testworkflowresolver.ExtractCredentialsInWorkflow(obj, secrets.Append)
		if err != nil {
			return s.BadRequest(c, errPrefix, "auto-creating secrets", err)
		}

		// Update the resource
		err = s.TestWorkflowsClient.Update(ctx, environmentId, *testworkflows.MapKubeToAPI(obj))
		if err != nil {
			s.Metrics.IncUpdateTestWorkflow(err)
			return s.BadRequest(c, errPrefix, "client error", err)
		}

		// Create secrets
		err = s.SecretManager.InsertBatch(c.Context(), execNamespace, secrets, &metav1.OwnerReference{
			APIVersion: testworkflowsv1.GroupVersion.String(),
			Kind:       testworkflowsv1.Resource,
			Name:       obj.Name,
			UID:        obj.UID,
		})
		s.Metrics.IncUpdateTestWorkflow(err)
		if err != nil {
			err = s.TestWorkflowsClient.Update(context.Background(), environmentId, *initial)
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
		ctx := c.Context()
		environmentId := s.getEnvironmentId()

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
			tplsMap := make(map[string]*testworkflowsv1.TestWorkflowTemplate, len(tpls))
			for name := range tpls {
				tpl, err := s.TestWorkflowTemplatesClient.Get(ctx, environmentId, name)
				if err != nil {
					return s.BadRequest(c, errPrefix, "fetching error", err)
				}
				tplsMap[name] = testworkflows.MapTemplateAPIToKube(tpl)
			}

			// Resolve the TestWorkflow
			secrets := s.SecretManager.Batch("tw-", obj.Name)
			err = testworkflowresolver.ApplyTemplates(obj, tplsMap, testworkflowresolver.EnvVarSourceToSecretExpression(secrets.Append))
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
		ctx := c.Context()
		name := c.Params("id")
		selector := c.Query("selector")
		s.Log.Debugw("getting test workflow", "name", name, "selector", selector)

		errPrefix := "failed to execute test workflow"

		// Load the execution request
		var request testkube.TestWorkflowExecutionRequest
		err = c.BodyParser(&request)
		if err != nil && !errors.Is(err, fiber.ErrUnprocessableEntity) {
			return s.BadRequest(c, errPrefix, "invalid body", err)
		}

		runningContext, user := testworkflowexecutor.GetNewRunningContext(request.RunningContext, request.ParentExecutionIds)

		var scheduleExecution cloud.ScheduleExecution
		if request.Target != nil {
			target := &cloud.ExecutionTarget{ReplicateBy: request.Target.ReplicateBy}
			if request.Target.Match != nil {
				target.Match = make(map[string]*cloud.ExecutionTargetMatch)
				for k, v := range request.Target.Match {
					target.Match[k] = &cloud.ExecutionTargetMatch{OneOf: v.OneOf, NotOneOf: v.NotOneOf}
				}
			}
			scheduleExecution.Targets = []*cloud.ExecutionTarget{target}
		}
		if name != "" {
			scheduleExecution.Selector = &cloud.ScheduleResourceSelector{Name: name}
			scheduleExecution.Config = request.Config
		} else if selector != "" {
			sel, err := metav1.ParseToLabelSelector(selector)
			if err != nil {
				return s.InternalError(c, errPrefix, "invalid selector", err)
			}
			if len(sel.MatchExpressions) > 0 {
				return s.InternalError(c, errPrefix, "invalid selector", errors.New("only simple selectors are allowed"))
			}
			scheduleExecution.Selector = &cloud.ScheduleResourceSelector{Labels: sel.MatchLabels}
			scheduleExecution.Config = request.Config
		}

		resp := s.testWorkflowExecutor.Execute(ctx, "", &cloud.ScheduleRequest{
			Executions:           []*cloud.ScheduleExecution{&scheduleExecution},
			DisableWebhooks:      request.DisableWebhooks,
			Tags:                 request.Tags,
			RunningContext:       runningContext,
			ParentExecutionIds:   request.ParentExecutionIds,
			KubernetesObjectName: request.TestWorkflowExecutionName,
			User:                 user,
		})

		results := make([]testkube.TestWorkflowExecution, 0)
		for v := range resp.Channel() {
			results = append(results, *v)
		}

		if resp.Error() != nil {
			return s.InternalError(c, errPrefix, "execution error", resp.Error())
		}

		s.Log.Debugw("executing test workflow", "name", name, "selector", selector)
		if len(results) != 0 {
			if name != "" {
				return c.JSON(results[0])
			}

			return c.JSON(results)
		}

		return s.InternalError(c, errPrefix, "error", errors.New("no execution results"))
	}
}

func (s *TestkubeAPI) getFilteredTestWorkflowList(c *fiber.Ctx) ([]testkube.TestWorkflow, error) {
	ctx := c.Context()
	environmentId := s.getEnvironmentId()

	selector := c.Query("selector")
	labelSelector, err := metav1.ParseToLabelSelector(selector)
	if err != nil {
		return nil, err
	}
	if len(labelSelector.MatchExpressions) > 0 {
		return nil, errors.New("MatchExpressions are not supported")
	}

	workflows, err := s.TestWorkflowsClient.List(ctx, environmentId, testworkflowclient.ListOptions{
		Labels: labelSelector.MatchLabels,
	})
	if err != nil {
		return nil, err
	}

	search := c.Query("textSearch")
	if search != "" {
		// filter items array
		for i := len(workflows) - 1; i >= 0; i-- {
			if !strings.Contains(workflows[i].Name, search) {
				workflows = append(workflows[:i], workflows[i+1:]...)
			}
		}
	}
	return workflows, nil
}
