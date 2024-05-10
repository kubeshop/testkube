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
	"errors"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	kubeclient "sigs.k8s.io/controller-runtime/pkg/client"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/pkg/client/testworkflows/v1"
	apiv1 "github.com/kubeshop/testkube/internal/app/api/v1"
	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	configRepo "github.com/kubeshop/testkube/pkg/repository/config"
	"github.com/kubeshop/testkube/pkg/tcl/repositorytcl/testworkflow"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowexecutor"
)

type apiTCL struct {
	apiv1.TestkubeAPI
	ProContext                  *config.ProContext
	TestWorkflowResults         testworkflow.Repository
	TestWorkflowOutput          testworkflow.OutputRepository
	TestWorkflowsClient         testworkflowsv1.Interface
	TestWorkflowTemplatesClient testworkflowsv1.TestWorkflowTemplatesInterface
	TestWorkflowExecutor        testworkflowexecutor.TestWorkflowExecutor
	configMap                   configRepo.Repository
}

type ApiTCL interface {
	AppendRoutes()
	GetTestWorkflowNotificationsStream(ctx context.Context, executionID string) (chan testkube.TestWorkflowExecutionNotification, error)
}

func NewApiTCL(
	testkubeAPI apiv1.TestkubeAPI,
	proContext *config.ProContext,
	kubeClient kubeclient.Client,
	testWorkflowResults testworkflow.Repository,
	testWorkflowOutput testworkflow.OutputRepository,
	configMap configRepo.Repository,
	testWorkflowExecutor testworkflowexecutor.TestWorkflowExecutor,
) ApiTCL {
	testWorkflowTemplatesClient := testworkflowsv1.NewTestWorkflowTemplatesClient(kubeClient, testkubeAPI.Namespace)
	go testWorkflowExecutor.Recover(context.Background())
	return &apiTCL{
		TestkubeAPI:                 testkubeAPI,
		ProContext:                  proContext,
		TestWorkflowResults:         testWorkflowResults,
		TestWorkflowOutput:          testWorkflowOutput,
		TestWorkflowsClient:         testworkflowsv1.NewClient(kubeClient, testkubeAPI.Namespace),
		TestWorkflowTemplatesClient: testWorkflowTemplatesClient,
		TestWorkflowExecutor:        testWorkflowExecutor,
		configMap:                   configMap,
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

	// Register TestWorkflows as additional source for labels
	s.WithLabelSources(s.TestWorkflowsClient, s.TestWorkflowTemplatesClient)

	testWorkflows := root.Group("/test-workflows")
	testWorkflows.Get("/", s.pro(s.ListTestWorkflowsHandler()))
	testWorkflows.Post("/", s.pro(s.CreateTestWorkflowHandler()))
	testWorkflows.Delete("/", s.pro(s.DeleteTestWorkflowsHandler()))
	testWorkflows.Get("/:id", s.pro(s.GetTestWorkflowHandler()))
	testWorkflows.Put("/:id", s.pro(s.UpdateTestWorkflowHandler()))
	testWorkflows.Delete("/:id", s.pro(s.DeleteTestWorkflowHandler()))
	testWorkflows.Get("/:id/executions", s.pro(s.ListTestWorkflowExecutionsHandler()))
	testWorkflows.Post("/:id/executions", s.pro(s.ExecuteTestWorkflowHandler()))
	testWorkflows.Get("/:id/metrics", s.pro(s.GetTestWorkflowMetricsHandler()))
	testWorkflows.Get("/:id/executions/:executionID", s.pro(s.GetTestWorkflowExecutionHandler()))
	testWorkflows.Post("/:id/abort", s.pro(s.AbortAllTestWorkflowExecutionsHandler()))
	testWorkflows.Post("/:id/executions/:executionID/abort", s.pro(s.AbortTestWorkflowExecutionHandler()))
	testWorkflows.Post("/:id/executions/:executionID/pause", s.pro(s.PauseTestWorkflowExecutionHandler()))
	testWorkflows.Post("/:id/executions/:executionID/resume", s.pro(s.ResumeTestWorkflowExecutionHandler()))
	testWorkflows.Get("/:id/executions/:executionID/logs", s.pro(s.GetTestWorkflowExecutionLogsHandler()))

	testWorkflowExecutions := root.Group("/test-workflow-executions")
	testWorkflowExecutions.Get("/", s.pro(s.ListTestWorkflowExecutionsHandler()))
	testWorkflowExecutions.Get("/:executionID", s.pro(s.GetTestWorkflowExecutionHandler()))
	testWorkflowExecutions.Get("/:executionID/notifications", s.pro(s.StreamTestWorkflowExecutionNotificationsHandler()))
	testWorkflowExecutions.Get("/:executionID/notifications/stream", s.pro(s.StreamTestWorkflowExecutionNotificationsWebSocketHandler()))
	testWorkflowExecutions.Post("/:executionID/abort", s.pro(s.AbortTestWorkflowExecutionHandler()))
	testWorkflowExecutions.Post("/:executionID/pause", s.pro(s.PauseTestWorkflowExecutionHandler()))
	testWorkflowExecutions.Post("/:executionID/resume", s.pro(s.ResumeTestWorkflowExecutionHandler()))
	testWorkflowExecutions.Get("/:executionID/logs", s.pro(s.GetTestWorkflowExecutionLogsHandler()))
	testWorkflowExecutions.Get("/:executionID/artifacts", s.pro(s.ListTestWorkflowExecutionArtifactsHandler()))
	testWorkflowExecutions.Get("/:executionID/artifacts/:filename", s.pro(s.GetTestWorkflowArtifactHandler()))
	testWorkflowExecutions.Get("/:executionID/artifact-archive", s.pro(s.GetTestWorkflowArtifactArchiveHandler()))

	testWorkflowWithExecutions := root.Group("/test-workflow-with-executions")
	testWorkflowWithExecutions.Get("/", s.pro(s.ListTestWorkflowWithExecutionsHandler()))
	testWorkflowWithExecutions.Get("/:id", s.pro(s.GetTestWorkflowWithExecutionHandler()))

	root.Post("/preview-test-workflow", s.pro(s.PreviewTestWorkflowHandler()))

	testWorkflowTemplates := root.Group("/test-workflow-templates")
	testWorkflowTemplates.Get("/", s.pro(s.ListTestWorkflowTemplatesHandler()))
	testWorkflowTemplates.Post("/", s.pro(s.CreateTestWorkflowTemplateHandler()))
	testWorkflowTemplates.Delete("/", s.pro(s.DeleteTestWorkflowTemplatesHandler()))
	testWorkflowTemplates.Get("/:id", s.pro(s.GetTestWorkflowTemplateHandler()))
	testWorkflowTemplates.Put("/:id", s.pro(s.UpdateTestWorkflowTemplateHandler()))
	testWorkflowTemplates.Delete("/:id", s.pro(s.DeleteTestWorkflowTemplateHandler()))
}
