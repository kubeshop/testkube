// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package v1

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson/primitive"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
	testworkflowmappers "github.com/kubeshop/testkube/pkg/tcl/mapperstcl/testworkflows"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowresolver"
)

func (s *apiTCL) ListTestWorkflowsHandler() fiber.Handler {
	errPrefix := "failed to list test workflows"
	return func(c *fiber.Ctx) (err error) {
		workflows, err := s.getFilteredTestWorkflowList(c)
		if err != nil {
			return s.BadGateway(c, errPrefix, "client problem", err)
		}
		err = SendResourceList(c, "TestWorkflow", testworkflowsv1.GroupVersion, testworkflowmappers.MapTestWorkflowKubeToAPI, workflows.Items...)
		if err != nil {
			return s.InternalError(c, errPrefix, "serialization problem", err)
		}
		return
	}
}

func (s *apiTCL) GetTestWorkflowHandler() fiber.Handler {
	return func(c *fiber.Ctx) (err error) {
		name := c.Params("id")
		errPrefix := fmt.Sprintf("failed to get test workflow '%s'", name)
		workflow, err := s.TestWorkflowsClient.Get(name)
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}
		err = SendResource(c, "TestWorkflow", testworkflowsv1.GroupVersion, testworkflowmappers.MapKubeToAPI, workflow)
		if err != nil {
			return s.InternalError(c, errPrefix, "serialization problem", err)
		}
		return
	}
}

func (s *apiTCL) DeleteTestWorkflowHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("id")
		errPrefix := fmt.Sprintf("failed to delete test workflow '%s'", name)
		err := s.TestWorkflowsClient.Delete(name)
		s.Metrics.IncDeleteTestWorkflow(err)
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}
		skipExecutions := c.Query("skipDeleteExecutions", "")
		if skipExecutions != "true" {
			err = s.TestWorkflowResults.DeleteByTestWorkflow(context.Background(), name)
			if err != nil {
				return s.ClientError(c, "deleting executions", err)
			}
		}
		return c.SendStatus(http.StatusNoContent)
	}
}

func (s *apiTCL) DeleteTestWorkflowsHandler() fiber.Handler {
	errPrefix := "failed to delete test workflows"
	return func(c *fiber.Ctx) error {
		selector := c.Query("selector")
		workflows, err := s.TestWorkflowsClient.List(selector)
		if err != nil {
			return s.BadGateway(c, errPrefix, "client problem", err)
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
			err = s.TestWorkflowResults.DeleteByTestWorkflows(context.Background(), names)
			if err != nil {
				return s.ClientError(c, "deleting executions", err)
			}
		}

		return c.SendStatus(http.StatusNoContent)
	}
}

func (s *apiTCL) CreateTestWorkflowHandler() fiber.Handler {
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
			obj = testworkflowmappers.MapAPIToKube(v)
		}

		// Validate resource
		if obj == nil || obj.Name == "" {
			return s.BadRequest(c, errPrefix, "invalid body", errors.New("name is required"))
		}
		obj.Namespace = s.Namespace

		// Create the resource
		obj, err = s.TestWorkflowsClient.Create(obj)
		s.Metrics.IncCreateTestWorkflow(err)
		if err != nil {
			return s.BadRequest(c, errPrefix, "client error", err)
		}
		s.sendCreateWorkflowTelemetry(c.Context(), obj)

		err = SendResource(c, "TestWorkflow", testworkflowsv1.GroupVersion, testworkflowmappers.MapKubeToAPI, obj)
		if err != nil {
			return s.InternalError(c, errPrefix, "serialization problem", err)
		}
		return
	}
}

