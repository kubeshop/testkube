package client

import (
	"bytes"
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

func (c RESTClient[I, O]) List() ([]O, error) {
	path := c.Path
	r, err := nethttp.NewRequest(nethttp.MethodGet, c.BaseUrl+path, nil)
	r.Header.Add("Authorization", "Bearer "+c.Token)
	if err != nil {
		return nil, err
	}
	resp, err := c.Client.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		d, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error getting %s: can't read response: %s", c.Path, err)
		}
		return nil, fmt.Errorf("error getting %s: %s", path, d)
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
	r, err := nethttp.NewRequest(nethttp.MethodGet, c.BaseUrl+path+qs, nil)
	r.Header.Add("Authorization", "Bearer "+c.Token)
	if err != nil {
		return nil, err
	}
	resp, err := c.Client.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		d, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error getting %s: can't read response: %s", c.Path, err)
		}
		return nil, fmt.Errorf("error getting %s: %s", path, d)
	}

	var orgsResponse ListResponse[O]
	err = json.NewDecoder(resp.Body).Decode(&orgsResponse)
	return orgsResponse.Elements, err
}

func (c RESTClient[I, O]) Get(id string) (e O, err error) {
	path := c.BaseUrl + c.Path + "/" + id
	req, err := nethttp.NewRequest(nethttp.MethodGet, path, nil)
	req.Header.Add("Authorization", "Bearer "+c.Token)
	if err != nil {
		return e, err
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return e, err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		d, err := io.ReadAll(resp.Body)
		if err != nil {
			return e, fmt.Errorf("error getting %s: can't read response: %s", c.Path, err)
		}
		return e, fmt.Errorf("error getting %s: %s", path, d)
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

	r, err := nethttp.NewRequest(nethttp.MethodPost, c.BaseUrl+path, bytes.NewBuffer(d))
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

	r, err := nethttp.NewRequest(nethttp.MethodPatch, c.BaseUrl+path+"/"+id, bytes.NewBuffer(d))
	if err != nil {
		return err
	}
	r.Header.Add("Content-type", "application/json")
	r.Header.Add("Authorization", "Bearer "+c.Token)

	resp, err := c.Client.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

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

	r, err := nethttp.NewRequest(nethttp.MethodDelete, c.BaseUrl+path, nil)
	if err != nil {
		return err
	}
	r.Header.Add("Content-type", "application/json")
	r.Header.Add("Authorization", "Bearer "+c.Token)

	resp, err := c.Client.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		d, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("error deleting %s: can't read response: %s", c.Path, err)
		}
		return fmt.Errorf("error creating %s: %s", c.Path, d)
	}

	return nil
}
