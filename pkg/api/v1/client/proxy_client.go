package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/problem"
)

// GetClientSet configures Kube client set, can override host with local proxy
func GetClientSet(overrideHost string) (clientset kubernetes.Interface, err error) {
	clcfg, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	if err != nil {
		return clientset, err
	}

	restcfg, err := clientcmd.NewNonInteractiveClientConfig(
		*clcfg, "", &clientcmd.ConfigOverrides{}, nil).ClientConfig()
	if err != nil {
		return clientset, err
	}

	// override host is needed to override kubeconfig kubernetes proxy host name
	// to local proxy passed to API server run local proxy first by `make api-proxy`
	if overrideHost != "" {
		restcfg.Host = overrideHost
	}

	return kubernetes.NewForConfig(restcfg)
}

// NewProxyClient returns new proxy client
func NewProxyClient[A All](client kubernetes.Interface, config APIConfig) ProxyClient[A] {
	return ProxyClient[A]{
		client: client,
		config: config,
	}
}

// ProxyClient implements proxy client
type ProxyClient[A All] struct {
	client kubernetes.Interface
	config APIConfig
}

// baseExecute is base execute method
func (t ProxyClient[A]) baseExec(method, uri, resource string, body []byte, params map[string]string) (resp rest.Result, err error) {
	req := t.getProxy(method).
		Suffix(uri)
	if body != nil {
		req.Body(body)
	}

	for key, value := range params {
		if value != "" {
			req.Param(key, value)
		}
	}

	resp = req.Do(context.Background())

	if err = t.responseError(resp); err != nil {
		return resp, fmt.Errorf("api/%s-%s returned error: %w", method, resource, err)
	}

	return resp, nil
}

// Execute is a method to make an api call for a single object
func (t ProxyClient[A]) Execute(method, uri string, body []byte, params map[string]string) (result A, err error) {
	resp, err := t.baseExec(method, uri, fmt.Sprintf("%T", result), body, params)
	if err != nil {
		return result, err
	}

	return t.getFromResponse(resp)
}

// ExecuteMultiple is a method to make an api call for multiple objects
func (t ProxyClient[A]) ExecuteMultiple(method, uri string, body []byte, params map[string]string) (result []A, err error) {
	resp, err := t.baseExec(method, uri, fmt.Sprintf("%T", result), body, params)
	if err != nil {
		return result, err
	}

	return t.getFromResponses(resp)
}

// Delete is a method to make delete api call
func (t ProxyClient[A]) Delete(uri, selector string, isContentExpected bool) error {
	return t.ExecuteMethod(http.MethodDelete, uri, selector, isContentExpected)
}

func (t ProxyClient[A]) ExecuteMethod(method, uri string, selector string, isContentExpected bool) error {
	resp, err := t.baseExec(method, uri, uri, nil, map[string]string{"selector": selector})
	if err != nil {
		return err
	}

	if isContentExpected {
		var code int
		resp.StatusCode(&code)
		if code != http.StatusNoContent {
			respBody, err := resp.Raw()
			if err != nil {
				return err
			}
			return fmt.Errorf("request returned error: %s", respBody)
		}
	}

	return nil
}

// GetURI returns uri for api method
func (t ProxyClient[A]) GetURI(pathTemplate string, params ...interface{}) string {
	path := fmt.Sprintf(pathTemplate, params...)
	return fmt.Sprintf("%s%s", Version, path)
}

// GetLogs returns logs stream from job pods, based on job pods logs
func (t ProxyClient[A]) GetLogs(uri string, logs chan output.Output) error {
	resp, err := t.getProxy(http.MethodGet).
		Suffix(uri).
		SetHeader("Accept", "text/event-stream").
		Stream(context.Background())
	if err != nil {
		return err
	}

	go func() {
		defer close(logs)
		defer resp.Close()

		StreamToLogsChannel(resp, logs)
	}()

	return nil
}

// GetFile returns file artifact
func (t ProxyClient[A]) GetFile(uri, fileName, destination string) (name string, err error) {
	req, err := t.getProxy(http.MethodGet).
		Suffix(uri).
		SetHeader("Accept", "text/event-stream").
		Stream(context.Background())
	if err != nil {
		return name, err
	}
	defer req.Close()

	f, err := os.Create(filepath.Join(destination, filepath.Base(fileName)))
	if err != nil {
		return name, err
	}

	if _, err = f.ReadFrom(req); err != nil {
		return name, err
	}

	defer f.Close()
	return f.Name(), err
}

func (t ProxyClient[A]) getProxy(requestType string) *rest.Request {
	return t.client.CoreV1().RESTClient().Verb(requestType).
		Namespace(t.config.Namespace).
		Resource("services").
		SetHeader("Content-Type", "application/json").
		Name(fmt.Sprintf("%s:%d", t.config.ServiceName, t.config.ServicePort)).
		SubResource("proxy")
}

func (t ProxyClient[A]) getFromResponse(resp rest.Result) (result A, err error) {
	bytes, err := resp.Raw()
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(bytes, &result)
	return result, err
}

func (t ProxyClient[A]) getFromResponses(resp rest.Result) (result []A, err error) {
	bytes, err := resp.Raw()
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(bytes, &result)
	return result, err
}

func (t ProxyClient[A]) getProblemFromResponse(resp rest.Result) (problem.Problem, error) {
	bytes, respErr := resp.Raw()

	problemResponse := problem.Problem{}
	err := json.Unmarshal(bytes, &problemResponse)

	// add kubeAPI client error to details
	if respErr != nil {
		problemResponse.Detail += ";\nresp error:" + respErr.Error()
	}

	return problemResponse, err
}

// responseError tries to lookup if response is of Problem type
func (t ProxyClient[A]) responseError(resp rest.Result) error {
	if resp.Error() != nil {
		pr, err := t.getProblemFromResponse(resp)

		// if can't process response return content from response
		if err != nil {
			content, _ := resp.Raw()
			return fmt.Errorf("api server response: '%s'\nerror: %w", content, resp.Error())
		}

		return fmt.Errorf("api server problem: %s", pr.Detail)
	}

	return nil
}
