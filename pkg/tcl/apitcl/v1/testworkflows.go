// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package v1

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/pkg/errors"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	mappers2 "github.com/kubeshop/testkube/pkg/tcl/mapperstcl/testworkflows"
)

func (s *apiTCL) ListTestWorkflowsHandler() fiber.Handler {
	errPrefix := "failed to list test workflows"
	return func(c *fiber.Ctx) (err error) {
		workflows, err := s.getFilteredTestWorkflowList(c)
		if err != nil {
			return s.BadGateway(c, errPrefix, "client problem", err)
		}
		err = SendResourceList(c, "TestWorkflow", testworkflowsv1.GroupVersion, mappers2.MapTestWorkflowKubeToAPI, workflows.Items...)
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
		err = SendResource(c, "TestWorkflow", testworkflowsv1.GroupVersion, mappers2.MapKubeToAPI, workflow)
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
		//skipExecutions := c.Query("skipDeleteExecutions", "")
		//if skipExecutions != "true" {
		//	// TODO: Delete Executions
		//}
		return c.SendStatus(http.StatusNoContent)
	}
}

func (s *apiTCL) DeleteTestWorkflowsHandler() fiber.Handler {
	errPrefix := "failed to delete test workflows"
	return func(c *fiber.Ctx) error {
		selector := c.Query("selector")
		_, err := s.TestWorkflowsClient.List(selector)
		if err != nil {
			return s.BadGateway(c, errPrefix, "client problem", err)
		}

		err = s.TestWorkflowsClient.DeleteByLabels(selector)
		if err != nil {
			return s.ClientError(c, errPrefix, err)
		}

		//skipExecutions := c.Query("skipDeleteExecutions", "")
		//for range workflows.Items {
		//	s.Metrics.IncDeleteTestWorkflow(err)
		//	if skipExecutions != "true" {
		//		// TODO: Delete Executions
		//	}
		//}
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
			obj = mappers2.MapAPIToKube(v)
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

		err = SendResource(c, "TestWorkflow", testworkflowsv1.GroupVersion, mappers2.MapKubeToAPI, obj)
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
			obj = mappers2.MapAPIToKube(v)
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

		err = SendResource(c, "TestWorkflow", testworkflowsv1.GroupVersion, mappers2.MapKubeToAPI, obj)
		if err != nil {
			return s.InternalError(c, errPrefix, "serialization problem", err)
		}
		return
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
