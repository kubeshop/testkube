package webhook

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	v1 "github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	cloudwebhook "github.com/kubeshop/testkube/pkg/cloud/data/webhook"
)

// The subscriber's endpoint is briefly unhealthy (Sheng's scenario: transient
// infra hiccup). It returns 503 twice, then 200; the retry loop hides the two
// failures and reports success to the caller.
func TestWebhookListener_Notify_RetriesOn5xxThenSucceeds(t *testing.T) {
	var callCount int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(&callCount, 1)
		if n <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	svr := httptest.NewServer(handler)
	defer svr.Close()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	repo := cloudwebhook.NewMockWebhookRepository(mockCtrl)
	repo.EXPECT().CollectExecutionResult(gomock.Any(), gomock.Any(), "l1", "", http.StatusOK).AnyTimes()

	l := NewWebhookListener("l1", svr.URL, "", testEventTypes, "", "", nil, false, nil, nil,
		listenerWithMetrics(v1.NewMetrics()),
		listenerWithWebhookResultsRepository(repo))
	l.sendRetryBaseDelay = time.Millisecond

	r := l.Notify(testkube.Event{
		Type_:                 testkube.EventStartTestWorkflow,
		TestWorkflowExecution: exampleExecution(),
	})

	assert.Equal(t, "", r.Error())
	assert.Equal(t, int32(3), atomic.LoadInt32(&callCount))
}

// The subscriber stays down for the whole retry budget. The loop exhausts every
// attempt and only then reports failure, so the payload is not lost silently.
func TestWebhookListener_Notify_GivesUpAfter5xxRetryBudget(t *testing.T) {
	var callCount int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	svr := httptest.NewServer(handler)
	defer svr.Close()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	repo := cloudwebhook.NewMockWebhookRepository(mockCtrl)
	repo.EXPECT().CollectExecutionResult(gomock.Any(), gomock.Any(), "l1", gomock.Any(), http.StatusServiceUnavailable).AnyTimes()

	l := NewWebhookListener("l1", svr.URL, "", testEventTypes, "", "", nil, false, nil, nil,
		listenerWithMetrics(v1.NewMetrics()),
		listenerWithWebhookResultsRepository(repo))
	l.sendRetryBaseDelay = time.Millisecond

	r := l.Notify(testkube.Event{
		Type_:                 testkube.EventStartTestWorkflow,
		TestWorkflowExecution: exampleExecution(),
	})

	assert.NotEqual(t, "", r.Error())
	assert.Equal(t, int32(webhookSendRetryCount), atomic.LoadInt32(&callCount))
}

// 4xx is a client-side error the subscriber will not resolve by retrying (bad
// URL, unauthorized, wrong payload shape). Retrying would just spam the endpoint,
// so the loop exits after the first attempt.
func TestWebhookListener_Notify_DoesNotRetryOn4xx(t *testing.T) {
	var callCount int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusNotFound)
	})
	svr := httptest.NewServer(handler)
	defer svr.Close()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	repo := cloudwebhook.NewMockWebhookRepository(mockCtrl)
	repo.EXPECT().CollectExecutionResult(gomock.Any(), gomock.Any(), "l1", gomock.Any(), http.StatusNotFound).AnyTimes()

	l := NewWebhookListener("l1", svr.URL, "", testEventTypes, "", "", nil, false, nil, nil,
		listenerWithMetrics(v1.NewMetrics()),
		listenerWithWebhookResultsRepository(repo))
	l.sendRetryBaseDelay = time.Millisecond

	r := l.Notify(testkube.Event{
		Type_:                 testkube.EventStartTestWorkflow,
		TestWorkflowExecution: exampleExecution(),
	})

	assert.NotEqual(t, "", r.Error())
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount))
}

// 429 is treated as retryable: rate limiters usually clear within a short delay,
// and the alternative (drop the event) is what Sheng flagged as unacceptable.
func TestWebhookListener_Notify_RetriesOn429ThenSucceeds(t *testing.T) {
	var callCount int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if atomic.AddInt32(&callCount, 1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	svr := httptest.NewServer(handler)
	defer svr.Close()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	repo := cloudwebhook.NewMockWebhookRepository(mockCtrl)
	repo.EXPECT().CollectExecutionResult(gomock.Any(), gomock.Any(), "l1", "", http.StatusOK).AnyTimes()

	l := NewWebhookListener("l1", svr.URL, "", testEventTypes, "", "", nil, false, nil, nil,
		listenerWithMetrics(v1.NewMetrics()),
		listenerWithWebhookResultsRepository(repo))
	l.sendRetryBaseDelay = time.Millisecond

	r := l.Notify(testkube.Event{
		Type_:                 testkube.EventStartTestWorkflow,
		TestWorkflowExecution: exampleExecution(),
	})

	assert.Equal(t, "", r.Error())
	assert.Equal(t, int32(2), atomic.LoadInt32(&callCount))
}
