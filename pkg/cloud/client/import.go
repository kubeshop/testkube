package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	nethttp "net/http"
	"os"

	apiclient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/http"
)

const (
	maxErrorResponseBytes = 1024 * 1024 // 1 MB cap for error responses
	importFormFileKey     = "file"
)

// HTTPError represents an HTTP error response with a status code.
type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("import failed (HTTP %d): %s", e.StatusCode, e.Body)
}

// NewImportClient creates a new client for importing execution data archives to the control plane.
func NewImportClient(baseUrl, token, orgID, envID string) *ImportClient {
	return &ImportClient{
		BaseUrl: baseUrl,
		Path:    "/organizations/" + orgID + "/environments/" + envID + "/agent/import",
		Client:  http.NewClient(),
		Token:   token,
	}
}

// ImportClient handles uploading execution data archives to the control plane.
type ImportClient struct {
	BaseUrl string
	Path    string
	Client  http.HttpClient
	Token   string
}

// Import uploads an execution data archive (tar.gz) to the control plane.
func (c *ImportClient) Import(ctx context.Context, archivePath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("opening archive %s: %w", archivePath, err)
	}
	defer f.Close()

	// Build the multipart body in memory so the pipe goroutine cannot leak.
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile(importFormFileKey, apiclient.ExportArchiveFileName)
	if err != nil {
		return fmt.Errorf("creating form file: %w", err)
	}
	if _, err := io.Copy(part, f); err != nil {
		return fmt.Errorf("copying archive data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("closing multipart writer: %w", err)
	}

	url := c.BaseUrl + c.Path
	req, err := nethttp.NewRequestWithContext(ctx, nethttp.MethodPost, url, &buf)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("uploading archive: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorResponseBytes))
		return &HTTPError{StatusCode: resp.StatusCode, Body: string(body)}
	}

	return nil
}
