package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/internal/crdcommon"
	opcrd "github.com/kubeshop/testkube/k8s"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/controlplane"
	"github.com/kubeshop/testkube/pkg/crd"
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
			err = crdcommon.DeserializeCRD(obj, c.Body())
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
		if secrets.HasData() {
			uid, _ := s.TestWorkflowsClient.GetKubernetesObjectUID(ctx, environmentId, obj.Name)
			var ref *metav1.OwnerReference
			if uid != "" {
				ref = &metav1.OwnerReference{
					APIVersion: testworkflowsv1.GroupVersion.String(),
					Kind:       testworkflowsv1.Resource,
					Name:       obj.Name,
					UID:        uid,
				}
			}
			err = s.SecretManager.InsertBatch(ctx, execNamespace, secrets, ref)
			if err != nil {
				_ = s.TestWorkflowsClient.Delete(context.Background(), environmentId, obj.Name)
				return s.BadRequest(c, errPrefix, "auto-creating secrets", err)
			}
		}
		s.Metrics.IncCreateTestWorkflow(err)

		s.sendCreateWorkflowTelemetry(ctx, obj)

		err = SendResource(c, "TestWorkflow", testworkflowsv1.GroupVersion, testworkflows.MapKubeToAPI, obj)
		if err != nil {
			return s.InternalError(c, errPrefix, "serialization problem", err)
		}
		return
	}
}

func (s *TestkubeAPI) ValidateTestWorkflowHandler() fiber.Handler {
	errPrefix := "failed to validate test workflow"
	return func(c *fiber.Ctx) (err error) {
		// Deserialize resource
		body := c.Body()
		obj := new(testworkflowsv1.TestWorkflow)
		if err = crdcommon.DeserializeCRD(obj, body); err != nil {
			return s.BadRequest(c, errPrefix, "invalid body", err)
		}

		// Validate resource
		if obj.Kind != "" && obj.Kind != "TestWorkflow" {
			return s.BadRequest(c, errPrefix, "invalid meta", errors.New("only TestWorkflow object is accepted"))
		}

		if obj.APIVersion != "" && obj.APIVersion != "testworkflows.testkube.io/v1" {
			return s.BadRequest(c, errPrefix, "invalid meta", errors.New("only TestWorkflow version v1 is accepted"))
		}

		if obj.Name == "" {
			return s.BadRequest(c, errPrefix, "invalid name", errors.New("name is required"))
		}

		// Validate spec
		if err = crd.ValidateYAMLAgainstSchema(opcrd.SchemaTestWorkflow, body); err != nil {
			return s.BadRequest(c, errPrefix, "invalid spec", err)
		}

		c.Status(http.StatusNoContent)
		return nil
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
			err = crdcommon.DeserializeCRD(obj, c.Body())
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

		// Create secrets if necessary
		if secrets.HasData() {
			uid, _ := s.TestWorkflowsClient.GetKubernetesObjectUID(ctx, environmentId, obj.Name)
			var ref *metav1.OwnerReference
			if uid != "" {
				ref = &metav1.OwnerReference{
					APIVersion: testworkflowsv1.GroupVersion.String(),
					Kind:       testworkflowsv1.Resource,
					Name:       obj.Name,
					UID:        uid,
				}
			}
			err = s.SecretManager.InsertBatch(c.Context(), execNamespace, secrets, ref)
			if err != nil {
				err = s.TestWorkflowsClient.Update(context.Background(), environmentId, *initial)
				if err != nil {
					s.Log.Errorf("failed to recover previous TestWorkflow state: %v", err)
				}
				return s.BadRequest(c, errPrefix, "auto-creating secrets", err)
			}
		}
		s.Metrics.IncUpdateTestWorkflow(err)

		err = SendResource(c, "TestWorkflow", testworkflowsv1.GroupVersion, testworkflows.MapKubeToAPI, obj)
		if err != nil {
			return s.InternalError(c, errPrefix, "serialization problem", err)
		}
		return
	}
}

