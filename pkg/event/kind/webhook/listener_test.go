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
}

func exampleExecution() *testkube.TestWorkflowExecution {
	execution := testkube.NewQueuedExecution()
	execution.Id = executionID
	return execution
}

func TestWebhookListener_hasBecomeState(t *testing.T) {
	t.Run("should return true when no previous finished state exists (first execution)", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		mockRepo := testworkflow.NewMockRepository(mockCtrl)
		mockRepo.EXPECT().
			GetPreviousFinishedState(gomock.Any(), "test-workflow", gomock.Any()).
			Return(testkube.TestWorkflowStatus(""), nil)

		mockWebhookRepository := cloudwebhook.NewMockWebhookRepository(mockCtrl)
		l := NewWebhookListener(
			"test-listener",
			"http://example.com",
			"",
			[]testkube.EventType{testkube.BECOME_TESTWORKFLOW_FAILED_EventType},
			"",
			"",
			nil,
			false,
			nil,
			nil,
			listenerWithTestWorkflowResultsRepository(mockRepo),
			listenerWithMetrics(v1.NewMetrics()),
			listenerWithWebhookResultsRepository(mockWebhookRepository),
		)

		event := testkube.Event{
			Type_: testkube.EventTypePtr(testkube.BECOME_TESTWORKFLOW_FAILED_EventType),
			TestWorkflowExecution: &testkube.TestWorkflowExecution{
				Workflow: &testkube.TestWorkflow{
					Name: "test-workflow",
				},
			},
		}

		// when
		became, err := l.hasBecomeState(event)

		// then
		assert.NoError(t, err)
		assert.True(t, became, "should return true for first execution (no previous state)")
	})

	t.Run("should return true when previous state is passed and current is failed", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		mockRepo := testworkflow.NewMockRepository(mockCtrl)
		mockRepo.EXPECT().
			GetPreviousFinishedState(gomock.Any(), "test-workflow", gomock.Any()).
			Return(testkube.PASSED_TestWorkflowStatus, nil)

		mockWebhookRepository := cloudwebhook.NewMockWebhookRepository(mockCtrl)
		l := NewWebhookListener(
			"test-listener",
			"http://example.com",
			"",
			[]testkube.EventType{testkube.BECOME_TESTWORKFLOW_FAILED_EventType},
			"",
			"",
			nil,
			false,
			nil,
			nil,
			listenerWithTestWorkflowResultsRepository(mockRepo),
			listenerWithMetrics(v1.NewMetrics()),
			listenerWithWebhookResultsRepository(mockWebhookRepository),
		)

		event := testkube.Event{
			Type_: testkube.EventTypePtr(testkube.BECOME_TESTWORKFLOW_FAILED_EventType),
			TestWorkflowExecution: &testkube.TestWorkflowExecution{
				Workflow: &testkube.TestWorkflow{
					Name: "test-workflow",
				},
			},
		}

		// when
		became, err := l.hasBecomeState(event)

		// then
		assert.NoError(t, err)
		assert.True(t, became, "should return true when transitioning from passed to failed")
	})

	t.Run("should return false when previous state is failed and current is failed", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		mockRepo := testworkflow.NewMockRepository(mockCtrl)
		mockRepo.EXPECT().
			GetPreviousFinishedState(gomock.Any(), "test-workflow", gomock.Any()).
			Return(testkube.FAILED_TestWorkflowStatus, nil)

		mockWebhookRepository := cloudwebhook.NewMockWebhookRepository(mockCtrl)
		l := NewWebhookListener(
			"test-listener",
			"http://example.com",
			"",
			[]testkube.EventType{testkube.BECOME_TESTWORKFLOW_FAILED_EventType},
			"",
			"",
			nil,
			false,
			nil,
			nil,
			listenerWithTestWorkflowResultsRepository(mockRepo),
			listenerWithMetrics(v1.NewMetrics()),
			listenerWithWebhookResultsRepository(mockWebhookRepository),
		)

		event := testkube.Event{
			Type_: testkube.EventTypePtr(testkube.BECOME_TESTWORKFLOW_FAILED_EventType),
			TestWorkflowExecution: &testkube.TestWorkflowExecution{
				Workflow: &testkube.TestWorkflow{
					Name: "test-workflow",
				},
			},
		}

		// when
		became, err := l.hasBecomeState(event)

		// then
		assert.NoError(t, err)
		assert.False(t, became, "should return false when state hasn't changed (failed to failed)")
	})

	t.Run("should return false when previous state is queued (not a valid transition)", func(t *testing.T) {
		// given
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		mockRepo := testworkflow.NewMockRepository(mockCtrl)
		mockRepo.EXPECT().
			GetPreviousFinishedState(gomock.Any(), "test-workflow", gomock.Any()).
			Return(testkube.QUEUED_TestWorkflowStatus, nil)

		mockWebhookRepository := cloudwebhook.NewMockWebhookRepository(mockCtrl)
		l := NewWebhookListener(
			"test-listener",
			"http://example.com",
			"",
			[]testkube.EventType{testkube.BECOME_TESTWORKFLOW_FAILED_EventType},
			"",
			"",
			nil,
			false,
			nil,
			nil,
			listenerWithTestWorkflowResultsRepository(mockRepo),
			listenerWithMetrics(v1.NewMetrics()),
			listenerWithWebhookResultsRepository(mockWebhookRepository),
		)

		event := testkube.Event{
			Type_: testkube.EventTypePtr(testkube.BECOME_TESTWORKFLOW_FAILED_EventType),
			TestWorkflowExecution: &testkube.TestWorkflowExecution{
				Workflow: &testkube.TestWorkflow{
					Name: "test-workflow",
				},
			},
		}

		// when
		became, err := l.hasBecomeState(event)

		// then
		assert.NoError(t, err)
		assert.False(t, became, "should return false when transitioning from queued to failed (not a 'become' event)")
	})
}
