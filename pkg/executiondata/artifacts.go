package executiondata

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// MaxInlineArtifactSize caps what read_artifact() may pull into an expression.
	// Larger files belong in a `fetch` block, which writes them to disk instead.
	MaxInlineArtifactSize = 1024 * 1024

	// ArtifactDownloadTimeout bounds a single artifact download.
	ArtifactDownloadTimeout = 60 * time.Second
)

// ReadArtifact downloads a single artifact of an execution and returns its content.
func ReadArtifact(ctx context.Context, repository ExecutionRepository, id, path string) (string, error) {
	if repository == nil {
		return "", fmt.Errorf("cannot read artifacts: this workflow has no connection to the control plane")
	}

	artifacts, err := repository.ListArtifacts(ctx, id, []string{path})
	if err != nil {
		return "", err
	}

	artifact, err := exactArtifact(artifacts, path)
	if err != nil {
		return "", err
	}
	if artifact.Size > MaxInlineArtifactSize {
		return "", fmt.Errorf("artifact %q is %d bytes, over the %d byte limit: use a 'fetch' block to download it to the file system instead",
			path, artifact.Size, MaxInlineArtifactSize)
	}

	return download(ctx, artifact.Url, path)
}

// exactArtifact picks the artifact stored under the requested path. The control plane
// treats the path as a pattern, so a wildcard may bring back more than one file.
func exactArtifact(artifacts []Artifact, path string) (Artifact, error) {
	if len(artifacts) == 0 {
		return Artifact{}, fmt.Errorf("artifact %q not found", path)
	}
	for _, artifact := range artifacts {
		if artifact.Path == path {
			return artifact, nil
		}
	}
	if len(artifacts) > 1 {
		return Artifact{}, fmt.Errorf("%q matches %d artifacts: point at a single file, or use a 'fetch' block to download them all", path, len(artifacts))
	}
	return artifacts[0], nil
}

// FetchResult summarises a completed fetch.
type FetchResult struct {
	Files int
	Bytes int64
}

// FetchArtifacts downloads every artifact of an execution matching the patterns into dir,
// preserving the layout the execution stored them in.
func FetchArtifacts(ctx context.Context, repository ExecutionRepository, id string, patterns []string, dir string) (FetchResult, error) {
	var result FetchResult
	if repository == nil {
		return result, fmt.Errorf("cannot fetch artifacts: this workflow has no connection to the control plane")
	}
	if dir == "" {
		return result, fmt.Errorf("cannot fetch artifacts: no target directory provided")
	}

	artifacts, err := repository.ListArtifacts(ctx, id, patterns)
	if err != nil {
		return result, err
	}

	for _, artifact := range artifacts {
		target, err := targetPath(dir, artifact.Path)
		if err != nil {
			return result, err
		}
		if err = os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return result, fmt.Errorf("creating directory for artifact %q: %w", artifact.Path, err)
		}
		written, err := downloadTo(ctx, artifact.Url, artifact.Path, target)
		if err != nil {
			return result, err
		}
		result.Files++
		result.Bytes += written
	}
	return result, nil
}

// targetPath resolves where an artifact is written, refusing paths that would land
// outside the target directory.
func targetPath(dir, path string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(path))
	escapes := clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator))
	if clean == "." || filepath.IsAbs(clean) || escapes {
		return "", fmt.Errorf("artifact %q would be written outside of %q", path, dir)
	}
	return filepath.Join(filepath.Clean(dir), clean), nil
}

func downloadTo(ctx context.Context, url, path, target string) (int64, error) {
	res, err := get(ctx, url, path)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()

	file, err := os.Create(target)
	if err != nil {
		return 0, fmt.Errorf("creating file for artifact %q: %w", path, err)
	}
	defer file.Close()

	written, err := io.Copy(file, res.Body)
	if err != nil {
		return 0, fmt.Errorf("saving artifact %q: %w", path, err)
	}
	return written, nil
}

func download(ctx context.Context, url, path string) (string, error) {
	res, err := get(ctx, url, path)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	// Read one byte past the limit, so an artifact the control plane reported no size
	// for is still rejected instead of being silently truncated.
	content, err := io.ReadAll(io.LimitReader(res.Body, MaxInlineArtifactSize+1))
	if err != nil {
		return "", fmt.Errorf("reading artifact %q: %w", path, err)
	}
	if len(content) > MaxInlineArtifactSize {
		return "", fmt.Errorf("artifact %q is over the %d byte limit: use a 'fetch' block to download it to the file system instead",
			path, MaxInlineArtifactSize)
	}
	return string(content), nil
}

// artifactClient bounds the whole exchange, body included, unlike a context deadline
// that the caller would have to keep alive while streaming.
var artifactClient = &http.Client{Timeout: ArtifactDownloadTimeout}

func get(ctx context.Context, url, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("downloading artifact %q: %w", path, err)
	}
	res, err := artifactClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading artifact %q: %w", path, err)
	}
	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		return nil, fmt.Errorf("downloading artifact %q: unexpected status %s", path, res.Status)
	}
	return res, nil
}
