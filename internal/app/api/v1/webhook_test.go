package v1

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	executorsclientv1 "github.com/kubeshop/testkube/pkg/operator/client/executors/v1"
)

func TestGetWebhookHandler_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := executorsclientv1.NewMockWebhooksInterface(ctrl)
	testAPI := &TestkubeAPI{WebhooksClient: mockClient}

	app := fiber.New()
	app.Get("/webhooks/:name", testAPI.GetWebhookHandler())

	mockClient.EXPECT().
		Get("missing-webhook").
		Return(nil, status.Error(codes.NotFound, "webhook not found")).
		Times(1)

	req := httptest.NewRequest("GET", "/webhooks/missing-webhook", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestGetWebhookHandler_ClientError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := executorsclientv1.NewMockWebhooksInterface(ctrl)
	testAPI := &TestkubeAPI{WebhooksClient: mockClient}

	app := fiber.New()
	app.Get("/webhooks/:name", testAPI.GetWebhookHandler())

	mockClient.EXPECT().
		Get("broken-webhook").
		Return(nil, errors.New("client failed")).
		Times(1)

	req := httptest.NewRequest("GET", "/webhooks/broken-webhook", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
}

func TestDeleteWebhookHandler_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := executorsclientv1.NewMockWebhooksInterface(ctrl)
	testAPI := &TestkubeAPI{WebhooksClient: mockClient}

	app := fiber.New()
	app.Delete("/webhooks/:name", testAPI.DeleteWebhookHandler())

	mockClient.EXPECT().
		Delete("missing-webhook").
		Return(status.Error(codes.NotFound, "webhook not found")).
		Times(1)

	req := httptest.NewRequest("DELETE", "/webhooks/missing-webhook", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestDeleteWebhookHandler_ClientError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := executorsclientv1.NewMockWebhooksInterface(ctrl)
	testAPI := &TestkubeAPI{WebhooksClient: mockClient}

	app := fiber.New()
	app.Delete("/webhooks/:name", testAPI.DeleteWebhookHandler())

	mockClient.EXPECT().
		Delete("broken-webhook").
		Return(errors.New("client failed")).
		Times(1)

	req := httptest.NewRequest("DELETE", "/webhooks/broken-webhook", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
}
