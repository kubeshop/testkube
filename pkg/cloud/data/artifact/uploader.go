package artifact

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"

	"github.com/kubeshop/testkube/pkg/log"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/cloud/data/executor"
	"github.com/kubeshop/testkube/pkg/executor/scraper"
)

type CloudUploader struct {
	executor executor.Executor
	// skipVerify is used to skip TLS verification when artifacts
	skipVerify bool
}

func NewCloudUploader(executor executor.Executor, skipVerify bool) *CloudUploader {
	return &CloudUploader{executor: executor, skipVerify: skipVerify}
}

func (u *CloudUploader) Upload(ctx context.Context, object *scraper.Object, execution testkube.Execution) error {
	log.DefaultLogger.Infow("cloud uploader is requesting signed URL", "file", object.Name, "folder", execution.Id, "size", object.Size)
	req := &PutObjectSignedURLRequest{
		Object:        object.Name,
		ExecutionID:   execution.Id,
		TestName:      execution.TestName,
		TestSuiteName: execution.TestSuiteName,
	}
	signedURL, err := u.getSignedURL(ctx, req)
	if err != nil {
		return errors.Wrapf(err, "failed to get signed URL for object [%s]", req.Object)
	}

	log.DefaultLogger.Infow("cloud uploader is uploading file", "file", object.Name, "folder", req.ExecutionID, "size", object.Size)
	if err := u.putObject(ctx, signedURL, object.Data); err != nil {
		return errors.Wrapf(err, "failed to send object [%s] to cloud", req.Object)
	}

	return nil
}

func (u *CloudUploader) getSignedURL(ctx context.Context, req *PutObjectSignedURLRequest) (string, error) {
	response, err := u.executor.Execute(ctx, CmdScraperPutObjectSignedURL, req)
	if err != nil {
		return "", err
	}
	var commandResponse PutObjectSignedURLResponse
	if err := json.Unmarshal(response, &commandResponse); err != nil {
		return "", err
	}
	return commandResponse.URL, nil
}

func (u *CloudUploader) putObject(ctx context.Context, url string, data io.Reader) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, data)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: u.skipVerify}
	client := &http.Client{Transport: tr}
	rsp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to send file to cloud")
	}
	if rsp.StatusCode != http.StatusOK {
		return errors.Errorf("error getting file from presigned url: expected 200 OK response code, got %d", rsp.StatusCode)
	}
	return nil
}

func (u *CloudUploader) Close() error {
	return u.executor.Close()
}
