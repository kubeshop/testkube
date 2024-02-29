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
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	kubeclient "sigs.k8s.io/controller-runtime/pkg/client"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/pkg/client/testworkflows/v1"
	apiv1 "github.com/kubeshop/testkube/internal/app/api/v1"
	"github.com/kubeshop/testkube/internal/config"
)

type apiTCL struct {
	apiv1.TestkubeAPI
	ProContext                  *config.ProContext
	TestWorkflowsClient         testworkflowsv1.Interface
	TestWorkflowTemplatesClient testworkflowsv1.TestWorkflowTemplatesInterface
}

type ApiTCL interface {
	AppendRoutes()
}

func NewApiTCL(
	testkubeAPI apiv1.TestkubeAPI,
	proContext *config.ProContext,
	kubeClient kubeclient.Client,
) ApiTCL {
	return &apiTCL{
		TestkubeAPI:                 testkubeAPI,
		ProContext:                  proContext,
		TestWorkflowsClient:         testworkflowsv1.NewClient(kubeClient, testkubeAPI.Namespace),
		TestWorkflowTemplatesClient: testworkflowsv1.NewTestWorkflowTemplatesClient(kubeClient, testkubeAPI.Namespace),
	}
}

func (s *apiTCL) NotImplemented(c *fiber.Ctx) error {
	return s.Error(c, http.StatusNotImplemented, errors.New("not implemented yet"))
}

func (s *apiTCL) BadGateway(c *fiber.Ctx, prefix, description string, err error) error {
	return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: %s: %w", prefix, description, err))
}

func (s *apiTCL) InternalError(c *fiber.Ctx, prefix, description string, err error) error {
	return s.Error(c, http.StatusInternalServerError, fmt.Errorf("%s: %s: %w", prefix, description, err))
}

func (s *apiTCL) BadRequest(c *fiber.Ctx, prefix, description string, err error) error {
	return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: %s: %w", prefix, description, err))
}

func (s *apiTCL) NotFound(c *fiber.Ctx, prefix, description string, err error) error {
	return s.Error(c, http.StatusNotFound, fmt.Errorf("%s: %s: %w", prefix, description, err))
}

func (s *apiTCL) ClientError(c *fiber.Ctx, prefix string, err error) error {
	if IsNotFound(err) {
		return s.NotFound(c, prefix, "client not found", err)
	}
	return s.BadGateway(c, prefix, "client problem", err)
}

func (s *apiTCL) AppendRoutes() {
	root := s.Routes

	testWorkflows := root.Group("/test-workflows")
	testWorkflows.Get("/", s.pro(s.ListTestWorkflowsHandler()))
	testWorkflows.Post("/", s.pro(s.CreateTestWorkflowHandler()))
	testWorkflows.Delete("/", s.pro(s.DeleteTestWorkflowsHandler()))
	testWorkflows.Get("/:id", s.pro(s.GetTestWorkflowHandler()))
	testWorkflows.Put("/:id", s.pro(s.UpdateTestWorkflowHandler()))
	testWorkflows.Delete("/:id", s.pro(s.DeleteTestWorkflowHandler()))

	root.Post("/preview-test-workflow", s.pro(s.PreviewTestWorkflowHandler()))

	testWorkflowTemplates := root.Group("/test-workflow-templates")
	testWorkflowTemplates.Get("/", s.pro(s.ListTestWorkflowTemplatesHandler()))
	testWorkflowTemplates.Post("/", s.pro(s.CreateTestWorkflowTemplateHandler()))
	testWorkflowTemplates.Delete("/", s.pro(s.DeleteTestWorkflowTemplatesHandler()))
	testWorkflowTemplates.Get("/:id", s.pro(s.GetTestWorkflowTemplateHandler()))
	testWorkflowTemplates.Put("/:id", s.pro(s.UpdateTestWorkflowTemplateHandler()))
	testWorkflowTemplates.Delete("/:id", s.pro(s.DeleteTestWorkflowTemplateHandler()))
}