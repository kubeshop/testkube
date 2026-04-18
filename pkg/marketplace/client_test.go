package marketplace

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_ListWorkflows(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/catalog-index.json" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`[
			{"path":"workflows/databases/redis/redis-connectivity.yaml","name":"redis-connectivity","category":"databases","component":"redis","displayName":"Redis","description":"Check redis","icon":"x","readme":"r.md"},
			{"path":"workflows/networking/ingress/ingress-health-check.yaml","name":"ingress-health-check","category":"networking","component":"ingress","displayName":"Ingress","description":"Check ingress"}
		]`))
	}))
	t.Cleanup(srv.Close)

	c := NewClient(WithBaseURL(srv.URL))
	workflows, err := c.ListWorkflows(context.Background())
	if err != nil {
		t.Fatalf("ListWorkflows: %v", err)
	}
	if len(workflows) != 2 {
		t.Fatalf("expected 2 workflows, got %d", len(workflows))
	}
	if workflows[0].Name != "redis-connectivity" || workflows[0].Category != "databases" {
		t.Errorf("unexpected first workflow: %+v", workflows[0])
	}
	if workflows[1].Name != "ingress-health-check" || workflows[1].Readme != "" {
		t.Errorf("unexpected second workflow: %+v", workflows[1])
	}
}

func TestClient_GetWorkflow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"path":"p.yaml","name":"redis-connectivity","category":"databases","component":"redis","displayName":"Redis","description":"x"}]`))
	}))
	t.Cleanup(srv.Close)

	c := NewClient(WithBaseURL(srv.URL))

	got, err := c.GetWorkflow(context.Background(), "redis-connectivity")
	if err != nil {
		t.Fatalf("GetWorkflow: %v", err)
	}
	if got.Path != "p.yaml" {
		t.Errorf("expected path p.yaml, got %q", got.Path)
	}

	_, err = c.GetWorkflow(context.Background(), "nope")
	if err == nil {
		t.Fatalf("expected error for missing workflow")
	}
	if !errors.Is(err, ErrWorkflowNotFound) {
		t.Errorf("expected ErrWorkflowNotFound sentinel, got %v", err)
	}
}

// TestClient_GetWorkflow_CatalogFetchErrorIsNotMisclassified guards against a
// regression where a 404 from the catalog endpoint itself gets interpreted as
// ErrWorkflowNotFound by string matching. Callers rely on errors.Is to
// distinguish between "catalog unreachable" and "workflow missing".
func TestClient_GetWorkflow_CatalogFetchErrorIsNotMisclassified(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	}))
	t.Cleanup(srv.Close)

	c := NewClient(WithBaseURL(srv.URL))
	_, err := c.GetWorkflow(context.Background(), "redis-connectivity")
	if err == nil {
		t.Fatal("expected error when catalog fetch returns 404")
	}
	if errors.Is(err, ErrWorkflowNotFound) {
		t.Errorf("catalog fetch 404 must not be reported as ErrWorkflowNotFound, got %v", err)
	}
}

func TestClient_GetWorkflowYAMLAndReadme(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/workflows/a.yaml":
			_, _ = w.Write([]byte("yaml: body"))
		case "/workflows/a.md":
			_, _ = w.Write([]byte("# readme"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	c := NewClient(WithBaseURL(srv.URL))
	wf := Workflow{Path: "workflows/a.yaml", Readme: "workflows/a.md", Name: "a"}

	body, err := c.GetWorkflowYAML(context.Background(), wf)
	if err != nil {
		t.Fatalf("GetWorkflowYAML: %v", err)
	}
	if string(body) != "yaml: body" {
		t.Errorf("unexpected yaml body: %q", body)
	}

	readme, err := c.GetReadme(context.Background(), wf)
	if err != nil {
		t.Fatalf("GetReadme: %v", err)
	}
	if string(readme) != "# readme" {
		t.Errorf("unexpected readme body: %q", readme)
	}

	readme2, err := c.GetReadme(context.Background(), Workflow{})
	if err != nil {
		t.Fatalf("GetReadme empty: %v", err)
	}
	if readme2 != nil {
		t.Errorf("expected nil readme, got %q", readme2)
	}
}

func TestClient_FetchURLHandlesErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	}))
	t.Cleanup(srv.Close)

	c := NewClient(WithBaseURL(srv.URL))
	_, err := c.FetchURL(context.Background(), srv.URL+"/missing")
	if err == nil {
		t.Fatal("expected error on 404")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected error to mention status 404, got %v", err)
	}
}
