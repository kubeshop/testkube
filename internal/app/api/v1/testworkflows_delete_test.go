package v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	testworkflowrepository "github.com/kubeshop/testkube/pkg/repository/testworkflow"
)

func TestDeleteTestWorkflowsHandlerDeletesExplicitNamesOnly(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := testworkflowclient.NewMockTestWorkflowClient(ctrl)
	outputRepository := testworkflowrepository.NewMockOutputRepository(ctrl)
	resultsRepository := testworkflowrepository.NewMockRepository(ctrl)

	gomock.InOrder(
		client.EXPECT().
			Get(gomock.Any(), "test-env", "workflow-a").
			Return(&testkube.TestWorkflow{Name: "workflow-a"}, nil),
		client.EXPECT().
			Get(gomock.Any(), "test-env", "workflow-b").
			Return(&testkube.TestWorkflow{Name: "workflow-b"}, nil),
		client.EXPECT().
			Delete(gomock.Any(), "test-env", "workflow-a").
			Return(nil),
		client.EXPECT().
			Delete(gomock.Any(), "test-env", "workflow-b").
			Return(nil),
	)
	outputRepository.EXPECT().
		DeleteOutputForTestWorkflows(gomock.Any(), []string{"workflow-a", "workflow-b"}).
		Return(nil)
	resultsRepository.EXPECT().
		DeleteByTestWorkflows(gomock.Any(), []string{"workflow-a", "workflow-b"}).
		Return(nil)

	testAPI := &TestkubeAPI{
		TestWorkflowsClient: client,
		TestWorkflowOutput:  outputRepository,
		TestWorkflowResults: resultsRepository,
		Metrics:             metrics.NewMetrics(),
		Log:                 log.DefaultLogger,
		proContext:          &config.ProContext{EnvID: "test-env"},
	}
	app := fiber.New()
	app.Delete("/test-workflows", testAPI.DeleteTestWorkflowsHandler())

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodDelete,
		"/test-workflows?testWorkflowNames=workflow-a,workflow-b",
		nil,
	)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestDeleteTestWorkflowsHandlerDeletesBySelector(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := testworkflowclient.NewMockTestWorkflowClient(ctrl)
	labels := map[string]string{"team": "backend"}

	client.EXPECT().
		List(gomock.Any(), "test-env", testworkflowclient.ListOptions{Labels: labels}).
		Return([]testkube.TestWorkflow{{Name: "workflow-a"}}, nil)
	client.EXPECT().
		DeleteByLabels(gomock.Any(), "test-env", labels).
		Return(uint32(1), nil)

	testAPI := &TestkubeAPI{
		TestWorkflowsClient: client,
		Metrics:             metrics.NewMetrics(),
		Log:                 log.DefaultLogger,
		proContext:          &config.ProContext{EnvID: "test-env"},
	}
	app := fiber.New()
	app.Delete("/test-workflows", testAPI.DeleteTestWorkflowsHandler())

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodDelete,
		"/test-workflows?selector=team%3Dbackend&skipDeleteExecutions=true",
		nil,
	)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}
