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

func (c RESTClient[T]) Create(entity T) error {
	d, err := json.Marshal(entity)
	if err != nil {
		return err
	}
	r, err := nethttp.NewRequest("POST", c.BaseUrl+c.Path, bytes.NewBuffer(d))
	if err != nil {
		return err
	}
	r.Header.Add("Authorization", "Bearer "+c.Token)

	resp, err := c.Client.Do(r)
	if err != nil {
		return err
	}

	if resp.StatusCode > 299 {
		d, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error creating %s: %s", c.Path, d)
	}

	return nil
}
