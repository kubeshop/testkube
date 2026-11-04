// Package client talks to the Control Plane's generic, name-keyed endpoints
// for Control Plane resources (see the declarative package for the wire
// format).
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/kubeshop/testkube/pkg/controlplane/declarative"
)

// apiTokenPrefix marks a Testkube API token, as opposed to a user session.
const apiTokenPrefix = "tkcapi_"

// maxErrorBodyBytes bounds how much of an error response is read.
const maxErrorBodyBytes = 64 << 10

var (
	// ErrNotFound is returned when the named resource does not exist.
	ErrNotFound = errors.New("not found")
	// ErrUnsupported is returned when the Control Plane does not serve the
	// generic resource endpoints for a kind, as one that predates them does.
	ErrUnsupported = errors.New("the Control Plane does not support this resource kind; it may need upgrading")
	// ErrConflict is returned when the Control Plane refuses a change, such
	// as one to a resource a GitOps agent owns.
	ErrConflict = errors.New("conflict")
	// ErrInvalid is returned when the Control Plane rejects the object, for
	// example because it references something that does not exist.
	ErrInvalid = errors.New("invalid")
	// ErrRequiresUserSession is returned for an organization-scoped kind when
	// the credential is an API token, which those endpoints refuse.
	ErrRequiresUserSession = errors.New("organization-scoped resources need a user session; API tokens are environment-scoped - log in with `testkube login`")
)

// Client reads and writes Control Plane resources.
type Client interface {
	// Get returns the named resource, with its status.
	Get(ctx context.Context, kind declarative.Kind, name string) (declarative.Object, error)
	// List returns every resource of the kind.
	List(ctx context.Context, kind declarative.Kind) ([]declarative.Object, error)
	// Apply creates or replaces a resource, and returns it as stored.
	Apply(ctx context.Context, obj declarative.Object) (declarative.Object, error)
	// Delete removes the named resource.
	Delete(ctx context.Context, kind declarative.Kind, name string) error
}

// Doer sends HTTP requests.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Config locates the Control Plane and the organization and environment to
// address.
type Config struct {
	URL            string
	Token          string
	OrganizationID string
	EnvironmentID  string
	HTTPClient     Doer
}

// New returns a client for the Control Plane described by cfg.
func New(cfg Config) Client {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}
	cfg.URL = strings.TrimSuffix(cfg.URL, "/")
	return &httpClient{cfg: cfg}
}

type httpClient struct {
	cfg Config
}

// StatusError is a response the Control Plane rejected. It wraps one of the
// sentinel errors when the status has a meaning of its own.
type StatusError struct {
	StatusCode int
	Detail     string
	// Errors lists every problem the Control Plane found, such as each
	// reference a 422 could not resolve.
	Errors []string
	kind   error
}

func (e *StatusError) Error() string {
	var b strings.Builder
	if e.kind != nil {
		b.WriteString(e.kind.Error())
	} else {
		fmt.Fprintf(&b, "Control Plane returned HTTP %d", e.StatusCode)
	}
	if e.Detail != "" {
		b.WriteString(": ")
		b.WriteString(e.Detail)
	}
	for _, msg := range e.Errors {
		b.WriteString("\n  - ")
		b.WriteString(msg)
	}
	return b.String()
}

func (e *StatusError) Unwrap() error {
	return e.kind
}

func (c *httpClient) Get(ctx context.Context, kind declarative.Kind, name string) (declarative.Object, error) {
	body, err := c.do(ctx, http.MethodGet, kind, name, nil)
	if err != nil {
		return nil, err
	}
	return decodeObject(body, kind)
}

