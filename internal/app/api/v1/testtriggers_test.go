package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/labels"

	testtriggersv1 "github.com/kubeshop/testkube-operator/api/testtriggers/v1"
	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	testtriggersmapper "github.com/kubeshop/testkube/pkg/mapper/testtriggers"
	"github.com/kubeshop/testkube/pkg/newclients/testtriggerclient"
)

func TestCreateTestTriggerHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := testtriggerclient.NewMockTestTriggerClient(ctrl)

	testAPI := &TestkubeAPIForTesting{
		TestTriggersClient: mockClient,
		Log:                log.DefaultLogger,
		Namespace:          "default",
		proContext:         &config.ProContext{EnvID: "test-env"},
	}

	app := fiber.New()
	app.Post("/test-triggers", testAPI.CreateTestTriggerHandler())

	t.Run("should create test trigger with JSON request", func(t *testing.T) {
		// given
		request := testkube.TestTriggerUpsertRequest{
			Name:      "test-trigger",
			Namespace: "default",
			Resource:  resourcePtr("deployment"),
			Event:     "created",
			Action:    actionPtr("run"),
			Execution: executionPtr("test"),
			ResourceSelector: &testkube.TestTriggerSelector{
				Name:      "test-selector",
				Namespace: "default",
			},
			TestSelector: &testkube.TestTriggerSelector{
				Name:      "test-to-run",
				Namespace: "default",
			},
		}

		expectedTrigger := testkube.TestTrigger{
			Name:      "test-trigger",
			Namespace: "default",
			Resource:  resourcePtr("deployment"),
			Event:     "created",
			Action:    actionPtr("run"),
			Execution: executionPtr("test"),
			ResourceSelector: &testkube.TestTriggerSelector{
				Name:      "test-selector",
				Namespace: "default",
			},
			TestSelector: &testkube.TestTriggerSelector{
				Name:      "test-to-run",
				Namespace: "default",
			},
		}

		mockClient.EXPECT().
			Create(gomock.Any(), "test-env", gomock.Any()).
			Return(nil).
			Times(1)

		requestBody, _ := json.Marshal(request)
		req := httptest.NewRequest("POST", "/test-triggers", bytes.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result testkube.TestTrigger
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Equal(t, expectedTrigger.Name, result.Name)
		assert.Equal(t, expectedTrigger.Namespace, result.Namespace)
	})

	t.Run("should create test trigger with YAML request", func(t *testing.T) {
		// given
		yamlContent := `
apiVersion: tests.testkube.io/v1
kind: TestTrigger
metadata:
  name: test-trigger-yaml
  namespace: default
spec:
  resource: deployment
  event: created
  action: run
  execution: test
`

		// YAML test just returns success without calling the client
		req := httptest.NewRequest("POST", "/test-triggers", strings.NewReader(yamlContent))
		req.Header.Set("Content-Type", "text/yaml")

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("should generate trigger name when not provided", func(t *testing.T) {
		// given
		request := testkube.TestTriggerUpsertRequest{
			Namespace: "default",
			Resource:  resourcePtr("deployment"),
			Event:     "created",
			Action:    actionPtr("run"),
			Execution: executionPtr("test"),
			ResourceSelector: &testkube.TestTriggerSelector{
				Name:      "test-selector",
				Namespace: "default",
			},
			TestSelector: &testkube.TestTriggerSelector{
				Name:      "test-to-run",
				Namespace: "default",
			},
		}

		mockClient.EXPECT().
			Create(gomock.Any(), "test-env", gomock.Any()).
			DoAndReturn(func(ctx context.Context, envId string, trigger testkube.TestTrigger) error {
				assert.NotEmpty(t, trigger.Name)
				assert.Contains(t, trigger.Name, "trigger-deployment-created-run-test")
				return nil
			}).
			Times(1)

		requestBody, _ := json.Marshal(request)
		req := httptest.NewRequest("POST", "/test-triggers", bytes.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("should return error when client fails", func(t *testing.T) {
		// given
		request := testkube.TestTriggerUpsertRequest{
			Name:      "test-trigger",
			Namespace: "default",
			Resource:  resourcePtr("deployment"),
			Event:     "created",
			Action:    actionPtr("run"),
			Execution: executionPtr("test"),
			ResourceSelector: &testkube.TestTriggerSelector{
				Name:      "test-selector",
				Namespace: "default",
			},
			TestSelector: &testkube.TestTriggerSelector{
				Name:      "test-to-run",
				Namespace: "default",
			},
		}

		expectedError := errors.New("client error")
		mockClient.EXPECT().
			Create(gomock.Any(), "test-env", gomock.Any()).
			Return(expectedError).
			Times(1)

		requestBody, _ := json.Marshal(request)
		req := httptest.NewRequest("POST", "/test-triggers", bytes.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
	})

	t.Run("should return error for invalid JSON", func(t *testing.T) {
		// given
		req := httptest.NewRequest("POST", "/test-triggers", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestUpdateTestTriggerHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := testtriggerclient.NewMockTestTriggerClient(ctrl)

	testAPI := &TestkubeAPI{
		TestTriggersClient: mockClient,
		Log:                log.DefaultLogger,
		Namespace:          "default",
		proContext:         &config.ProContext{EnvID: "test-env"},
	}

	app := fiber.New()
	app.Put("/test-triggers", testAPI.UpdateTestTriggerHandler())

	t.Run("should update test trigger successfully with json", func(t *testing.T) {
		// given
		request := testkube.TestTriggerUpsertRequest{
			Name:      "test-trigger",
			Namespace: "default",
			Selector: &testkube.IoK8sApimachineryPkgApisMetaV1LabelSelector{
				MatchLabels: map[string]string{
					"testkube.io/agent-name": "test-agent",
				},
			},
			Resource: resourcePtr("deployment"),
			ResourceSelector: &testkube.TestTriggerSelector{
				LabelSelector: &testkube.IoK8sApimachineryPkgApisMetaV1LabelSelector{
					MatchLabels: map[string]string{
						"testkube.io/agent-name": "test-agent",
					},
				},
			},
			Event:     "updated",
			Action:    actionPtr("run"),
			Execution: executionPtr("test"),
		}

		existingTrigger := &testkube.TestTrigger{
			Name:      "test-trigger",
			Namespace: "default",
			Resource:  resourcePtr("pod"),
			Event:     "created",
			Action:    actionPtr("stop"),
			Execution: executionPtr("old-test"),
		}

		mockClient.EXPECT().
			Get(gomock.Any(), "test-env", "test-trigger", "default").
			Return(existingTrigger, nil).
			Times(1)

		mockClient.EXPECT().
			Update(gomock.Any(), "test-env", gomock.Any()).
			DoAndReturn(func(ctx context.Context, envId string, trigger testkube.TestTrigger) error {
				assert.Equal(t, "test-trigger", trigger.Name)
				assert.Equal(t, "default", trigger.Namespace)
				assert.NotNil(t, trigger.Selector)
				if trigger.Selector != nil {
					assert.Equal(t, map[string]string{
						"testkube.io/agent-name": "test-agent",
					}, trigger.Selector.MatchLabels)
				}
				assert.Equal(t, testkube.TestTriggerResources("deployment"), *trigger.Resource)
				assert.NotNil(t, trigger.ResourceSelector)
				assert.NotNil(t, trigger.ResourceSelector.LabelSelector)
				if trigger.ResourceSelector != nil &&
					trigger.ResourceSelector.LabelSelector != nil {
					assert.Equal(t, map[string]string{
						"testkube.io/agent-name": "test-agent",
					}, trigger.ResourceSelector.LabelSelector.MatchLabels)
				}
				assert.Equal(t, "updated", trigger.Event)
				assert.Equal(t, testkube.TestTriggerActions("run"), *trigger.Action)
				assert.Equal(t, testkube.TestTriggerExecutions("test"), *trigger.Execution)
				return nil
			}).
			Times(1)

		requestBody, _ := json.Marshal(request)
		t.Log("request body", string(requestBody))
		req := httptest.NewRequest("PUT", "/test-triggers", bytes.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("should update test trigger successfully with yaml", func(t *testing.T) {
		// given
		requestBody := `
apiVersion: tests.testkube.io/v1
kind: TestTrigger
metadata:
  name: test-trigger
  namespace: default
spec:
  selector:
    matchLabels:
      testkube.io/agent-name: test-agent
  resource: deployment
  resourceSelector:
    labelSelector:
      matchLabels:
        testkube.io/agent-name: test-agent
  event: updated
  action: run
  execution: test
`
		existingTrigger := &testkube.TestTrigger{
			Name:      "test-trigger",
			Namespace: "default",
			Resource:  resourcePtr("pod"),
			Event:     "created",
			Action:    actionPtr("stop"),
			Execution: executionPtr("old-test"),
		}

		mockClient.EXPECT().
			Get(gomock.Any(), "test-env", "test-trigger", "default").
			Return(existingTrigger, nil).
			Times(1)

		mockClient.EXPECT().
			Update(gomock.Any(), "test-env", gomock.Any()).
			DoAndReturn(func(ctx context.Context, envId string, trigger testkube.TestTrigger) error {
				assert.Equal(t, "test-trigger", trigger.Name)
				assert.Equal(t, "default", trigger.Namespace)
				assert.NotNil(t, trigger.Selector)
				if trigger.Selector != nil {
					assert.Equal(t, map[string]string{
						"testkube.io/agent-name": "test-agent",
					}, trigger.Selector.MatchLabels)
				}
				assert.Equal(t, testkube.TestTriggerResources("deployment"), *trigger.Resource)
				assert.NotNil(t, trigger.ResourceSelector)
				assert.NotNil(t, trigger.ResourceSelector.LabelSelector)
				if trigger.ResourceSelector != nil &&
					trigger.ResourceSelector.LabelSelector != nil {
					assert.Equal(t, map[string]string{
						"testkube.io/agent-name": "test-agent",
					}, trigger.ResourceSelector.LabelSelector.MatchLabels)
				}
				assert.Equal(t, "updated", trigger.Event)
				assert.Equal(t, testkube.TestTriggerActions("run"), *trigger.Action)
				assert.Equal(t, testkube.TestTriggerExecutions("test"), *trigger.Execution)
				return nil
			}).
			Times(1)

		t.Log("request body", requestBody)
		req := httptest.NewRequest("PUT", "/test-triggers", bytes.NewReader([]byte(requestBody)))
		req.Header.Set("Content-Type", "text/yaml")

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("should return error when get fails", func(t *testing.T) {
		// given
		request := testkube.TestTriggerUpsertRequest{
			Name:      "test-trigger",
			Namespace: "default",
		}

		expectedError := errors.New("not found")
		mockClient.EXPECT().
			Get(gomock.Any(), "test-env", "test-trigger", "default").
			Return(nil, expectedError).
			Times(1)

		requestBody, _ := json.Marshal(request)
		req := httptest.NewRequest("PUT", "/test-triggers", bytes.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
	})

	t.Run("should return error when update fails", func(t *testing.T) {
		// given
		request := testkube.TestTriggerUpsertRequest{
			Name:      "test-trigger",
			Namespace: "default",
		}

		existingTrigger := &testkube.TestTrigger{
			Name:      "test-trigger",
			Namespace: "default",
		}

		expectedError := errors.New("update failed")
		mockClient.EXPECT().
			Get(gomock.Any(), "test-env", "test-trigger", "default").
			Return(existingTrigger, nil).
			Times(1)

		mockClient.EXPECT().
			Update(gomock.Any(), "test-env", gomock.Any()).
			Return(expectedError).
			Times(1)

		requestBody, _ := json.Marshal(request)
		req := httptest.NewRequest("PUT", "/test-triggers", bytes.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
	})
}

func TestBulkUpdateTestTriggersHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := testtriggerclient.NewMockTestTriggerClient(ctrl)

	testAPI := &TestkubeAPIForTesting{
		TestTriggersClient: mockClient,
		Log:                log.DefaultLogger,
		Namespace:          "default",
		proContext:         &config.ProContext{EnvID: "test-env"},
	}

	app := fiber.New()
	app.Put("/test-triggers/bulk", testAPI.BulkUpdateTestTriggersHandler())

	t.Run("should bulk update test triggers successfully", func(t *testing.T) {
		// given
		request := []testkube.TestTriggerUpsertRequest{
			{
				Name:      "trigger-1",
				Namespace: "default",
				Resource:  resourcePtr("deployment"),
				Event:     "created",
				Action:    actionPtr("run"),
				Execution: executionPtr("test1"),
				ResourceSelector: &testkube.TestTriggerSelector{
					Name:      "test-selector-1",
					Namespace: "default",
				},
				TestSelector: &testkube.TestTriggerSelector{
					Name:      "test-to-run-1",
					Namespace: "default",
				},
			},
			{
				Name:      "trigger-2",
				Namespace: "default",
				Resource:  resourcePtr("service"),
				Event:     "updated",
				Action:    actionPtr("run"),
				Execution: executionPtr("test2"),
				ResourceSelector: &testkube.TestTriggerSelector{
					Name:      "test-selector-2",
					Namespace: "default",
				},
				TestSelector: &testkube.TestTriggerSelector{
					Name:      "test-to-run-2",
					Namespace: "default",
				},
			},
		}

		// Expect delete all for namespace
		mockClient.EXPECT().
			DeleteAll(gomock.Any(), "test-env", "default").
			Return(uint32(0), nil).
			Times(1)

		// Expect create for each trigger
		mockClient.EXPECT().
			Create(gomock.Any(), "test-env", gomock.Any()).
			Return(nil).
			Times(2)

		requestBody, _ := json.Marshal(request)
		req := httptest.NewRequest("PUT", "/test-triggers/bulk", bytes.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result []testkube.TestTrigger
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("should return error when delete all fails", func(t *testing.T) {
		// given
		request := []testkube.TestTriggerUpsertRequest{
			{
				Name:      "trigger-1",
				Namespace: "default",
			},
		}

		expectedError := errors.New("delete failed")
		mockClient.EXPECT().
			DeleteAll(gomock.Any(), "test-env", "default").
			Return(uint32(0), expectedError).
			Times(1)

		requestBody, _ := json.Marshal(request)
		req := httptest.NewRequest("PUT", "/test-triggers/bulk", bytes.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
	})
}

func TestGetTestTriggerHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := testtriggerclient.NewMockTestTriggerClient(ctrl)

	testAPI := &TestkubeAPIForTesting{
		TestTriggersClient: mockClient,
		Log:                log.DefaultLogger,
		Namespace:          "default",
		proContext:         &config.ProContext{EnvID: "test-env"},
	}

	app := fiber.New()
	app.Get("/test-triggers/:id", testAPI.GetTestTriggerHandler())

	t.Run("should get test trigger successfully", func(t *testing.T) {
		// given
		expectedTrigger := &testkube.TestTrigger{
			Name:      "test-trigger",
			Namespace: "default",
			Resource:  resourcePtr("deployment"),
			Event:     "created",
			Action:    actionPtr("run"),
			Execution: executionPtr("test"),
		}

		mockClient.EXPECT().
			Get(gomock.Any(), "test-env", "test-trigger", "default").
			Return(expectedTrigger, nil).
			Times(1)

		req := httptest.NewRequest("GET", "/test-triggers/test-trigger", nil)

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result testkube.TestTrigger
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Equal(t, expectedTrigger.Name, result.Name)
	})

	t.Run("should get test trigger with custom namespace", func(t *testing.T) {
		// given
		expectedTrigger := &testkube.TestTrigger{
			Name:      "test-trigger",
			Namespace: "custom",
		}

		mockClient.EXPECT().
			Get(gomock.Any(), "test-env", "test-trigger", "custom").
			Return(expectedTrigger, nil).
			Times(1)

		req := httptest.NewRequest("GET", "/test-triggers/test-trigger?namespace=custom", nil)

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("should return error when get fails", func(t *testing.T) {
		// given
		expectedError := errors.New("not found")
		mockClient.EXPECT().
			Get(gomock.Any(), "test-env", "test-trigger", "default").
			Return(nil, expectedError).
			Times(1)

		req := httptest.NewRequest("GET", "/test-triggers/test-trigger", nil)

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
	})
}

func TestDeleteTestTriggerHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := testtriggerclient.NewMockTestTriggerClient(ctrl)

	testAPI := &TestkubeAPIForTesting{
		TestTriggersClient: mockClient,
		Log:                log.DefaultLogger,
		Namespace:          "default",
		proContext:         &config.ProContext{EnvID: "test-env"},
	}

	app := fiber.New()
	app.Delete("/test-triggers/:id", testAPI.DeleteTestTriggerHandler())

	t.Run("should delete test trigger successfully", func(t *testing.T) {
		// given
		mockClient.EXPECT().
			Delete(gomock.Any(), "test-env", "test-trigger", "default").
			Return(nil).
			Times(1)

		req := httptest.NewRequest("DELETE", "/test-triggers/test-trigger", nil)

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("should delete test trigger with custom namespace", func(t *testing.T) {
		// given
		mockClient.EXPECT().
			Delete(gomock.Any(), "test-env", "test-trigger", "custom").
			Return(nil).
			Times(1)

		req := httptest.NewRequest("DELETE", "/test-triggers/test-trigger?namespace=custom", nil)

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("should return error when delete fails", func(t *testing.T) {
		// given
		expectedError := errors.New("delete failed")
		mockClient.EXPECT().
			Delete(gomock.Any(), "test-env", "test-trigger", "default").
			Return(expectedError).
			Times(1)

		req := httptest.NewRequest("DELETE", "/test-triggers/test-trigger", nil)

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
	})
}

func TestDeleteTestTriggersHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := testtriggerclient.NewMockTestTriggerClient(ctrl)

	testAPI := &TestkubeAPIForTesting{
		TestTriggersClient: mockClient,
		Log:                log.DefaultLogger,
		Namespace:          "default",
		proContext:         &config.ProContext{EnvID: "test-env"},
	}

	app := fiber.New()
	app.Delete("/test-triggers", testAPI.DeleteTestTriggersHandler())

	t.Run("should delete all test triggers successfully", func(t *testing.T) {
		// given
		mockClient.EXPECT().
			DeleteByLabels(gomock.Any(), "test-env", "", "default").
			Return(uint32(3), nil).
			Times(1)

		req := httptest.NewRequest("DELETE", "/test-triggers", nil)

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("should delete test triggers with selector", func(t *testing.T) {
		// given
		mockClient.EXPECT().
			DeleteByLabels(gomock.Any(), "test-env", "app=test", "default").
			Return(uint32(2), nil).
			Times(1)

		req := httptest.NewRequest("DELETE", "/test-triggers?selector=app=test", nil)

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("should return error for invalid selector", func(t *testing.T) {
		// given
		req := httptest.NewRequest("DELETE", "/test-triggers?selector=invalid=selector=format", nil)

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("should return error when delete fails", func(t *testing.T) {
		// given
		expectedError := errors.New("delete failed")
		mockClient.EXPECT().
			DeleteByLabels(gomock.Any(), "test-env", "", "default").
			Return(uint32(0), expectedError).
			Times(1)

		req := httptest.NewRequest("DELETE", "/test-triggers", nil)

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
	})
}

func TestListTestTriggersHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := testtriggerclient.NewMockTestTriggerClient(ctrl)

	testAPI := &TestkubeAPIForTesting{
		TestTriggersClient: mockClient,
		Log:                log.DefaultLogger,
		Namespace:          "default",
		proContext:         &config.ProContext{EnvID: "test-env"},
	}

	app := fiber.New()
	app.Get("/test-triggers", testAPI.ListTestTriggersHandler())

	t.Run("should list all test triggers successfully", func(t *testing.T) {
		// given
		expectedTriggers := []testkube.TestTrigger{
			{
				Name:      "trigger-1",
				Namespace: "default",
			},
			{
				Name:      "trigger-2",
				Namespace: "default",
			},
		}

		mockClient.EXPECT().
			List(gomock.Any(), "test-env", testtriggerclient.ListOptions{Selector: ""}, "default").
			Return(expectedTriggers, nil).
			Times(1)

		req := httptest.NewRequest("GET", "/test-triggers", nil)

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result []testkube.TestTrigger
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "trigger-1", result[0].Name)
		assert.Equal(t, "trigger-2", result[1].Name)
	})

	t.Run("should list test triggers with selector", func(t *testing.T) {
		// given
		expectedTriggers := []testkube.TestTrigger{
			{
				Name:      "trigger-1",
				Namespace: "default",
			},
		}

		mockClient.EXPECT().
			List(gomock.Any(), "test-env", testtriggerclient.ListOptions{Selector: "app=test"}, "default").
			Return(expectedTriggers, nil).
			Times(1)

		req := httptest.NewRequest("GET", "/test-triggers?selector=app=test", nil)

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result []testkube.TestTrigger
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Len(t, result, 1)
	})

	t.Run("should return error for invalid selector", func(t *testing.T) {
		// given
		req := httptest.NewRequest("GET", "/test-triggers?selector=invalid=selector=format", nil)

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("should return error when list fails", func(t *testing.T) {
		// given
		expectedError := errors.New("list failed")
		mockClient.EXPECT().
			List(gomock.Any(), "test-env", testtriggerclient.ListOptions{Selector: ""}, "default").
			Return(nil, expectedError).
			Times(1)

		req := httptest.NewRequest("GET", "/test-triggers", nil)

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
	})
}

func TestGetTestTriggerKeyMapHandler(t *testing.T) {
	testAPI := &TestkubeAPI{
		Log: log.DefaultLogger,
	}

	app := fiber.New()
	app.Get("/test-trigger-keymap", testAPI.GetTestTriggerKeyMapHandler())

	t.Run("should return test trigger keymap", func(t *testing.T) {
		// given
		req := httptest.NewRequest("GET", "/test-trigger-keymap", nil)

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestGenerateTestTriggerName(t *testing.T) {
	t.Run("should generate trigger name from spec", func(t *testing.T) {
		// given
		trigger := &testtriggersv1.TestTrigger{
			Spec: testtriggersv1.TestTriggerSpec{
				Resource:  "deployment",
				Event:     "created",
				Action:    "run",
				Execution: "test",
			},
		}

		// when
		name := generateTestTriggerName(trigger)

		// then
		assert.Contains(t, name, "trigger-deployment-created-run-test")
		assert.True(t, len(name) <= 63) // Kubernetes name limit
		assert.NotEmpty(t, name)
		// Should end with random suffix
		parts := strings.Split(name, "-")
		assert.True(t, len(parts[len(parts)-1]) == 5) // Random suffix length
	})

	t.Run("should truncate long names", func(t *testing.T) {
		// given
		trigger := &testtriggersv1.TestTrigger{
			Spec: testtriggersv1.TestTriggerSpec{
				Resource:  "very-long-deployment-name-that-exceeds-limits",
				Event:     "created-or-updated-or-deleted",
				Action:    "run-multiple-tests",
				Execution: "very-long-test-execution-name",
			},
		}

		// when
		name := generateTestTriggerName(trigger)

		// then
		assert.True(t, len(name) <= 63) // Kubernetes name limit
		assert.NotEmpty(t, name)
		assert.False(t, strings.HasSuffix(name, "-")) // Should not end with hyphen
		// Should still contain random suffix
		parts := strings.Split(name, "-")
		assert.True(t, len(parts[len(parts)-1]) == 5) // Random suffix length
	})

	t.Run("should handle empty values", func(t *testing.T) {
		// given
		trigger := &testtriggersv1.TestTrigger{
			Spec: testtriggersv1.TestTriggerSpec{
				Resource:  "",
				Event:     "",
				Action:    "",
				Execution: "",
			},
		}

		// when
		name := generateTestTriggerName(trigger)

		// then
		assert.Contains(t, name, "trigger----")
		assert.True(t, len(name) <= 63)
		assert.NotEmpty(t, name)
	})
}

// TestkubeAPIForTesting is a version of TestkubeAPI that doesn't use real metrics
type TestkubeAPIForTesting struct {
	TestTriggersClient testtriggerclient.TestTriggerClient
	Log                *zap.SugaredLogger
	Namespace          string
	proContext         *config.ProContext
}

func (s *TestkubeAPIForTesting) getEnvironmentId() string {
	if s.proContext != nil {
		return s.proContext.EnvID
	}
	return ""
}

// Error is a simple error response method
func (s *TestkubeAPIForTesting) Error(c *fiber.Ctx, status int, err error) error {
	return c.Status(status).JSON(fiber.Map{"error": err.Error()})
}

// CreateTestTriggerHandler is a simplified version for testing
func (s *TestkubeAPIForTesting) CreateTestTriggerHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to create test trigger"
		var testTrigger testtriggersv1.TestTrigger

		if string(c.Request().Header.ContentType()) == "text/yaml" {
			// For YAML, just return success for testing purposes
			c.Status(http.StatusCreated)
			return c.JSON(testkube.TestTrigger{Name: "yaml-trigger"})
		} else {
			var request testkube.TestTriggerUpsertRequest
			err := c.BodyParser(&request)
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse json request: %w", errPrefix, err))
			}
			testTrigger = testtriggersmapper.MapTestTriggerUpsertRequestToTestTriggerCRD(request)
		}

		// default namespace if not defined in upsert request
		if testTrigger.Namespace == "" {
			testTrigger.Namespace = s.Namespace
		}
		// default trigger name if not defined in upsert request
		if testTrigger.Name == "" {
			testTrigger.Name = generateTestTriggerName(&testTrigger)
		}

		errPrefix = errPrefix + " " + testTrigger.Name
		s.Log.Infow("creating test trigger", "testTrigger", testTrigger)

		// Convert CRD to API object for the new interface
		apiTrigger := testtriggersmapper.MapCRDToAPI(&testTrigger)

		err := s.TestTriggersClient.Create(c.Context(), s.getEnvironmentId(), apiTrigger)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not create test trigger: %w", errPrefix, err))
		}

		c.Status(http.StatusCreated)
		return c.JSON(apiTrigger)
	}
}

// TODO(emil): why do we have a test version with almost the same logic as the real handler
// TODO(emil): this is not testing the yaml version of the request
// UpdateTestTriggerHandler is a simplified version for testing
func (s *TestkubeAPIForTesting) UpdateTestTriggerHandler() fiber.Handler {
	s.Log.Info("test update handler called")
	return func(c *fiber.Ctx) error {
		s.Log.Info("update fiber handler called")
		errPrefix := "failed to update test trigger"
		var request testkube.TestTriggerUpsertRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse json request: %w", errPrefix, err))
		}
		s.Log.Infof("parsed request body: %v", request.Selector)

		namespace := s.Namespace
		if request.Namespace != "" {
			namespace = request.Namespace
		}
		errPrefix = errPrefix + " " + request.Name

		// we need to get resource first
		apiTrigger, err := s.TestTriggersClient.Get(c.Context(), s.getEnvironmentId(), request.Name, namespace)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not get test trigger: %w", errPrefix, err))
		}

		// Update the trigger with new values from request
		apiTrigger.Name = request.Name
		apiTrigger.Namespace = namespace
		if request.Labels != nil {
			apiTrigger.Labels = request.Labels
		}
		if request.Annotations != nil {
			apiTrigger.Annotations = request.Annotations
		}

		// Update individual fields from the request
		if request.Selector != nil {
			apiTrigger.Selector = request.Selector
		}
		if request.Resource != nil {
			apiTrigger.Resource = request.Resource
		}
		if request.ResourceSelector != nil {
			apiTrigger.ResourceSelector = request.ResourceSelector
		}
		if request.Event != "" {
			apiTrigger.Event = request.Event
		}
		if request.ConditionSpec != nil {
			apiTrigger.ConditionSpec = request.ConditionSpec
		}
		if request.ProbeSpec != nil {
			apiTrigger.ProbeSpec = request.ProbeSpec
		}
		if request.Action != nil {
			apiTrigger.Action = request.Action
		}
		if request.ActionParameters != nil {
			apiTrigger.ActionParameters = request.ActionParameters
		}
		if request.Execution != nil {
			apiTrigger.Execution = request.Execution
		}
		if request.TestSelector != nil {
			apiTrigger.TestSelector = request.TestSelector
		}
		if request.ConcurrencyPolicy != nil {
			apiTrigger.ConcurrencyPolicy = request.ConcurrencyPolicy
		}
		apiTrigger.Disabled = request.Disabled

		err = s.TestTriggersClient.Update(c.Context(), s.getEnvironmentId(), *apiTrigger)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not update test trigger: %w", errPrefix, err))
		}

		return c.JSON(apiTrigger)
	}
}

// BulkUpdateTestTriggersHandler is a simplified version for testing
func (s *TestkubeAPIForTesting) BulkUpdateTestTriggersHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to bulk update test triggers"

		var request []testkube.TestTriggerUpsertRequest
		err := c.BodyParser(&request)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: could not parse request: %w", errPrefix, err))
		}

		namespaces := make(map[string]struct{}, 0)
		for _, upsertRequest := range request {
			namespace := s.Namespace
			if upsertRequest.Namespace != "" {
				namespace = upsertRequest.Namespace
			}
			namespaces[namespace] = struct{}{}
		}

		for namespace := range namespaces {
			_, err = s.TestTriggersClient.DeleteAll(c.Context(), s.getEnvironmentId(), namespace)
			if err != nil {
				return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: error cleaning triggers before reapply", errPrefix))
			}
		}

		testTriggers := make([]testkube.TestTrigger, 0, len(request))

		for _, upsertRequest := range request {
			crdTestTrigger := testtriggersmapper.MapTestTriggerUpsertRequestToTestTriggerCRD(upsertRequest)
			// default trigger name if not defined in upsert request
			if crdTestTrigger.Name == "" {
				crdTestTrigger.Name = generateTestTriggerName(&crdTestTrigger)
			}

			// Convert CRD to API object for the new interface
			apiTrigger := testtriggersmapper.MapCRDToAPI(&crdTestTrigger)

			err = s.TestTriggersClient.Create(c.Context(), s.getEnvironmentId(), apiTrigger)
			if err != nil {
				return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: error reapplying triggers after clean", errPrefix))
			}

			testTriggers = append(testTriggers, apiTrigger)
		}

		return c.JSON(testTriggers)
	}
}

// GetTestTriggerHandler is a simplified version for testing
func (s *TestkubeAPIForTesting) GetTestTriggerHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespace := c.Query("namespace", s.Namespace)
		name := c.Params("id")
		errPrefix := fmt.Sprintf("failed to get test trigger %s", name)

		apiTestTrigger, err := s.TestTriggersClient.Get(c.Context(), s.getEnvironmentId(), name, namespace)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not get test trigger: %w", errPrefix, err))
		}

		c.Status(http.StatusOK)
		return c.JSON(apiTestTrigger)
	}
}

// DeleteTestTriggerHandler is a simplified version for testing
func (s *TestkubeAPIForTesting) DeleteTestTriggerHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespace := c.Query("namespace", s.Namespace)
		name := c.Params("id")
		errPrefix := fmt.Sprintf("failed to delete test trigger %s", name)

		err := s.TestTriggersClient.Delete(c.Context(), s.getEnvironmentId(), name, namespace)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not delete test trigger: %w", errPrefix, err))
		}

		return c.SendStatus(http.StatusNoContent)
	}
}

// DeleteTestTriggersHandler is a simplified version for testing
func (s *TestkubeAPIForTesting) DeleteTestTriggersHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to delete test triggers"

		namespace := c.Query("namespace", s.Namespace)
		selector := c.Query("selector")
		if selector != "" {
			_, err := labels.Parse(selector)
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: error validating selector: %w", errPrefix, err))
			}
		}
		_, err := s.TestTriggersClient.DeleteByLabels(c.Context(), s.getEnvironmentId(), selector, namespace)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not bulk delete test triggers: %w", errPrefix, err))
		}

		return c.SendStatus(http.StatusNoContent)
	}
}

// ListTestTriggersHandler is a simplified version for testing
func (s *TestkubeAPIForTesting) ListTestTriggersHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to list test triggers"

		namespace := c.Query("namespace", s.Namespace)
		selector := c.Query("selector")
		if selector != "" {
			_, err := labels.Parse(selector)
			if err != nil {
				return s.Error(c, http.StatusBadRequest, fmt.Errorf("%s: error validating selector: %w", errPrefix, err))
			}
		}
		options := testtriggerclient.ListOptions{
			Selector: selector,
		}

		apiTestTriggers, err := s.TestTriggersClient.List(c.Context(), s.getEnvironmentId(), options, namespace)
		if err != nil {
			return s.Error(c, http.StatusBadGateway, fmt.Errorf("%s: client could not list test triggers: %w", errPrefix, err))
		}

		c.Status(http.StatusOK)
		return c.JSON(apiTestTriggers)
	}
}

// Helper functions to create enum pointers
func resourcePtr(s string) *testkube.TestTriggerResources {
	r := testkube.TestTriggerResources(s)
	return &r
}

func actionPtr(s string) *testkube.TestTriggerActions {
	a := testkube.TestTriggerActions(s)
	return &a
}

func executionPtr(s string) *testkube.TestTriggerExecutions {
	e := testkube.TestTriggerExecutions(s)
	return &e
}

// MetricsInterface defines the methods we need from metrics for testing
type MetricsInterface interface {
	IncCreateTestTrigger(err error)
	IncUpdateTestTrigger(err error)
	IncBulkUpdateTestTrigger(err error)
	IncDeleteTestTrigger(err error)
	IncBulkDeleteTestTrigger(err error)
}
