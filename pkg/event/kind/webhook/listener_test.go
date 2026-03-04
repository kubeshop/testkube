package webhook

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	v1 "github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	cloudwebhook "github.com/kubeshop/testkube/pkg/cloud/data/webhook"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
)

const executionID = "id-1"

var testEventTypes = []testkube.EventType{testkube.EventType("")}

func TestWebhookListener_Notify(t *testing.T) {
	t.Run("send event success response", func(t *testing.T) {
		// given
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var event testkube.Event
			err := json.NewDecoder(r.Body).Decode(&event)
			// then
			assert.NoError(t, err)
			assert.Equal(t, executionID, event.TestWorkflowExecution.Id)
		})

		svr := httptest.NewServer(testHandler)
		defer svr.Close()

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockWebhookRepository := cloudwebhook.NewMockWebhookRepository(mockCtrl)
		mockWebhookRepository.EXPECT().CollectExecutionResult(gomock.Any(), gomock.Any(), "l1", "", http.StatusOK).AnyTimes()
		l := NewWebhookListener("l1", svr.URL, "", testEventTypes, "", "", nil, false, nil, nil, listenerWithMetrics(v1.NewMetrics()), listenerWithWebhookResultsRepository(mockWebhookRepository))

		// when
		r := l.Notify(testkube.Event{
			Type_:                 testkube.EventStartTestWorkflow,
			TestWorkflowExecution: exampleExecution(),
		})

		assert.Equal(t, "", r.Error())

	})

	t.Run("send event failed response", func(t *testing.T) {
		// given
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
		})

		svr := httptest.NewServer(testHandler)
		defer svr.Close()

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockWebhookRepository := cloudwebhook.NewMockWebhookRepository(mockCtrl)
		mockWebhookRepository.EXPECT().CollectExecutionResult(gomock.Any(), gomock.Any(), "l1", gomock.Any(), http.StatusBadGateway).AnyTimes()
		l := NewWebhookListener("l1", svr.URL, "", testEventTypes, "", "", nil, false, nil, nil, listenerWithMetrics(v1.NewMetrics()), listenerWithWebhookResultsRepository(mockWebhookRepository))

		// when
		r := l.Notify(testkube.Event{
			Type_:                 testkube.EventStartTestWorkflow,
			TestWorkflowExecution: exampleExecution(),
		})

		// then
		assert.NotEqual(t, "", r.Error())

	})

	t.Run("send event bad uri", func(t *testing.T) {
		// given

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockWebhookRepository := cloudwebhook.NewMockWebhookRepository(mockCtrl)
		mockWebhookRepository.EXPECT().CollectExecutionResult(gomock.Any(), gomock.Any(), "l1", gomock.Any(), 0).AnyTimes()
		s := NewWebhookListener("l1", "http://baduri.badbadbad", "", testEventTypes, "", "", nil, false, nil, nil, listenerWithMetrics(v1.NewMetrics()), listenerWithWebhookResultsRepository(mockWebhookRepository))

		// when
		r := s.Notify(testkube.Event{
			Type_:                 testkube.EventStartTestWorkflow,
			TestWorkflowExecution: exampleExecution(),
		})

		// then
		assert.NotEqual(t, "", r.Error())
	})

	t.Run("send event success response using payload field", func(t *testing.T) {
		// given
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body := bytes.NewBuffer([]byte{})
			err := json.NewEncoder(body).Encode(testkube.Event{
				Type_:                 testkube.EventStartTestWorkflow,
				TestWorkflowExecution: exampleExecution(),
			})
			assert.NoError(t, err)

			data := make(map[string]string, 0)
			err = json.NewDecoder(r.Body).Decode(&data)
			// then
			assert.NoError(t, err)
			assert.Equal(t, body.String(), data["field"])
		})

		svr := httptest.NewServer(testHandler)
		defer svr.Close()

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockWebhookRepository := cloudwebhook.NewMockWebhookRepository(mockCtrl)
		mockWebhookRepository.EXPECT().CollectExecutionResult(gomock.Any(), gomock.Any(), "l1", "", http.StatusOK).AnyTimes()
		l := NewWebhookListener("l1", svr.URL, "", testEventTypes, "field", "", nil, false, nil, nil, listenerWithMetrics(v1.NewMetrics()), listenerWithWebhookResultsRepository(mockWebhookRepository))

		// when
		r := l.Notify(testkube.Event{
			Type_:                 testkube.EventStartTestWorkflow,
			TestWorkflowExecution: exampleExecution(),
		})

		assert.Equal(t, "", r.Error())

	})

	t.Run("send event success response using payload template", func(t *testing.T) {
		// given
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err)

			// then
			assert.Equal(t, "{\"id\": \"12345\"}", string(body))
		})

		svr := httptest.NewServer(testHandler)
		defer svr.Close()

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockWebhookRepository := cloudwebhook.NewMockWebhookRepository(mockCtrl)
		mockWebhookRepository.EXPECT().CollectExecutionResult(gomock.Any(), gomock.Any(), "l1", "", http.StatusOK).AnyTimes()
		l := NewWebhookListener("l1", svr.URL, "", testEventTypes, "", "{\"id\": \"{{ .Id }}\"}",
			map[string]string{"Content-Type": "application/json"}, false, nil, nil, listenerWithMetrics(v1.NewMetrics()), listenerWithWebhookResultsRepository(mockWebhookRepository))

		// when
		r := l.Notify(testkube.Event{
			Id:                    "12345",
			Type_:                 testkube.EventStartTestWorkflow,
			TestWorkflowExecution: exampleExecution(),
		})

		assert.Equal(t, "", r.Error())

	})

	t.Run("send event disabled webhook", func(t *testing.T) {
		// given

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockWebhookRepository := cloudwebhook.NewMockWebhookRepository(mockCtrl)
		mockWebhookRepository.EXPECT().CollectExecutionResult(gomock.Any(), gomock.Any(), "l1", "", 0).AnyTimes()
		s := NewWebhookListener("l1", "http://baduri.badbadbad", "", testEventTypes, "", "", nil, true, nil, nil, listenerWithMetrics(v1.NewMetrics()), listenerWithWebhookResultsRepository(mockWebhookRepository))

		// when
		match := s.Match(testkube.Event{
			Type_:                 testkube.EventStartTestWorkflow,
			TestWorkflowExecution: exampleExecution(),
		})

		// then
		assert.False(t, match)
	})

	t.Run("become event - state did not change - webhook should not execute", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		// Mock repository to return previous status as PASSED
		mockTestWorkflowRepo := testworkflow.NewMockRepository(mockCtrl)
		mockTestWorkflowRepo.EXPECT().
			GetPreviousFinishedState(gomock.Any(), "test-workflow", gomock.Any()).
			Return(testkube.PASSED_TestWorkflowStatus, nil).
			Times(1)

		// Mock webhook repository - should NOT be called since webhook won't execute
		mockWebhookRepository := cloudwebhook.NewMockWebhookRepository(mockCtrl)
		// No expectation set - if it's called, the test will fail

		// Create a webhook that will never be called
		callCount := 0
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
		})
		svr := httptest.NewServer(testHandler)
		defer svr.Close()

		// Setup listener with become event
		becomeUpEvent := testkube.BECOME_TESTWORKFLOW_UP_EventType
		l := NewWebhookListener("l1", svr.URL, "", []testkube.EventType{becomeUpEvent}, "", "", nil, false, nil, nil,
			listenerWithMetrics(v1.NewMetrics()),
			listenerWithWebhookResultsRepository(mockWebhookRepository),
			listenerWithTestWorkflowResultsRepository(mockTestWorkflowRepo))

		// Current execution is PASSED, previous was also PASSED
		currentStatus := testkube.PASSED_TestWorkflowStatus
		execution := exampleFinishedExecution(currentStatus)

		// when - notify with become-testworkflow-up event
		result := l.Notify(testkube.Event{
			Type_:                 &becomeUpEvent,
			TestWorkflowExecution: execution,
		})

		// then
		// Webhook should return success but NOT execute the HTTP call
		assert.Equal(t, "", result.Error(), "Expected no error")
		assert.Equal(t, "webhook is set to become state only; state has not become", result.Result, "Expected skip message")
		assert.Equal(t, 0, callCount, "Webhook HTTP handler should not have been called")
	})

	t.Run("become event - state changed from failed to passed - webhook should execute", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		// Mock repository to return previous status as FAILED
		mockTestWorkflowRepo := testworkflow.NewMockRepository(mockCtrl)
		mockTestWorkflowRepo.EXPECT().
			GetPreviousFinishedState(gomock.Any(), "test-workflow", gomock.Any()).
			Return(testkube.FAILED_TestWorkflowStatus, nil).
			Times(1)

		// Mock webhook repository - expecting HTTP 200 since webhook should execute
		mockWebhookRepository := cloudwebhook.NewMockWebhookRepository(mockCtrl)
		mockWebhookRepository.EXPECT().
			CollectExecutionResult(gomock.Any(), gomock.Any(), "l1", "", http.StatusOK).
			Times(1)

		// Create webhook endpoint
		callCount := 0
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
		})
		svr := httptest.NewServer(testHandler)
		defer svr.Close()

		// Setup listener with become event
		becomeUpEvent := testkube.BECOME_TESTWORKFLOW_UP_EventType
		l := NewWebhookListener("l1", svr.URL, "", []testkube.EventType{becomeUpEvent}, "", "", nil, false, nil, nil,
			listenerWithMetrics(v1.NewMetrics()),
			listenerWithWebhookResultsRepository(mockWebhookRepository),
			listenerWithTestWorkflowResultsRepository(mockTestWorkflowRepo))

		// Current execution is PASSED, previous was FAILED
		currentStatus := testkube.PASSED_TestWorkflowStatus
		execution := exampleFinishedExecution(currentStatus)

		// when - notify with become-testworkflow-up event
		result := l.Notify(testkube.Event{
			Type_:                 &becomeUpEvent,
			TestWorkflowExecution: execution,
		})

		// then
		// Webhook SHOULD execute the HTTP call
		assert.Equal(t, "", result.Error(), "Expected no error")
		assert.Equal(t, 1, callCount, "Webhook HTTP handler should have been called exactly once")
	})
}

func exampleExecution() *testkube.TestWorkflowExecution {
	execution := testkube.NewQueuedExecution()
	execution.Id = executionID
	return execution
}

func exampleFinishedExecution(status testkube.TestWorkflowStatus) *testkube.TestWorkflowExecution {
	execution := testkube.NewQueuedExecution()
	execution.Id = executionID
	execution.Workflow = &testkube.TestWorkflow{Name: "test-workflow"}
	execution.Result = &testkube.TestWorkflowResult{
		Status: &status,
	}
	return execution
}
