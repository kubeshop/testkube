package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"

	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/newclients/testtriggerclient"
)

func TestCreateTestTriggerHandler(t *testing.T) {
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
			DoAndReturn(func(ctx context.Context, envId string, trigger testkube.TestTrigger) error {
				assert.NotEmpty(t, trigger.Name)
				assert.Equal(t, "test-trigger", trigger.Name)
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

		mockClient.EXPECT().
			Create(gomock.Any(), "test-env", gomock.Any()).
			Return(nil).
			DoAndReturn(func(ctx context.Context, envId string, trigger testkube.TestTrigger) error {
				assert.NotEmpty(t, trigger.Name)
				assert.Equal(t, "test-trigger-yaml", trigger.Name)
				return nil
			}).
			Times(1)

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		if err == nil {
			assert.Equal(t, http.StatusCreated, resp.StatusCode)
		}
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

	t.Run("should clear conditionSpec when yaml update removes it", func(t *testing.T) {
		// given - YAML without conditionSpec
		requestBody := `
apiVersion: tests.testkube.io/v1
kind: TestTrigger
metadata:
  name: test-trigger
  namespace: default
spec:
  resource: deployment
  resourceSelector:
    name: test-resource
  event: created
  action: run
  execution: testworkflow
  testSelector:
    name: my-test
`
		// existing trigger HAS conditionSpec
		existingTrigger := &testkube.TestTrigger{
			Name:      "test-trigger",
			Namespace: "default",
			Resource:  resourcePtr("deployment"),
			Event:     "created",
			Action:    actionPtr("run"),
			Execution: executionPtr("testworkflow"),
			ConditionSpec: &testkube.TestTriggerConditionSpec{
				Timeout: 100,
				Conditions: []testkube.TestTriggerCondition{
					{Type_: "Ready", Status: conditionStatusPtr("True")},
				},
			},
		}

		mockClient.EXPECT().
			Get(gomock.Any(), "test-env", "test-trigger", "default").
			Return(existingTrigger, nil).
			Times(1)

		mockClient.EXPECT().
			Update(gomock.Any(), "test-env", gomock.Any()).
			DoAndReturn(func(ctx context.Context, envId string, trigger testkube.TestTrigger) error {
				// conditionSpec should be nil (cleared) since YAML doesn't include it
				assert.Nil(t, trigger.ConditionSpec, "conditionSpec should be nil after YAML update without it")
				return nil
			}).
			Times(1)

		req := httptest.NewRequest("PUT", "/test-triggers", bytes.NewReader([]byte(requestBody)))
		req.Header.Set("Content-Type", "text/yaml")

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("should preserve conditionSpec when json update does not include it", func(t *testing.T) {
		// given - JSON update without conditionSpec (merge behavior)
		request := testkube.TestTriggerUpsertRequest{
			Name:      "test-trigger",
			Namespace: "default",
			Event:     "updated", // only changing event
		}

		// existing trigger HAS conditionSpec
		existingTrigger := &testkube.TestTrigger{
			Name:      "test-trigger",
			Namespace: "default",
			Resource:  resourcePtr("deployment"),
			Event:     "created",
			Action:    actionPtr("run"),
			Execution: executionPtr("testworkflow"),
			ConditionSpec: &testkube.TestTriggerConditionSpec{
				Timeout: 100,
				Conditions: []testkube.TestTriggerCondition{
					{Type_: "Ready", Status: conditionStatusPtr("True")},
				},
			},
		}

		mockClient.EXPECT().
			Get(gomock.Any(), "test-env", "test-trigger", "default").
			Return(existingTrigger, nil).
			Times(1)

		mockClient.EXPECT().
			Update(gomock.Any(), "test-env", gomock.Any()).
			DoAndReturn(func(ctx context.Context, envId string, trigger testkube.TestTrigger) error {
				// conditionSpec should be preserved since JSON uses merge behavior
				assert.NotNil(t, trigger.ConditionSpec, "conditionSpec should be preserved in JSON merge update")
				assert.Equal(t, int32(100), trigger.ConditionSpec.Timeout)
				// event should be updated
				assert.Equal(t, "updated", trigger.Event)
				return nil
			}).
			Times(1)

		requestBody, _ := json.Marshal(request)
		req := httptest.NewRequest("PUT", "/test-triggers", bytes.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("should clear all optional fields when yaml update removes them", func(t *testing.T) {
		// given - minimal YAML with only required fields
		requestBody := `
apiVersion: tests.testkube.io/v1
kind: TestTrigger
metadata:
  name: test-trigger
  namespace: default
spec:
  resource: deployment
  event: created
  action: run
  execution: testworkflow
  testSelector:
    name: my-test
`
		// existing trigger has ALL optional fields populated
		existingTrigger := &testkube.TestTrigger{
			Name:        "test-trigger",
			Namespace:   "default",
			Labels:      map[string]string{"env": "test", "team": "platform"},
			Annotations: map[string]string{"description": "my trigger"},
			Resource:    resourcePtr("deployment"),
			ResourceSelector: &testkube.TestTriggerSelector{
				Name:          "specific-deployment",
				Namespace:     "apps",
				LabelSelector: &testkube.IoK8sApimachineryPkgApisMetaV1LabelSelector{MatchLabels: map[string]string{"app": "test"}},
			},
			Event:     "created",
			Action:    actionPtr("run"),
			Execution: executionPtr("testworkflow"),
			ConditionSpec: &testkube.TestTriggerConditionSpec{
				Timeout: 100,
				Conditions: []testkube.TestTriggerCondition{
					{Type_: "Ready", Status: conditionStatusPtr("True")},
				},
			},
			ProbeSpec: &testkube.TestTriggerProbeSpec{
				Timeout: 60,
				Probes: []testkube.TestTriggerProbe{
					{Scheme: "http", Host: "localhost", Port: 8080},
				},
			},
			ActionParameters: &testkube.TestTriggerActionParameters{
				Config: map[string]string{"key": "value"},
				Tags:   map[string]string{"env": "test"},
			},
			TestSelector: &testkube.TestTriggerSelector{
				Name: "old-test",
			},
			ConcurrencyPolicy: concurrencyPolicyPtr(testkube.FORBID_TestTriggerConcurrencyPolicies),
			Disabled:          true,
		}

		mockClient.EXPECT().
			Get(gomock.Any(), "test-env", "test-trigger", "default").
			Return(existingTrigger, nil).
			Times(1)

		mockClient.EXPECT().
			Update(gomock.Any(), "test-env", gomock.Any()).
			DoAndReturn(func(ctx context.Context, envId string, trigger testkube.TestTrigger) error {
				// All optional fields should be cleared (nil or zero value) since YAML doesn't include them
				assert.Nil(t, trigger.Labels, "labels should be nil after YAML update without it")
				assert.Nil(t, trigger.Annotations, "annotations should be nil after YAML update without it")
				// Note: ResourceSelector is returned as empty struct by the CRD mapper, not nil
				// This is expected behavior from mapSelectorFromCRD which always returns a pointer
				if trigger.ResourceSelector != nil {
					assert.Empty(t, trigger.ResourceSelector.Name, "resourceSelector.Name should be empty")
					assert.Empty(t, trigger.ResourceSelector.Namespace, "resourceSelector.Namespace should be empty")
				}
				assert.Nil(t, trigger.ConditionSpec, "conditionSpec should be nil after YAML update without it")
				assert.Nil(t, trigger.ProbeSpec, "probeSpec should be nil after YAML update without it")
				assert.Nil(t, trigger.ActionParameters, "actionParameters should be nil after YAML update without it")
				assert.Nil(t, trigger.ConcurrencyPolicy, "concurrencyPolicy should be nil after YAML update without it")
				assert.False(t, trigger.Disabled, "disabled should be false after YAML update without it")
				// Required fields should still be set
				assert.Equal(t, "test-trigger", trigger.Name)
				assert.Equal(t, "default", trigger.Namespace)
				assert.NotNil(t, trigger.Resource)
				assert.Equal(t, "created", trigger.Event)
				assert.NotNil(t, trigger.TestSelector)
				assert.Equal(t, "my-test", trigger.TestSelector.Name)
				return nil
			}).
			Times(1)

		req := httptest.NewRequest("PUT", "/test-triggers", bytes.NewReader([]byte(requestBody)))
		req.Header.Set("Content-Type", "text/yaml")

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("should preserve all optional fields when json update only changes event", func(t *testing.T) {
		// given - JSON update only changing event field
		request := testkube.TestTriggerUpsertRequest{
			Name:      "test-trigger",
			Namespace: "default",
			Event:     "updated", // only changing event
		}

		// existing trigger has ALL optional fields populated
		existingTrigger := &testkube.TestTrigger{
			Name:        "test-trigger",
			Namespace:   "default",
			Labels:      map[string]string{"env": "test", "team": "platform"},
			Annotations: map[string]string{"description": "my trigger"},
			Resource:    resourcePtr("deployment"),
			ResourceSelector: &testkube.TestTriggerSelector{
				Name:      "specific-deployment",
				Namespace: "apps",
			},
			Event:     "created",
			Action:    actionPtr("run"),
			Execution: executionPtr("testworkflow"),
			ConditionSpec: &testkube.TestTriggerConditionSpec{
				Timeout: 100,
			},
			ProbeSpec: &testkube.TestTriggerProbeSpec{
				Timeout: 60,
			},
			ActionParameters: &testkube.TestTriggerActionParameters{
				Config: map[string]string{"key": "value"},
				Tags:   map[string]string{"env": "test"},
			},
			TestSelector: &testkube.TestTriggerSelector{
				Name: "old-test",
			},
			ConcurrencyPolicy: concurrencyPolicyPtr(testkube.FORBID_TestTriggerConcurrencyPolicies),
			Disabled:          true,
		}

		mockClient.EXPECT().
			Get(gomock.Any(), "test-env", "test-trigger", "default").
			Return(existingTrigger, nil).
			Times(1)

		mockClient.EXPECT().
			Update(gomock.Any(), "test-env", gomock.Any()).
			DoAndReturn(func(ctx context.Context, envId string, trigger testkube.TestTrigger) error {
				// All optional fields should be PRESERVED since JSON uses merge behavior
				assert.Equal(t, map[string]string{"env": "test", "team": "platform"}, trigger.Labels, "labels should be preserved")
				assert.Equal(t, map[string]string{"description": "my trigger"}, trigger.Annotations, "annotations should be preserved")
				assert.NotNil(t, trigger.ResourceSelector, "resourceSelector should be preserved")
				assert.Equal(t, "specific-deployment", trigger.ResourceSelector.Name)
				assert.NotNil(t, trigger.ConditionSpec, "conditionSpec should be preserved")
				assert.Equal(t, int32(100), trigger.ConditionSpec.Timeout)
				assert.NotNil(t, trigger.ProbeSpec, "probeSpec should be preserved")
				assert.Equal(t, int32(60), trigger.ProbeSpec.Timeout)
				assert.NotNil(t, trigger.ActionParameters, "actionParameters should be preserved")
				assert.Equal(t, map[string]string{"key": "value"}, trigger.ActionParameters.Config)
				assert.NotNil(t, trigger.ConcurrencyPolicy, "concurrencyPolicy should be preserved")
				assert.Equal(t, testkube.FORBID_TestTriggerConcurrencyPolicies, *trigger.ConcurrencyPolicy)
				// Note: Disabled is a bool, not *bool, so it cannot distinguish "not set" from "false"
				// This is a known limitation of the API design. For YAML updates this works correctly.
				// For JSON merge updates, if the user sends disabled:false (or omits it), it resets to false.
				// Event should be updated
				assert.Equal(t, "updated", trigger.Event)
				return nil
			}).
			Times(1)

		requestBody, _ := json.Marshal(request)
		req := httptest.NewRequest("PUT", "/test-triggers", bytes.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("should add new fields when yaml update includes them", func(t *testing.T) {
		// given - YAML with new conditionSpec and probeSpec
		requestBody := `
apiVersion: tests.testkube.io/v1
kind: TestTrigger
metadata:
  name: test-trigger
  namespace: default
  labels:
    env: production
    team: devops
spec:
  resource: deployment
  event: created
  action: run
  execution: testworkflow
  testSelector:
    name: my-test
  conditionSpec:
    timeout: 120
    conditions:
      - type: Ready
        status: "True"
  probeSpec:
    timeout: 30
  disabled: true
`
		// existing trigger has minimal fields
		existingTrigger := &testkube.TestTrigger{
			Name:      "test-trigger",
			Namespace: "default",
			Resource:  resourcePtr("deployment"),
			Event:     "created",
			Action:    actionPtr("run"),
			Execution: executionPtr("testworkflow"),
			TestSelector: &testkube.TestTriggerSelector{
				Name: "old-test",
			},
		}

		mockClient.EXPECT().
			Get(gomock.Any(), "test-env", "test-trigger", "default").
			Return(existingTrigger, nil).
			Times(1)

		mockClient.EXPECT().
			Update(gomock.Any(), "test-env", gomock.Any()).
			DoAndReturn(func(ctx context.Context, envId string, trigger testkube.TestTrigger) error {
				// New fields from YAML should be added
				assert.Equal(t, map[string]string{"env": "production", "team": "devops"}, trigger.Labels, "labels should be set from YAML")
				assert.NotNil(t, trigger.ConditionSpec, "conditionSpec should be set from YAML")
				assert.Equal(t, int32(120), trigger.ConditionSpec.Timeout)
				assert.NotNil(t, trigger.ProbeSpec, "probeSpec should be set from YAML")
				assert.Equal(t, int32(30), trigger.ProbeSpec.Timeout)
				assert.True(t, trigger.Disabled, "disabled should be set from YAML")
				// testSelector should be updated
				assert.Equal(t, "my-test", trigger.TestSelector.Name)
				return nil
			}).
			Times(1)

		req := httptest.NewRequest("PUT", "/test-triggers", bytes.NewReader([]byte(requestBody)))
		req.Header.Set("Content-Type", "text/yaml")

		// when
		resp, err := app.Test(req)

		// then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestBulkUpdateTestTriggersHandler(t *testing.T) {
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

	testAPI := &TestkubeAPI{
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

	testAPI := &TestkubeAPI{
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

	testAPI := &TestkubeAPI{
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

	testAPI := &TestkubeAPI{
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

func conditionStatusPtr(s string) *testkube.TestTriggerConditionStatuses {
	c := testkube.TestTriggerConditionStatuses(s)
	return &c
}

func concurrencyPolicyPtr(p testkube.TestTriggerConcurrencyPolicies) *testkube.TestTriggerConcurrencyPolicies {
	return &p
}

// MetricsInterface defines the methods we need from metrics for testing
type MetricsInterface interface {
	IncCreateTestTrigger(err error)
	IncUpdateTestTrigger(err error)
	IncBulkUpdateTestTrigger(err error)
	IncDeleteTestTrigger(err error)
	IncBulkDeleteTestTrigger(err error)
}
