package artifacts

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/ui"
)

type CloudUploaderRequestEnhancer = func(req *http.Request, path string, size int64)

func NewCloudUploader(
	client controlplaneclient.ExecutionSelfClient,
	environmentId string,
	executionId string,
	workflowName string,
	stepRef string,
	opts ...CloudUploaderOpt,
) Uploader {
	uploader := &cloudUploader{
		client:        client,
		parallelism:   1,
		reqEnhancers:  make([]CloudUploaderRequestEnhancer, 0),
		environmentId: environmentId,
		executionId:   executionId,
		workflowName:  workflowName,
		stepRef:       stepRef,
	}
	for _, opt := range opts {
		opt(uploader)
	}
	return uploader
}

type cloudUploader struct {
	client        controlplaneclient.ExecutionSelfClient
	wg            sync.WaitGroup
	sema          chan struct{}
	parallelism   int
	error         atomic.Bool
	reqEnhancers  []CloudUploaderRequestEnhancer
	waitMu        sync.Mutex
	environmentId string
	executionId   string
	workflowName  string
	stepRef       string
}

func (d *cloudUploader) Start() (err error) {
	d.sema = make(chan struct{}, d.parallelism)
	return err
}

func (d *cloudUploader) getSignedURL(name, contentType string) (string, error) {
	return d.client.SaveExecutionArtifactGetPresignedURL(
		context.Background(),
		d.environmentId,
		d.executionId,
		d.workflowName,
		d.stepRef,
		name,
		contentType,
	)
}

func (d *cloudUploader) getContentType(path string, size int64) string {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPut, "/", &bytes.Buffer{})
	if err != nil {
		return ""
	}
	for _, r := range d.reqEnhancers {
		r(req, path, size)
	}
	contentType := req.Header.Get("Content-Type")
	if contentType == "" {
		return "application/octet-stream"
	}
	return contentType
}

func (d *cloudUploader) putObject(url string, path string, file io.Reader, size int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	if size == 0 {
		// http.Request won't send Content-Length: 0, if the body is non-nil
		file = nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, file)
	if err != nil {
		return err
	}
	for _, r := range d.reqEnhancers {
		r(req, path, size)
	}
	req.ContentLength = size
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/octet-stream")
	}
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{Transport: tr}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		return errors.Errorf("failed saving file: status code: %d / message: %s", res.StatusCode, string(b))
	}
	return nil
}

func (d *cloudUploader) upload(path string, file io.Reader, size int64) {
	url, err := d.getSignedURL(path, d.getContentType(path, size))
	if err != nil {
		d.error.Store(true)
		ui.Errf("%s: failed: get signed URL: %s", path, err.Error())
		return
	}
	err = d.putObject(url, path, file, size)
	if err != nil {
		d.error.Store(true)
		ui.Errf("%s: failed: store file: %s", path, err.Error())
		return
	}
}

func (d *cloudUploader) Add(path string, file io.Reader, size int64) error {
	d.wg.Add(1)
	d.sema <- struct{}{}
	go func() {
		d.upload(path, file, size)
		if f, ok := file.(io.Closer); ok {
			_ = f.Close()
		}
		d.wg.Done()
		<-d.sema
	}()
	return nil
}

func (d *cloudUploader) End() error {
	d.waitMu.Lock()
	defer d.waitMu.Unlock()
	d.wg.Wait()
	if d.error.Load() {
		return fmt.Errorf("upload failed")
	}
	return nil
}

type CloudUploaderOpt = func(uploader *cloudUploader)

func WithParallelismCloud(parallelism int) CloudUploaderOpt {
	return func(uploader *cloudUploader) {
		if parallelism < 1 {
			parallelism = 1
		}
		uploader.parallelism = parallelism
	}
}

func WithRequestEnhancerCloud(enhancer CloudUploaderRequestEnhancer) CloudUploaderOpt {
	return func(uploader *cloudUploader) {
		uploader.reqEnhancers = append(uploader.reqEnhancers, enhancer)
	}
}
