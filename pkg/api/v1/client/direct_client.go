package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/logs/events"
	"github.com/kubeshop/testkube/pkg/oauth"
	"github.com/kubeshop/testkube/pkg/problem"
)

type transport struct {
	headers map[string]string
	base    http.RoundTripper
}

// RoundTrip is a method to adjust http request
func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range t.headers {
		req.Header.Add(k, v)
	}

	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}

	return base.RoundTrip(req)
}

func ConfigureClient(client *http.Client, token *oauth2.Token, cloudApiKey string, headers map[string]string) {
	hs := headers
	if hs == nil {
		hs = make(map[string]string)
	}

	if token != nil {
		hs["Authorization"] = oauth.AuthorizationPrefix + " " + token.AccessToken
	}

	if cloudApiKey != "" {
		hs["Authorization"] = "Bearer " + cloudApiKey
	}

	client.Transport = &transport{headers: hs}
}

// NewDirectClient returns new direct client
func NewDirectClient[A All](httpClient *http.Client, apiURI, apiPathPrefix string) DirectClient[A] {
	if apiPathPrefix == "" {
		apiPathPrefix = "/" + Version
	}

	return DirectClient[A]{
		client:        httpClient,
		sseClient:     httpClient,
		apiURI:        apiURI,
		apiPathPrefix: apiPathPrefix,
	}
}

// DirectClient implements direct client
type DirectClient[A All] struct {
	client        *http.Client
	sseClient     *http.Client
	apiURI        string
	apiPathPrefix string
}

// baseExecute is base execute method
func (t DirectClient[A]) baseExec(method, uri, resource string, body []byte, params map[string]string) (resp *http.Response, err error) {
	var buffer io.Reader
	if body != nil {
		buffer = bytes.NewBuffer(body)
	}

	req, err := http.NewRequest(method, uri, buffer)
	if err != nil {
		return resp, err
	}

	req.Header.Set("Content-Type", "application/json")
	q := req.URL.Query()
	for key, value := range params {
		if value != "" {
			q.Add(key, value)
		}
	}
	req.URL.RawQuery = q.Encode()

	resp, err = t.client.Do(req)
	if err != nil {
		return resp, err
	}

	if err = t.responseError(resp); err != nil {
		return resp, fmt.Errorf("api/%s-%s returned error: %w", method, resource, err)
	}

	return resp, nil
}

func (t DirectClient[A]) WithSSEClient(client *http.Client) DirectClient[A] {
	t.sseClient = client
	return t
}

// Execute is a method to make an api call for a single object
func (t DirectClient[A]) Execute(method, uri string, body []byte, params map[string]string) (result A, err error) {
	resp, err := t.baseExec(method, uri, fmt.Sprintf("%T", result), body, params)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	return t.getFromResponse(resp)
}

// ExecuteMultiple is a method to make an api call for multiple objects
func (t DirectClient[A]) ExecuteMultiple(method, uri string, body []byte, params map[string]string) (result []A, err error) {
	resp, err := t.baseExec(method, uri, fmt.Sprintf("%T", result), body, params)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	return t.getFromResponses(resp)
}

// Delete is a method to make delete api call
func (t DirectClient[A]) Delete(uri, selector string, isContentExpected bool) error {
	return t.ExecuteMethod(http.MethodDelete, uri, selector, isContentExpected)
}

func (t DirectClient[A]) ExecuteMethod(method, uri string, selector string, isContentExpected bool) error {
	resp, err := t.baseExec(method, uri, uri, nil, map[string]string{"selector": selector})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if isContentExpected && resp.StatusCode != http.StatusNoContent {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return fmt.Errorf("request returned error: %s", respBody)
	}

	return nil
}

// GetURI returns uri for api method
func (t DirectClient[A]) GetURI(pathTemplate string, params ...interface{}) string {
	path := fmt.Sprintf(pathTemplate, params...)
	return fmt.Sprintf("%s%s%s", t.apiURI, t.apiPathPrefix, path)
}

// GetLogs returns logs stream from job pods, based on job pods logs
func (t DirectClient[A]) GetLogs(uri string, logs chan output.Output) error {
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "text/event-stream")
	resp, err := t.sseClient.Do(req)
	if err != nil {
		return err
	}

	go func() {
		defer close(logs)
		defer resp.Body.Close()

		StreamToLogsChannel(resp.Body, logs)
	}()

	return nil
}

// GetLogsV2 returns logs stream version 2 from log server, based on job pods logs
func (t DirectClient[A]) GetLogsV2(uri string, logs chan events.Log) error {
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "text/event-stream")
	resp, err := t.sseClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New("error getting logs, invalid status code: " + resp.Status)
	}

	go func() {
		defer close(logs)
		defer resp.Body.Close()

		StreamToLogsChannelV2(resp.Body, logs)
	}()

	return nil
}

// GetTestWorkflowExecutionNotifications returns logs stream from job pods, based on job pods logs
func (t DirectClient[A]) GetTestWorkflowExecutionNotifications(uri string, notifications chan testkube.TestWorkflowExecutionNotification) error {
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "text/event-stream")
	resp, err := t.sseClient.Do(req)
	if err != nil {
		return err
	}

	go func() {
		defer close(notifications)
		defer resp.Body.Close()

		StreamToTestWorkflowExecutionNotificationsChannel(resp.Body, notifications)
	}()

	return nil
}

// GetFile returns file artifact
func (t DirectClient[A]) GetFile(uri, fileName, destination string, params map[string][]string) (name string, err error) {
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return "", err
	}

	q := req.URL.Query()
	for key, values := range params {
		for _, value := range values {
			if value != "" {
				q.Add(key, value)
			}
		}
	}
	req.URL.RawQuery = q.Encode()

	resp, err := t.client.Do(req)
	if err != nil {
		return name, err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		return name, fmt.Errorf("error: %d", resp.StatusCode)
	}

	target := filepath.Join(destination, fileName)
	dir := filepath.Dir(target)
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		if err = os.MkdirAll(dir, os.ModePerm); err != nil {
			return name, err
		}
	} else if err != nil {
		return name, err
	}

	f, err := os.Create(target)
	if err != nil {
		return name, err
	}

	if _, err = io.Copy(f, resp.Body); err != nil {
		return name, err
	}

	if err = t.responseError(resp); err != nil {
		return name, fmt.Errorf("api/download-file returned error: %w", err)
	}

	return f.Name(), nil
}

func (t DirectClient[A]) getFromResponse(resp *http.Response) (result A, err error) {
	err = json.NewDecoder(resp.Body).Decode(&result)
	return
}

func (t DirectClient[A]) getFromResponses(resp *http.Response) (result []A, err error) {
	err = json.NewDecoder(resp.Body).Decode(&result)
	return
}

// responseError tries to lookup if response is of Problem type
func (t DirectClient[A]) responseError(resp *http.Response) error {
	if resp.StatusCode >= 400 {
		var pr problem.Problem

		bytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("can't get problem from api response: can't read response body %w", err)
		}

		err = json.Unmarshal(bytes, &pr)
		if err != nil {
			return fmt.Errorf("can't get problem from api response: %w, output: %s", err, string(bytes))
		}

		return fmt.Errorf("problem: %+v", pr.Detail)
	}

	return nil
}