func (s *apiTCL) UpdateTestWorkflowHandler() fiber.Handler {
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
			obj = testworkflowmappers.MapAPIToKube(v)
		}

		// Read existing resource
		workflow, err := s.TestWorkflowsClient.Get(name)
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}

		// Validate resource
		if obj == nil {
			return s.BadRequest(c, errPrefix, "invalid body", errors.New("body is required"))
		}
		obj.Namespace = workflow.Namespace
		obj.Name = workflow.Name
		obj.ResourceVersion = workflow.ResourceVersion

		// Update the resource
		obj, err = s.TestWorkflowsClient.Update(obj)
		s.Metrics.IncUpdateTestWorkflow(err)
		if err != nil {
			return s.BadRequest(c, errPrefix, "client error", err)
		}

		err = SendResource(c, "TestWorkflow", testworkflowsv1.GroupVersion, testworkflowmappers.MapKubeToAPI, obj)
		if err != nil {
			return s.InternalError(c, errPrefix, "serialization problem", err)
		}
		return
	}
}

func (s *apiTCL) PreviewTestWorkflowHandler() fiber.Handler {
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
			obj = testworkflowmappers.MapAPIToKube(v)
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

		err = SendResource(c, "TestWorkflow", testworkflowsv1.GroupVersion, testworkflowmappers.MapKubeToAPI, obj)
		if err != nil {
			return s.InternalError(c, errPrefix, "serialization problem", err)
		}
		return
	}
}

