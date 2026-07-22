package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func newTestWorkflowClientForServer(server *httptest.Server) TestWorkflowClient {
	return NewTestWorkflowClient(
		NewDirectClient[testkube.TestWorkflow](server.Client(), server.URL, ""),
		NewDirectClient[testkube.TestWorkflowWithExecution](server.Client(), server.URL, ""),
		NewDirectClient[testkube.TestWorkflowExecution](server.Client(), server.URL, ""),
		NewDirectClient[testkube.TestWorkflowExecutionsResult](server.Client(), server.URL, ""),
		NewDirectClient[testkube.Artifact](server.Client(), server.URL, ""),
	)
}

func testWorkflowExecutionItems(names ...string) testkube.TestWorkflowWithExecutions {
	items := make(testkube.TestWorkflowWithExecutions, 0, len(names))
	for _, name := range names {
		items = append(items, testkube.TestWorkflowWithExecution{
			Workflow: &testkube.TestWorkflow{Name: name},
		})
	}
	return items
}

func numberedNames(prefix string, n int) []string {
	names := make([]string, n)
	for i := range names {
		names[i] = fmt.Sprintf("%s-%d", prefix, i)
	}
	return names
}

func TestListTestWorkflowWithExecutions(t *testing.T) {
	tests := []struct {
		name         string
		limit        int
		respond      func(page string) []string
		wantLen      int
		wantRequests int
		check        func(t *testing.T, workflows testkube.TestWorkflowWithExecutions, requests []*http.Request)
	}{
		{
			name:  "explicit limit sends pageSize and no page",
			limit: 25,
			respond: func(string) []string {
				return []string{"a", "b"}
			},
			wantLen:      2,
			wantRequests: 1,
			check: func(t *testing.T, _ testkube.TestWorkflowWithExecutions, requests []*http.Request) {
				q := requests[0].URL.Query()
				assert.Equal(t, "25", q.Get("pageSize"))
				assert.False(t, q.Has("page"))
			},
		},
		{
			name:  "fetches all pages until a short page",
			limit: 0,
			respond: func(page string) []string {
				switch page {
				case "0":
					return numberedNames("p0", 100)
				case "1":
					return numberedNames("p1", 100)
				default:
					return numberedNames("p2", 30)
				}
			},
			wantLen:      230,
			wantRequests: 3,
			check: func(t *testing.T, _ testkube.TestWorkflowWithExecutions, requests []*http.Request) {
				for i, req := range requests {
					q := req.URL.Query()
					assert.Equal(t, "100", q.Get("pageSize"))
					assert.Equal(t, fmt.Sprintf("%d", i), q.Get("page"))
				}
			},
		},
		{
			name:  "deduplicates items that shift between pages",
			limit: 0,
			respond: func(page string) []string {
				if page == "0" {
					return append(numberedNames("p0", 99), "shared")
				}
				return append([]string{"shared"}, numberedNames("p1", 9)...)
			},
			wantLen:      109,
			wantRequests: 2,
			check: func(t *testing.T, workflows testkube.TestWorkflowWithExecutions, _ []*http.Request) {
				count := 0
				for _, w := range workflows {
					if w.Workflow != nil && w.Workflow.Name == "shared" {
						count++
					}
				}
				assert.Equal(t, 1, count)
			},
		},
		{
			// The standalone server returns the full result set for pages past
			// the end, so an exact multiple of the page size must not loop
			// forever. Unnamed items dedupe to one like any other name.
			name:  "stops when a page adds nothing new",
			limit: 0,
			respond: func(string) []string {
				return append([]string{"", ""}, numberedNames("wf", 98)...)
			},
			wantLen:      99,
			wantRequests: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mu sync.Mutex
			var requests []*http.Request
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				requests = append(requests, r)
				mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(testWorkflowExecutionItems(tt.respond(r.URL.Query().Get("page"))...))
			}))
			t.Cleanup(server.Close)

			client := newTestWorkflowClientForServer(server)

			workflows, err := client.ListTestWorkflowWithExecutions("", tt.limit)
			require.NoError(t, err)
			assert.Len(t, workflows, tt.wantLen)
			assert.Len(t, requests, tt.wantRequests)
			if tt.check != nil {
				tt.check(t, workflows, requests)
			}
		})
	}
}