func (s *TestkubeAPI) UpdateTestWorkflowStatusHandler() fiber.Handler {
	errPrefix := "failed to update test workflow status"
	return func(c *fiber.Ctx) (err error) {
		ctx := c.Context()
		environmentId := s.getEnvironmentId()

		name := c.Params("id")

		// Deserialize resource
		var workflow *testkube.TestWorkflow
		err = c.BodyParser(&workflow)
		if err != nil {
			return s.BadRequest(c, errPrefix, "invalid body", err)
		}

		// Validate resource
		if workflow == nil || workflow.Status == nil {
			return s.BadRequest(c, errPrefix, "invalid body", errors.New("workflow and status are required"))
		}

		// Ensure the name matches the URL parameter
		workflow.Name = name

		// Update only the status using the status subresource
		err = s.TestWorkflowsClient.UpdateStatus(ctx, environmentId, *workflow)
		if err != nil {
			s.Metrics.IncUpdateTestWorkflow(err)
			return s.ClientError(c, errPrefix, err)
		}

		s.Metrics.IncUpdateTestWorkflow(err)

		// Return the updated workflow
		updatedWorkflow, err := s.TestWorkflowsClient.Get(ctx, environmentId, name)
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}

		return c.JSON(updatedWorkflow)
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
			err = crdcommon.DeserializeCRD(obj, c.Body())
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
			target := &cloud.ExecutionTarget{Replicate: request.Target.Replicate}
			if request.Target.Match != nil {
				target.Match = make(map[string]*cloud.ExecutionTargetLabels)
				for k, v := range request.Target.Match {
					target.Match[k] = &cloud.ExecutionTargetLabels{Labels: v}
				}
			}
			if request.Target.Not != nil {
				target.Not = make(map[string]*cloud.ExecutionTargetLabels)
				for k, v := range request.Target.Not {
					target.Not[k] = &cloud.ExecutionTargetLabels{Labels: v}
				}
			}
			scheduleExecution.Targets = []*cloud.ExecutionTarget{target}
		}
		if request.Runtime != nil && len(request.Runtime.Variables) > 0 {
			scheduleExecution.Runtime = &cloud.TestWorkflowRuntime{
				EnvVars: request.Runtime.Variables,
			}
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

		resp := s.testWorkflowExecutor.Execute(ctx, common.StandaloneEnvironment, &cloud.ScheduleRequest{
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

// TODO: Add metrics
func (s *TestkubeAPI) ReRunTestWorkflowExecutionHandler() fiber.Handler {
	return func(c *fiber.Ctx) (err error) {
		ctx := c.Context()
		executionID := c.Params("executionID")
		s.Log.Debugw("rerunning test workflow execution", "id", executionID)

		errPrefix := "failed to rerun test workflow execution"

		// Load the running comtext
		var twrContext testkube.TestWorkflowRunningContext
		err = c.BodyParser(&twrContext)
		if err != nil && !errors.Is(err, fiber.ErrUnprocessableEntity) {
			return s.BadRequest(c, errPrefix, "invalid body", err)
		}

		execution, err := s.TestWorkflowResults.Get(ctx, executionID)
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}

		if execution.ResolvedWorkflow == nil {
			return s.ClientError(c, errPrefix, errors.New("can't find resolved workflow spec"))
		}

		name := execution.ResolvedWorkflow.Name
		workflow, err := json.Marshal(execution.ResolvedWorkflow)
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}

		// Load the execution request
		request := testkube.TestWorkflowExecutionRequest{
			RunningContext:  &twrContext,
			Tags:            execution.Tags,
			DisableWebhooks: execution.DisableWebhooks,
			Target:          execution.RunnerTarget,
			Runtime:         execution.Runtime,
		}

		request.Config = make(map[string]string)
		for key, value := range execution.ConfigParams {
			if value.Sensitive {
				return s.ClientError(c, errPrefix, errors.New("can't rerun test workflow execution with sensitive prameters"))
			}

			if value.Truncated {
				return s.ClientError(c, errPrefix, errors.New("can't rerun test workflow execution with truncated parameters"))
			}

			if !value.EmptyValue {
				request.Config[key] = value.Value
			}
		}

		runningContext, user := testworkflowexecutor.GetNewRunningContext(request.RunningContext, nil)

		var scheduleExecution cloud.ScheduleExecution
		if request.Target != nil {
			target := &cloud.ExecutionTarget{Replicate: request.Target.Replicate}
			if request.Target.Match != nil {
				target.Match = make(map[string]*cloud.ExecutionTargetLabels)
				for k, v := range request.Target.Match {
					target.Match[k] = &cloud.ExecutionTargetLabels{Labels: v}
				}
			}

			if request.Target.Not != nil {
				target.Not = make(map[string]*cloud.ExecutionTargetLabels)
				for k, v := range request.Target.Not {
					target.Not[k] = &cloud.ExecutionTargetLabels{Labels: v}
				}
			}

			scheduleExecution.Targets = []*cloud.ExecutionTarget{target}
		}

		scheduleExecution.Selector = &cloud.ScheduleResourceSelector{Name: name}
		scheduleExecution.Config = request.Config
		if request.Runtime != nil && len(request.Runtime.Variables) > 0 {
			scheduleExecution.Runtime = &cloud.TestWorkflowRuntime{
				EnvVars: request.Runtime.Variables,
			}
		}
		resp := s.testWorkflowExecutor.Execute(ctx, common.StandaloneEnvironment, &cloud.ScheduleRequest{
			Executions:         []*cloud.ScheduleExecution{&scheduleExecution},
			DisableWebhooks:    request.DisableWebhooks,
			Tags:               request.Tags,
			RunningContext:     runningContext,
			User:               user,
			ExecutionReference: &executionID,
			ResolvedWorkflow:   workflow,
		})

		results := make([]testkube.TestWorkflowExecution, 0)
		for v := range resp.Channel() {
			results = append(results, *v)
		}

		if resp.Error() != nil {
			return s.InternalError(c, errPrefix, "execution error", resp.Error())
		}

		s.Log.Debugw("rerunning test workflow execution", "id", executionID)
		if len(results) != 0 {
			return c.JSON(results[0])
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
