package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	nethttp "net/http"
	"net/url"
	"strings"

	"github.com/kubeshop/testkube/pkg/http"
)

type ListResponse[T All] struct {
	Elements []T `json:"elements"`
}

type All interface {
	Organization | Environment | Agent | AgentInput
}

type RESTClient[I All, O All] struct {
	BaseUrl string
	Path    string
	Client  http.HttpClient
	Token   string
}

func setBearerAuth(req *nethttp.Request, token string) {
	req.Header.Set("Authorization", "Bearer "+token)
}

// StatusError is returned for any response the Control Plane rejects. It keeps
// the status code so callers can tell apart the reasons a lookup fails - a 404
// for an id that does not exist, a 401 or 403 for a credential that is not
// accepted - which the response body alone does not reveal, as it is often
// empty.
type StatusError struct {
	URL        string
	StatusCode int
	Body       string
}

func (e *StatusError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("%s returned HTTP %d", e.URL, e.StatusCode)
	}
	return fmt.Sprintf("%s returned HTTP %d: %s", e.URL, e.StatusCode, e.Body)
}

// newStatusError drains the error response, capping it so a control plane that
// answers with a full HTML page does not dump it all into the terminal.
func newStatusError(url string, resp *nethttp.Response) error {
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorResponseBytes))
	if err != nil {
		return fmt.Errorf("%s returned HTTP %d, and the response could not be read: %s", url, resp.StatusCode, err)
	}
	return &StatusError{URL: url, StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(body))}
}

func (c RESTClient[I, O]) List() ([]O, error) {
	path := c.Path
	r, err := nethttp.NewRequestWithContext(context.Background(), nethttp.MethodGet, c.BaseUrl+path, nil)
	if err != nil {
		return nil, err
	}
	r.Header.Add("Authorization", "Bearer "+c.Token)
	resp, err := c.Client.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, newStatusError(c.BaseUrl+path, resp)
	}

	var orgsResponse ListResponse[O]
	err = json.NewDecoder(resp.Body).Decode(&orgsResponse)
	return orgsResponse.Elements, err
}

func (c RESTClient[I, O]) ListWithQuery(query map[string]string) ([]O, error) {
	path := c.Path
	qs := ""
	if len(query) > 0 {
		q := make([]string, len(query))
		for k, v := range query {
			q = append(q, fmt.Sprintf("%s=%s", url.QueryEscape(k), url.QueryEscape(v)))
		}
		qs = "?" + strings.Join(q, "&")
	}
	r, err := nethttp.NewRequestWithContext(context.Background(), nethttp.MethodGet, c.BaseUrl+path+qs, nil)
	if err != nil {
		return nil, err
	}
	r.Header.Add("Authorization", "Bearer "+c.Token)
	resp, err := c.Client.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, newStatusError(c.BaseUrl+path+qs, resp)
	}

	var orgsResponse ListResponse[O]
	err = json.NewDecoder(resp.Body).Decode(&orgsResponse)
	return orgsResponse.Elements, err
}

func (c RESTClient[I, O]) Get(id string) (e O, err error) {
	path := c.BaseUrl + c.Path + "/" + id
	req, err := nethttp.NewRequestWithContext(context.Background(), nethttp.MethodGet, path, nil)
	if err != nil {
		return e, err
	}
	req.Header.Add("Authorization", "Bearer "+c.Token)
	resp, err := c.Client.Do(req)
	if err != nil {
		return e, err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		return e, newStatusError(path, resp)
	}

	err = json.NewDecoder(resp.Body).Decode(&e)
	return
}

func (c RESTClient[I, O]) Create(entity I, overridePath ...string) (e O, err error) {
	d, err := json.Marshal(entity)
	if err != nil {
		return e, err
	}

	path := c.Path
	if len(overridePath) == 1 {
		path = overridePath[0]
	}

	r, err := nethttp.NewRequestWithContext(context.Background(), nethttp.MethodPost, c.BaseUrl+path, bytes.NewBuffer(d))
	if err != nil {
		return e, err
	}
	r.Header.Add("Content-type", "application/json")
	r.Header.Add("Authorization", "Bearer "+c.Token)

	resp, err := c.Client.Do(r)
	if err != nil {
		return e, err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		d, err := io.ReadAll(resp.Body)
		if err != nil {
			return e, fmt.Errorf("error creating %s: can't read response: %s", c.Path, err)
		}
		return e, fmt.Errorf("error creating %s: %s", c.Path, d)
	}

	err = json.NewDecoder(resp.Body).Decode(&e)
	if err != nil {
		return e, fmt.Errorf("error decoding response: %s", err)
	}

	return e, nil
}

func (c RESTClient[I, O]) Patch(id string, entity I, overridePath ...string) (err error) {
	d, err := json.Marshal(entity)
	if err != nil {
		return err
	}

	path := c.Path
	if len(overridePath) == 1 {
		path = overridePath[0]
	}

	r, err := nethttp.NewRequestWithContext(context.Background(), nethttp.MethodPatch, c.BaseUrl+path+"/"+id, bytes.NewBuffer(d))
	if err != nil {
		return err
	}
	r.Header.Add("Content-type", "application/json")
	r.Header.Add("Authorization", "Bearer "+c.Token)

	resp, err := c.Client.Do(r)
	if err != nil {
		return err
	}

	if resp.StatusCode > 299 {
		d, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("error updating %s: can't read response: %s", c.Path, err)
		}
		return fmt.Errorf("error updating %s: %s", c.Path, d)
	}

	return nil
}

func (c RESTClient[I, O]) Delete(id string, overridePath ...string) (err error) {
	path := c.Path + "/" + id
	if len(overridePath) == 1 {
		path = overridePath[0]
	}

	r, err := nethttp.NewRequestWithContext(context.Background(), nethttp.MethodDelete, c.BaseUrl+path, nil)
	if err != nil {
		return err
	}
	r.Header.Add("Content-type", "application/json")
	r.Header.Add("Authorization", "Bearer "+c.Token)

	resp, err := c.Client.Do(r)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		d, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("error deleting %s: can't read response: %s", c.Path, err)
		}
		return fmt.Errorf("error creating %s: %s", c.Path, d)
	}

	return nil
}
