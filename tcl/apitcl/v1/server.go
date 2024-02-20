// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package v1

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v2"

	apiv1 "github.com/kubeshop/testkube/internal/app/api/v1"
)

type apiTCL struct {
	apiv1.TestkubeAPI
}

type ApiTCL interface {
	AppendRoutes()
}

func NewApiTCL(testkubeAPI apiv1.TestkubeAPI) ApiTCL {
	return &apiTCL{
		TestkubeAPI: testkubeAPI,
	}
}

func (s *apiTCL) NotImplemented(c *fiber.Ctx) error {
	return s.Error(c, http.StatusNotImplemented, errors.New("not implemented yet"))
}

func (s *apiTCL) AppendRoutes() {
	root := s.Routes

	testWorkflows := root.Group("/testworkflows")
	testWorkflows.Get("/", s.pro(s.ListTestWorkflowsHandler()))
	testWorkflows.Get("/:id", s.pro(s.GetTestWorkflowHandler()))
	testWorkflows.Post("/:id", s.pro(s.CreateTestWorkflowHandler()))
	testWorkflows.Patch("/:id", s.pro(s.PatchTestWorkflowHandler()))
	testWorkflows.Delete("/:id", s.pro(s.DeleteTestWorkflowHandler()))

	testWorkflowTemplates := root.Group("/testworkflowtemplates")
	testWorkflowTemplates.Get("/", s.pro(s.ListTestWorkflowTemplatesHandler()))
	testWorkflowTemplates.Get("/:id", s.pro(s.GetTestWorkflowTemplateHandler()))
	testWorkflowTemplates.Post("/:id", s.pro(s.CreateTestWorkflowTemplateHandler()))
	testWorkflowTemplates.Patch("/:id", s.pro(s.PatchTestWorkflowTemplateHandler()))
	testWorkflowTemplates.Delete("/:id", s.pro(s.DeleteTestWorkflowTemplateHandler()))
}
