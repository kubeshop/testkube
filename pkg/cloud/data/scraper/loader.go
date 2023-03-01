package scraper

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/cloud/data/executor"
	"github.com/kubeshop/testkube/pkg/executor/scraper"
	"github.com/kubeshop/testkube/pkg/utils"
)

type CloudLoader struct {
	executor executor.Executor
}

func NewCloudLoader(executor executor.Executor) *CloudLoader {
	return &CloudLoader{executor: executor}

}

func (l *CloudLoader) Load(ctx context.Context, object *scraper.Object, meta map[string]any) error {
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

func (l *CloudLoader) validateInfo(info map[string]any) (*PutObjectSignedURLRequest, error) {
	if err := utils.CheckStringKey(info, "object"); err != nil {
		return nil, err
	}
	if err := utils.CheckStringKey(info, "executionId"); err != nil {
		return nil, err
	}
	if err := utils.CheckStringKey(info, "testName"); err != nil {
		return nil, err
	}
	req := &PutObjectSignedURLRequest{
		Object:      info["object"].(string),
		ExecutionID: info["executionId"].(string),
		TestName:    info["testName"].(string),
	}

	if info["testSuiteName"] != nil {
		if s, ok := info["testSuiteName"].(string); ok {
			req.TestSuiteName = s
		}
	}

	return req, nil
}

func (l *CloudLoader) getSignedURL(ctx context.Context, req *PutObjectSignedURLRequest) (string, error) {
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

func (l *CloudLoader) putObject(ctx context.Context, url string, data io.Reader) error {
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
