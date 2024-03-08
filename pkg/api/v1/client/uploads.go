package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/kubeshop/testkube/pkg/problem"
)

const uri string = "/uploads"

type CopyFileClient interface {
	UploadFile(parentName string, parentType TestingType, filePath string, fileContent []byte, timeout time.Duration) error
}

type CopyFileDirectClient struct {
	client        *http.Client
	apiURI        string
	apiPathPrefix string
}

func NewCopyFileDirectClient(httpClient *http.Client, apiURI, apiPathPrefix string) *CopyFileDirectClient {
	return &CopyFileDirectClient{
		client:        httpClient,
		apiURI:        apiURI,
		apiPathPrefix: apiPathPrefix,
	}
}

type CopyFileProxyClient struct {
	client kubernetes.Interface
	config APIConfig
}

func NewCopyFileProxyClient(client kubernetes.Interface, config APIConfig) *CopyFileProxyClient {
	return &CopyFileProxyClient{
		client: client,
		config: config,
	}
}

// UploadFile uploads a copy file to the API server
func (c CopyFileDirectClient) UploadFile(parentName string, parentType TestingType, filePath string, fileContent []byte, timeout time.Duration) error {
	body, writer, err := createUploadFileBody(filePath, fileContent, parentName, parentType)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.getUri(), body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	clientTimeout := c.client.Timeout
	if timeout != clientTimeout {
		c.client.Timeout = timeout
	}
	resp, err := c.client.Do(req)
	c.client.Timeout = clientTimeout
	if err != nil {
		return err
	}

	if err = httpResponseError(resp); err != nil {
		return fmt.Errorf("api %s returned error: %w", uri, err)
	}

	return nil
}

func (c CopyFileDirectClient) getUri() string {
	return strings.Join([]string{c.apiPathPrefix, c.apiURI, "/", Version, uri}, "")
}

// UploadFile uploads a copy file to the API server
func (c CopyFileProxyClient) UploadFile(parentName string, parentType TestingType, filePath string, fileContent []byte, timeout time.Duration) error {
	body, writer, err := createUploadFileBody(filePath, fileContent, parentName, parentType)
	if err != nil {
		return err
	}

	// by default the timeout is 0 for the K8s client, which means no timeout
	clientTimeout := time.Duration(0)
	if timeout != clientTimeout {
		clientTimeout = timeout
	}
	req := c.client.CoreV1().RESTClient().Verb(http.MethodPost).
		Namespace(c.config.Namespace).
		Resource("services").
		SetHeader("Content-Type", writer.FormDataContentType()).
		Name(fmt.Sprintf("%s:%d", c.config.ServiceName, c.config.ServicePort)).
		SubResource("proxy").
		Timeout(clientTimeout).
		Suffix(Version + uri).
		Body(body)

	resp := req.Do(context.Background())

	if err := k8sResponseError(resp); err != nil {
		return fmt.Errorf("api %s returned error: %w", uri, err)
	}

	return nil
}

func createUploadFileBody(filePath string, fileContent []byte, parentName string, parentType TestingType) (*bytes.Buffer, *multipart.Writer, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("attachment", filepath.Base(filePath))
	if err != nil {
		return body, writer, fmt.Errorf("could not send file: %w", err)
	}

	if _, err := io.Copy(part, bytes.NewBuffer(fileContent)); err != nil {
		return body, writer, fmt.Errorf("could not write file: %w", err)
	}
	err = writer.WriteField("parentName", parentName)
	if err != nil {
		return body, writer, fmt.Errorf("could not add parentName: %w", err)
	}
	err = writer.WriteField("parentType", string(parentType))
	if err != nil {
		return body, writer, fmt.Errorf("could not add parentType: %w", err)
	}
	err = writer.WriteField("filePath", filePath)
	if err != nil {
		return body, writer, fmt.Errorf("could not add filePath: %w", err)
	}
	err = writer.Close()
	if err != nil {
		return body, writer, fmt.Errorf("could not close copyfile writer: %w", err)
	}
	return body, writer, nil
}

// httpResponseError tries to lookup if response is of Problem type
func httpResponseError(resp *http.Response) error {
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

// k8sResponseError tries to lookup if response is of Problem type
func k8sResponseError(resp rest.Result) error {
	if resp.Error() != nil {
		pr, err := getProblemFromK8sResponse(resp)

		// if can't process response return content from response
		if err != nil {
			content, _ := resp.Raw()
			return fmt.Errorf("api server response: '%s'\nerror: %w", content, resp.Error())
		}

		return fmt.Errorf("api server problem: %s", pr.Detail)
	}

	return nil
}

// getProblemFromK8sResponse gets the error message from the K8s response
func getProblemFromK8sResponse(resp rest.Result) (problem.Problem, error) {
	bytes, respErr := resp.Raw()

	problemResponse := problem.Problem{}
	err := json.Unmarshal(bytes, &problemResponse)

	// add kubeAPI client error to details
	if respErr != nil {
		problemResponse.Detail += ";\nresp error:" + respErr.Error()
	}

	return problemResponse, err
}
