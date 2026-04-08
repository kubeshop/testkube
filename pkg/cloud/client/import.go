package client

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	nethttp "net/http"
	"os"

	"github.com/kubeshop/testkube/pkg/http"
)

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
func (c *ImportClient) Import(archivePath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("opening archive %s: %w", archivePath, err)
	}
	defer f.Close()

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		part, err := writer.CreateFormFile("archive", "testkube-export.tar.gz")
		if err != nil {
			pw.CloseWithError(fmt.Errorf("creating form file: %w", err))
			return
		}
		if _, err := io.Copy(part, f); err != nil {
			pw.CloseWithError(fmt.Errorf("copying archive data: %w", err))
			return
		}
		if err := writer.Close(); err != nil {
			pw.CloseWithError(fmt.Errorf("closing multipart writer: %w", err))
			return
		}
	}()

	url := c.BaseUrl + c.Path
	req, err := nethttp.NewRequestWithContext(context.Background(), nethttp.MethodPost, url, pr)
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
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("import failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	return nil
}