func (c *httpClient) List(ctx context.Context, kind declarative.Kind) ([]declarative.Object, error) {
	body, err := c.do(ctx, http.MethodGet, kind, "", nil)
	if err != nil {
		// A collection always exists, so a 404 means the endpoint does not.
		if errors.Is(err, ErrNotFound) {
			return nil, ErrUnsupported
		}
		return nil, err
	}
	var list struct {
		Elements []json.RawMessage `json:"elements"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("decoding %s list: %w", kind.Resource, err)
	}
	out := make([]declarative.Object, 0, len(list.Elements))
	for _, item := range list.Elements {
		obj, err := decodeObject(item, kind)
		if err != nil {
			return nil, err
		}
		out = append(out, obj)
	}
	return out, nil
}

func (c *httpClient) Apply(ctx context.Context, obj declarative.Object) (declarative.Object, error) {
	kind, err := declarative.KindOf(obj)
	if err != nil {
		return nil, err
	}
	payload, err := declarative.ApplyPayload(obj)
	if err != nil {
		return nil, err
	}
	body, err := c.do(ctx, http.MethodPut, kind, obj.GetName(), payload)
	if err != nil {
		// PUT creates what does not exist, so a 404 or 405 means the
		// endpoint does not.
		var statusErr *StatusError
		if errors.Is(err, ErrNotFound) || (errors.As(err, &statusErr) && statusErr.StatusCode == http.StatusMethodNotAllowed) {
			return nil, ErrUnsupported
		}
		return nil, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return declarative.PrepareForApply(obj)
	}
	return decodeObject(body, kind)
}

func (c *httpClient) Delete(ctx context.Context, kind declarative.Kind, name string) error {
	_, err := c.do(ctx, http.MethodDelete, kind, name, nil)
	return err
}

func (c *httpClient) path(kind declarative.Kind, name string) (string, error) {
	if c.cfg.OrganizationID == "" {
		return "", errors.New("no organization is set in the current context; run `testkube login` or `testkube set context`")
	}
	path := "/organizations/" + url.PathEscape(c.cfg.OrganizationID)
	if kind.Scope == declarative.ScopeEnvironment {
		if c.cfg.EnvironmentID == "" {
			return "", errors.New("no environment is set in the current context; run `testkube set context --env-id <id>`")
		}
		path += "/environments/" + url.PathEscape(c.cfg.EnvironmentID)
	}
	path += "/resources/" + url.PathEscape(kind.Resource)
	if name != "" {
		path += "/" + url.PathEscape(name)
	}
	return path, nil
}

func (c *httpClient) do(ctx context.Context, method string, kind declarative.Kind, name string, payload []byte) ([]byte, error) {
	if kind.Scope == declarative.ScopeOrganization && strings.HasPrefix(c.cfg.Token, apiTokenPrefix) {
		return nil, ErrRequiresUserSession
	}
	path, err := c.path(kind, name)
	if err != nil {
		return nil, err
	}
	var reader io.Reader
	if payload != nil {
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.cfg.URL+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.Token)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, newStatusError(resp)
	}
	return io.ReadAll(resp.Body)
}

func newStatusError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
	e := &StatusError{StatusCode: resp.StatusCode}
	var problem struct {
		Title  string `json:"title"`
		Detail string `json:"detail"`
		Errors []any  `json:"errors"`
	}
	if json.Unmarshal(body, &problem) == nil {
		e.Detail = problem.Detail
		if e.Detail == "" {
			e.Detail = problem.Title
		}
		for _, item := range problem.Errors {
			e.Errors = append(e.Errors, describeProblem(item))
		}
	} else {
		e.Detail = strings.TrimSpace(string(body))
	}
	switch resp.StatusCode {
	case http.StatusNotFound:
		e.kind = ErrNotFound
	case http.StatusConflict:
		e.kind = ErrConflict
	case http.StatusUnprocessableEntity, http.StatusBadRequest:
		e.kind = ErrInvalid
	}
	return e
}

// describeProblem renders one entry of a problem's errors, which may be a
// string or an object with a message (and optionally a field).
func describeProblem(item any) string {
	switch v := item.(type) {
	case string:
		return v
	case map[string]any:
		msg, _ := v["message"].(string)
		if msg == "" {
			msg, _ = v["detail"].(string)
		}
		if f, _ := v["field"].(string); f != "" {
			return f + ": " + msg
		}
		if msg != "" {
			return msg
		}
	}
	data, _ := json.Marshal(item)
	return string(data)
}

func decodeObject(data []byte, kind declarative.Kind) (declarative.Object, error) {
	obj := kind.New()
	if err := json.Unmarshal(data, obj); err != nil {
		return nil, fmt.Errorf("decoding %s: %w", kind.Kind, err)
	}
	// The response may leave out apiVersion and kind; the endpoint fixes them.
	obj.GetObjectKind().SetGroupVersionKind(kind.GroupVersionKind())
	return obj, nil
}
