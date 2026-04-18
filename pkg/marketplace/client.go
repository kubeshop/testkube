package marketplace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	tkhttp "github.com/kubeshop/testkube/pkg/http"
)

// ErrWorkflowNotFound is returned by GetWorkflow when the requested entry is
// absent from the marketplace catalog. Callers should check with errors.Is
// so that failures from catalog fetching (network errors, HTTP 404s whose
// error message happens to contain the phrase "not found", etc.) are not
// misclassified as a missing workflow.
var ErrWorkflowNotFound = errors.New("workflow not found in marketplace catalog")

// DefaultBaseURL is the raw GitHub URL for the kubeshop/testkube-marketplace
// main branch. It is the same base used by the Dashboard in useCatalog.ts.
const DefaultBaseURL = "https://raw.githubusercontent.com/kubeshop/testkube-marketplace/main/"

// CatalogIndexPath is the repo-relative path of the catalog index file.
const CatalogIndexPath = "catalog-index.json"

// HTTPDoer matches *http.Client so tests can substitute a stub.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client fetches marketplace catalog entries, workflow YAML, and readme
// content. Construct via NewClient; zero values are not usable.
type Client struct {
	baseURL string
	http    HTTPDoer
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithBaseURL overrides the base URL used for fetches. The URL must end with
// a trailing slash; one is added if missing. Primarily intended for tests.
func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		if !strings.HasSuffix(url, "/") {
			url += "/"
		}
		c.baseURL = url
	}
}

// WithHTTPClient overrides the underlying HTTP client.
func WithHTTPClient(h HTTPDoer) ClientOption {
	return func(c *Client) {
		c.http = h
	}
}

// NewClient returns a marketplace client that talks to the public
// testkube-marketplace GitHub repo. Options can override the base URL and
// HTTP client, which is useful in tests.
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		baseURL: DefaultBaseURL,
		http:    tkhttp.NewClient(),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// ListWorkflows fetches and decodes catalog-index.json.
func (c *Client) ListWorkflows(ctx context.Context) ([]Workflow, error) {
	body, err := c.fetch(ctx, CatalogIndexPath)
	if err != nil {
		return nil, err
	}
	var workflows []Workflow
	if err := json.Unmarshal(body, &workflows); err != nil {
		return nil, fmt.Errorf("decoding catalog index: %w", err)
	}
	return workflows, nil
}

// GetWorkflow finds a catalog entry by its (unique) name. Returns an error
// if the catalog cannot be fetched or no matching entry exists.
func (c *Client) GetWorkflow(ctx context.Context, name string) (*Workflow, error) {
	workflows, err := c.ListWorkflows(ctx)
	if err != nil {
		return nil, err
	}
	for i := range workflows {
		if workflows[i].Name == name {
			return &workflows[i], nil
		}
	}
	return nil, fmt.Errorf("%w: %q", ErrWorkflowNotFound, name)
}

// GetWorkflowYAML downloads the raw YAML for a catalog entry's Path field.
func (c *Client) GetWorkflowYAML(ctx context.Context, w Workflow) ([]byte, error) {
	if w.Path == "" {
		return nil, fmt.Errorf("workflow %q has no path", w.Name)
	}
	return c.fetch(ctx, w.Path)
}

// GetReadme downloads the raw readme for a catalog entry. Returns an empty
// byte slice and nil error when the entry has no readme.
func (c *Client) GetReadme(ctx context.Context, w Workflow) ([]byte, error) {
	if w.Readme == "" {
		return nil, nil
	}
	return c.fetch(ctx, w.Readme)
}

// FetchURL downloads raw bytes from an arbitrary URL using the client's HTTP
// client. It is used by the create testworkflow --url flag so both the
// marketplace CLI and generic URL fetches share the same transport.
func (c *Client) FetchURL(ctx context.Context, url string) ([]byte, error) {
	return doGet(ctx, c.http, url)
}

func (c *Client) fetch(ctx context.Context, path string) ([]byte, error) {
	return doGet(ctx, c.http, c.baseURL+strings.TrimLeft(path, "/"))
}

func doGet(ctx context.Context, h HTTPDoer, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building request for %s: %w", url, err)
	}
	resp, err := h.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response from %s: %w", url, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet := strings.TrimSpace(string(body))
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		return nil, fmt.Errorf("fetching %s: status %d: %s", url, resp.StatusCode, snippet)
	}
	return body, nil
}