// TODO: Add metrics
func (s *apiTCL) ExecuteTestWorkflowHandler() fiber.Handler {
	return func(c *fiber.Ctx) (err error) {
		ctx := c.Context()
		name := c.Params("id")
		errPrefix := fmt.Sprintf("failed to execute test workflow '%s'", name)
		workflow, err := s.TestWorkflowsClient.Get(name)
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}

		// Delete unnecessary data
		delete(workflow.Annotations, "kubectl.kubernetes.io/last-applied-configuration")

		// Preserve initial workflow
		initialWorkflow := workflow.DeepCopy()

		// Load the execution request
		var request testkube.TestWorkflowExecutionRequest
		err = c.BodyParser(&request)
		if err != nil && !errors.Is(err, fiber.ErrUnprocessableEntity) {
			return s.BadRequest(c, errPrefix, "invalid body", err)
		}

		// Fetch the templates
		tpls := testworkflowresolver.ListTemplates(workflow)
		tplsMap := make(map[string]testworkflowsv1.TestWorkflowTemplate, len(tpls))
		for tplName := range tpls {
			tpl, err := s.TestWorkflowTemplatesClient.Get(tplName)
			if err != nil {
				return s.BadRequest(c, errPrefix, "fetching error", err)
			}
			tplsMap[tplName] = *tpl
		}

		// Apply the configuration
		_, err = testworkflowresolver.ApplyWorkflowConfig(workflow, testworkflowmappers.MapConfigValueAPIToKube(request.Config))
		if err != nil {
			return s.BadRequest(c, errPrefix, "configuration", err)
		}

		// Resolve the TestWorkflow
		err = testworkflowresolver.ApplyTemplates(workflow, tplsMap)
		if err != nil {
			return s.BadRequest(c, errPrefix, "resolving error", err)
		}

		// Build the basic Execution data
		id := primitive.NewObjectID().Hex()
		now := time.Now()
		machine := expressionstcl.NewMachine().
			RegisterStringMap("internal", map[string]string{
				"storage.url":        os.Getenv("STORAGE_ENDPOINT"),
				"storage.accessKey":  os.Getenv("STORAGE_ACCESSKEYID"),
				"storage.secretKey":  os.Getenv("STORAGE_SECRETACCESSKEY"),
				"storage.region":     os.Getenv("STORAGE_REGION"),
				"storage.bucket":     os.Getenv("STORAGE_BUCKET"),
				"storage.token":      os.Getenv("STORAGE_TOKEN"),
				"storage.ssl":        common.GetOr(os.Getenv("STORAGE_SSL"), "false"),
				"storage.skipVerify": common.GetOr(os.Getenv("STORAGE_SKIP_VERIFY"), "false"),
				"storage.certFile":   os.Getenv("STORAGE_CERT_FILE"),
				"storage.keyFile":    os.Getenv("STORAGE_KEY_FILE"),
				"storage.caFile":     os.Getenv("STORAGE_CA_FILE"),

				"cloud.enabled":         strconv.FormatBool(os.Getenv("TESTKUBE_PRO_API_KEY") != "" || os.Getenv("TESTKUBE_CLOUD_API_KEY") != ""),
				"cloud.api.key":         common.GetOr(os.Getenv("TESTKUBE_PRO_API_KEY"), os.Getenv("TESTKUBE_CLOUD_API_KEY")),
				"cloud.api.tlsInsecure": common.GetOr(os.Getenv("TESTKUBE_PRO_TLS_INSECURE"), os.Getenv("TESTKUBE_CLOUD_TLS_INSECURE"), "false"),
				"cloud.api.skipVerify":  common.GetOr(os.Getenv("TESTKUBE_PRO_SKIP_VERIFY"), os.Getenv("TESTKUBE_CLOUD_SKIP_VERIFY"), "false"),
				"cloud.api.url":         common.GetOr(os.Getenv("TESTKUBE_PRO_URL"), os.Getenv("TESTKUBE_CLOUD_URL")),

				"dashboard.url": os.Getenv("TESTKUBE_DASHBOARD_URI"),
				"api.url":       s.ApiUrl,
				"namespace":     s.Namespace,
			}).
			RegisterStringMap("workflow", map[string]string{
				"name": workflow.Name,
			}).
			RegisterStringMap("execution", map[string]string{
				"id": id,
			})

		// Preserve resolved TestWorkflow
		resolvedWorkflow := workflow.DeepCopy()

		// Process the TestWorkflow
		bundle, err := testworkflowprocessor.NewFullFeatured(s.ImageInspector).
			Bundle(c.Context(), workflow, machine)
		if err != nil {
			return s.BadRequest(c, errPrefix, "processing error", err)
		}

		// Load execution identifier data
		// TODO: Consider if that should not be shared (as now it is between Tests and Test Suites)
		number, _ := s.ExecutionResults.GetNextExecutionNumber(context.Background(), workflow.Name)
		executionName := request.Name
		if executionName == "" {
			executionName = fmt.Sprintf("%s-%d", workflow.Name, number)
		}

		// Ensure it is unique name
		// TODO: Consider if we shouldn't make name unique across all TestWorkflows
		next, _ := s.TestWorkflowResults.GetByNameAndTestWorkflow(ctx, executionName, workflow.Name)
		if next.Name == executionName {
			return s.BadRequest(c, errPrefix, "execution name already exists", errors.New(executionName))
		}

		// Build Execution entity
		// TODO: Consider storing "config" as well
		execution := testkube.TestWorkflowExecution{
			Id:          id,
			Name:        executionName,
			Number:      number,
			ScheduledAt: now,
			StatusAt:    now,
			Signature:   testworkflowprocessor.MapSignatureListToInternal(bundle.Signature),
			Result: &testkube.TestWorkflowResult{
				Status:          common.Ptr(testkube.QUEUED_TestWorkflowStatus),
				PredictedStatus: common.Ptr(testkube.PASSED_TestWorkflowStatus),
				Initialization: &testkube.TestWorkflowStepResult{
					Status: common.Ptr(testkube.QUEUED_TestWorkflowStepStatus),
				},
				Steps: testworkflowprocessor.MapSignatureListToStepResults(bundle.Signature),
			},
			Output:           []testkube.TestWorkflowOutput{},
			Workflow:         testworkflowmappers.MapKubeToAPI(initialWorkflow),
			ResolvedWorkflow: testworkflowmappers.MapKubeToAPI(resolvedWorkflow),
		}
		err = s.TestWorkflowResults.Insert(ctx, execution)
		if err != nil {
			return s.InternalError(c, errPrefix, "inserting execution to storage", err)
		}

		// Schedule the execution
		s.TestWorkflowExecutor.Schedule(bundle, execution)
		s.sendRunWorkflowTelemetry(c.Context(), workflow)

		return c.JSON(execution)
	}
}

func (s *apiTCL) getFilteredTestWorkflowList(c *fiber.Ctx) (*testworkflowsv1.TestWorkflowList, error) {
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
