package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	nethttp "net/http"

	"github.com/kubeshop/testkube/pkg/http"
)

type ListResponse[T All] struct {
	Elements []T `json:"elements"`
}

type All interface {
	Organization | Environment
}

type RESTClient[T All] struct {
	BaseUrl string
	Path    string
	Client  http.HttpClient
	Token   string
}

func (c RESTClient[T]) List() ([]T, error) {
	r, err := nethttp.NewRequest("GET", c.BaseUrl+c.Path, nil)
	r.Header.Add("Authorization", "Bearer "+c.Token)
	if err != nil {
		return nil, err
	}
	resp, err := c.Client.Do(r)
	if err != nil {
		return nil, err
	}

	var orgsResponse ListResponse[T]
	err = json.NewDecoder(resp.Body).Decode(&orgsResponse)
	return orgsResponse.Elements, err
}

func (c RESTClient[T]) Get(id string) (e T, err error) {
	path := c.BaseUrl + c.Path + "/" + id
	req, err := nethttp.NewRequest("GET", path, nil)
	req.Header.Add("Authorization", "Bearer "+c.Token)
	if err != nil {
		return e, err
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return e, err
	}

	if resp.StatusCode > 299 {
		d, err := io.ReadAll(resp.Body)
		if err != nil {
			return e, fmt.Errorf("error creating %s: can't read response: %s", c.Path, err)
		}
		return e, fmt.Errorf("error getting %s: %s", path, d)
	}

	err = json.NewDecoder(resp.Body).Decode(&e)
	return
}

func (c RESTClient[T]) Create(entity T, overridePath ...string) (e T, err error) {
	d, err := json.Marshal(entity)
	if err != nil {
		return e, err
	}

	path := c.Path
	if len(overridePath) == 1 {
		path = overridePath[0]
	}

	r, err := nethttp.NewRequest("POST", c.BaseUrl+path, bytes.NewBuffer(d))
	if err != nil {
		return e, err
	}
	r.Header.Add("Content-type", "application/json")
	r.Header.Add("Authorization", "Bearer "+c.Token)

	resp, err := c.Client.Do(r)
	if err != nil {
		return e, err
	}

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
