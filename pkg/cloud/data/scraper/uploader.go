package scraper

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/kubeshop/testkube/pkg/utils"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/cloud/data/executor"
	"github.com/kubeshop/testkube/pkg/executor/scraper"
)

type CloudUploader struct {
	executor executor.Executor
}

func NewCloudUploader(executor executor.Executor) *CloudUploader {
	return &CloudUploader{executor: executor}
}

func (l *CloudUploader) Upload(ctx context.Context, object *scraper.Object, meta map[string]any) error {
	meta["object"] = object.Name
	req, err := l.validateInfo(meta)
	if err != nil {
		return err
	}
	signedURL, err := l.getSignedURL(ctx, req)
	if err != nil {
		return errors.Wrapf(err, "failed to get signed URL for object [%s]", req.Object)
	}

	if err := l.putObject(ctx, signedURL, object.Data); err != nil {
		return errors.Wrapf(err, "failed to send object [%s] to cloud", req.Object)
	}

	return nil
}

func (l *CloudUploader) validateInfo(info map[string]any) (*PutObjectSignedURLRequest, error) {
	object, err := utils.GetStringKey(info, "object")
	if err != nil {
		return nil, err
	}
	executionID, err := utils.GetStringKey(info, "executionId")
	if err != nil {
		return nil, err
	}
	testName, err := utils.GetStringKey(info, "testName")
	if err != nil {
		return nil, err
	}
	testSuiteName, _ := utils.GetStringKey(info, "testSuiteName")
	req := &PutObjectSignedURLRequest{
		Object:        object,
		ExecutionID:   executionID,
		TestName:      testName,
		TestSuiteName: testSuiteName,
	}

	if info["testSuiteName"] != nil {
		if s, ok := info["testSuiteName"].(string); ok {
			req.TestSuiteName = s
		}
	}

	return req, nil
}

func (l *CloudUploader) getSignedURL(ctx context.Context, req *PutObjectSignedURLRequest) (string, error) {
	response, err := l.executor.Execute(ctx, CmdScraperPutObjectSignedURL, req)
	if err != nil {
		return "", err
	}
	var commandResponse PutObjectSignedURLResponse
	if err := json.Unmarshal(response, &commandResponse); err != nil {
		return "", err
	}
	return commandResponse.URL, nil
}

func (l *CloudUploader) putObject(ctx context.Context, url string, data io.Reader) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, data)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to send file to cloud")
	}
	if rsp.StatusCode != http.StatusOK {
		return errors.New("response code was not OK")
	}
	return nil
}

func ExtractCloudLoaderMeta(execution testkube.Execution) map[string]any {
	return map[string]any{
		"executionId":   execution.Id,
		"testName":      execution.TestName,
		"testSuiteName": execution.TestSuiteName,
	}
}
