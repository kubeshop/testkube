package artifact

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	intconfig "github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/archive"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/cloud/data/executor"
	"github.com/kubeshop/testkube/pkg/storage"
)

type CloudArtifactsStorage struct {
	executor executor.Executor
}

var ErrOperationNotSupported = errors.New("operation not supported")

func NewCloudArtifactsStorage(cloudClient cloud.TestKubeCloudAPIClient, proContext *intconfig.ProContext) *CloudArtifactsStorage {
	return &CloudArtifactsStorage{executor: executor.NewCloudGRPCExecutor(cloudClient, proContext)}
}

func (c *CloudArtifactsStorage) ListFiles(ctx context.Context, executionID, testName, testSuiteName, testWorkflowName string) ([]testkube.Artifact, error) {
	req := ListFilesRequest{
		ExecutionID:      executionID,
		TestName:         testName,
		TestSuiteName:    testSuiteName,
		TestWorkflowName: testWorkflowName,
	}
	response, err := c.executor.Execute(ctx, CmdArtifactsListFiles, req)
	if err != nil {
		return nil, err
	}
	var commandResponse ListFilesResponse
	if err = json.Unmarshal(response, &commandResponse); err != nil {
		return nil, err
	}

	return commandResponse.Artifacts, nil
}

func (c *CloudArtifactsStorage) DownloadFile(ctx context.Context, file, executionID, testName, testSuiteName, testWorkflowName string) (io.ReadCloser, error) {
	req := DownloadFileRequest{
		File:             file,
		ExecutionID:      executionID,
		TestName:         testName,
		TestSuiteName:    testSuiteName,
		TestWorkflowName: testWorkflowName,
	}
	response, err := c.executor.Execute(ctx, CmdArtifactsDownloadFile, req)
	if err != nil {
		return nil, err
	}
	var commandResponse DownloadFileResponse
	if err = json.Unmarshal(response, &commandResponse); err != nil {
		return nil, err
	}

	data, err := c.getObject(ctx, commandResponse.URL)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (c *CloudArtifactsStorage) DownloadArchive(ctx context.Context, executionID string,
	testName, testSuiteName, testWorkflowName string, masks []string) (io.Reader, error) {
	var regexps []*regexp.Regexp
	for _, mask := range masks {
		values := strings.Split(mask, ",")
		for _, value := range values {
			re, err := regexp.Compile(value)
			if err != nil {
				return nil, err
			}

			regexps = append(regexps, re)
		}
	}

	var files []*archive.File
	objects, err := c.ListFiles(ctx, executionID, testName, testSuiteName, testWorkflowName)
	if err != nil {
		return nil, err
	}

	current := time.Now()
	for _, obj := range objects {
		found := len(regexps) == 0
		for i := range regexps {
			if found = regexps[i].MatchString(obj.Name); found {
				break
			}
		}

		if !found {
			continue
		}

		files = append(files, &archive.File{
			Name:    obj.Name,
			Size:    int64(obj.Size),
			Mode:    int64(os.ModePerm),
			ModTime: current,
		})
	}

	// Download files concurrently with controlled parallelism
	// Limit concurrent downloads to avoid overwhelming the server
	const maxConcurrentDownloads = 10
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		semaphore = make(chan struct{}, maxConcurrentDownloads)
		errOnce   sync.Once
		dlErr     error
	)

	// Check if context is already cancelled before starting downloads
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	for i := range files {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			// Acquire semaphore slot
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Check if another goroutine has already encountered an error
			mu.Lock()
			if dlErr != nil {
				mu.Unlock()
				return
			}
			mu.Unlock()

			// Check context again before expensive operation
			select {
			case <-ctx.Done():
				errOnce.Do(func() {
					mu.Lock()
					dlErr = ctx.Err()
					mu.Unlock()
				})
				return
			default:
			}

			reader, err := c.DownloadFile(ctx, files[idx].Name, executionID, testName, testSuiteName, testWorkflowName)
			if err != nil {
				errOnce.Do(func() {
					mu.Lock()
					dlErr = err
					mu.Unlock()
				})
				return
			}
			defer reader.Close()

			buf := &bytes.Buffer{}
			if _, err = buf.ReadFrom(reader); err != nil {
				errOnce.Do(func() {
					mu.Lock()
					dlErr = err
					mu.Unlock()
				})
				return
			}

			// Safely assign the buffer to the file
			mu.Lock()
			files[idx].Data = buf
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// Check if any error occurred during concurrent downloads
	if dlErr != nil {
		return nil, dlErr
	}

	service := archive.NewTarballService()
	data := &bytes.Buffer{}
	if err = service.Create(data, files); err != nil {
		return nil, err
	}

	return data, nil
}

func (c *CloudArtifactsStorage) getObject(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get file from cloud storage")
	}
	if rsp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("error getting file from presigned url: expected 200 OK response code, got %d", rsp.StatusCode)
	}

	return rsp.Body, nil
}

func (c *CloudArtifactsStorage) UploadFile(ctx context.Context, bucketFolder string, filePath string, reader io.Reader, objectSize int64) error {
	return errors.WithStack(ErrOperationNotSupported)
}

func (c *CloudArtifactsStorage) PlaceFiles(ctx context.Context, bucketFolders []string, prefix string) error {
	return errors.WithStack(ErrOperationNotSupported)
}

func (c *CloudArtifactsStorage) GetValidBucketName(parentType string, parentName string) string {
	return ""
}

var _ storage.ArtifactsStorage = (*CloudArtifactsStorage)(nil)
