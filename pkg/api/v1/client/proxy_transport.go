package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubeshop/testkube/pkg/problem"
	"github.com/kubeshop/testkube/pkg/executor/output"	
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

// NewProxyTransport returns new proxy transport
func NewProxyTransport[A All](client kubernetes.Interface, config APIConfig) ProxyTransport[A] {
	return ProxyTransport[A]{
		client: client,
		config: config,
	}
}

// ProxyTransport implements proxy transport
type ProxyTransport[A All] struct {
	client kubernetes.Interface
	config APIConfig
}

// Execute is a method to make an api call for a single object
func (t ProxyTransport[A]) Execute(method, uri string, body []byte, params map[string]string) (result A, err error) {
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

	resp := req.Do(context.Background())

	if err := t.responseError(resp); err != nil {
		return result, fmt.Errorf("api/%s-%T returned error: %w", method, result, err)
	}

	return t.getFromResponse(resp)
}

// ExecuteMultiple is a method to make an api call for multiple objects
func (t ProxyTransport[A]) ExecuteMultiple(method, uri string, body []byte, params map[string]string) (result []A, err error) {
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

	resp := req.Do(context.Background())

	if err := t.responseError(resp); err != nil {
		return result, fmt.Errorf("api/%ss-%T returned error: %w", method, result, err)
	}

	return t.getFromResponses(resp)
}

// Delete is a method to make delete api call
func (t ProxyTransport[A]) Delete(uri, selector string, isContentExpected bool) error {
	req := t.getProxy(http.MethodDelete).
		Suffix(uri)

	if selector != "" {
		req.Param("selector", selector)
	}

	resp := req.Do(context.Background())

	if resp.Error() != nil {
		return resp.Error()
	}

	if err := t.responseError(resp); err != nil {
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
func (t ProxyTransport[A]) GetURI(pathTemplate string, params ...interface{}) string {
	path := fmt.Sprintf(pathTemplate, params...)
	return fmt.Sprintf("%s%s", Version, path)
}

// GetLogs returns logs stream from job pods, based on job pods logs
func (t ProxyTransport[A]) GetLogs(uri string, logs chan output.Output) error {
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

func (t ProxyTransport[A]) getProxy(requestType string) *rest.Request {
	return t.client.CoreV1().RESTClient().Verb(requestType).
		Namespace(t.config.Namespace).
		Resource("services").
		SetHeader("Content-Type", "application/json").
		Name(fmt.Sprintf("%s:%d", t.config.ServiceName, t.config.ServicePort)).
		SubResource("proxy")
}

func (t ProxyTransport[A]) getFromResponse(resp rest.Result) (result A, err error) {
	bytes, err := resp.Raw()
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(bytes, &result)
	return result, err
}

func (t ProxyTransport[A]) getFromResponses(resp rest.Result) (result []A, err error) {
	bytes, err := resp.Raw()
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(bytes, &result)
	return result, err
}

func (t ProxyTransport[A]) getProblemFromResponse(resp rest.Result) (problem.Problem, error) {
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
func (t ProxyTransport[A]) responseError(resp rest.Result) error {
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
