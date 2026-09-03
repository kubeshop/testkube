package artifacts

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/kubeshop/testkube/pkg/testworkflows/localartifacts"
)

const localUploadTimeout = 30 * time.Minute

// NewLocalUploader creates the uploader used by testkube local artifact
// stages. Unlike the Cloud uploader, it does not require a Control Plane
// connection or execution metadata: the run-owned relay validates the bearer
// token and records the requested safe path.
func NewLocalUploader(uploadURL, token, stepRef string) (Uploader, error) {
	parsed, err := url.ParseRequestURI(uploadURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("local artifact upload URL must be an absolute HTTP URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("local artifact upload URL must use HTTP or HTTPS")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("local artifact upload URL must not contain credentials, query, or fragment")
	}
	if token == "" {
		return nil, fmt.Errorf("local artifact token is not configured")
	}
	stepRef, err = localartifacts.ValidateStepRef(stepRef)
	if err != nil {
		return nil, fmt.Errorf("invalid local artifact step reference: %w", err)
	}

	return &localUploader{
		uploadURL:  strings.TrimRight(uploadURL, "/"),
		token:      token,
		pathPrefix: path.Join("steps", stepRef),
		client: &http.Client{
			Timeout: localUploadTimeout,
		},
	}, nil
}

type localUploader struct {
	uploadURL  string
	token      string
	pathPrefix string
	client     *http.Client
}

func (u *localUploader) Start() error {
	return nil
}

func (u *localUploader) Add(filePath string, file io.Reader, size int64) error {
	if closer, ok := file.(io.Closer); ok {
		defer closer.Close()
	}

	filePath, err := localartifacts.ValidateRelativePath(filePath)
	if err != nil {
		return fmt.Errorf("unsafe local artifact path: %w", err)
	}
	destination := path.Join(u.pathPrefix, filePath)

	ctx, cancel := context.WithTimeout(context.Background(), localUploadTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u.uploadURL, file)
	if err != nil {
		return fmt.Errorf("create local artifact upload request: %w", err)
	}
	if size >= 0 {
		req.ContentLength = size
	}
	req.Header.Set(localartifacts.TokenHeader, u.token)
	req.Header.Set(localartifacts.PathHeader, destination)
	req.Header.Set("Content-Type", "application/octet-stream")

	res, err := u.client.Do(req)
	if err != nil {
		return fmt.Errorf("upload local artifact %q: %w", destination, err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNoContent {
		return nil
	}

	message, _ := io.ReadAll(io.LimitReader(res.Body, 4<<10))
	if len(message) == 0 {
		return fmt.Errorf("upload local artifact %q: receiver returned HTTP %d", destination, res.StatusCode)
	}
	return fmt.Errorf("upload local artifact %q: receiver returned HTTP %d: %s", destination, res.StatusCode, strings.TrimSpace(string(message)))
}

func (u *localUploader) End() error {
	return nil
}
