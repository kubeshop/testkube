package localinstall

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/kubeshop/testkube/pkg/archive"
)

var (
	ErrUnsupportedPlatform = errors.New("no download for this system")
	ErrChecksumMismatch    = errors.New("download does not match its checksum")
	ErrSaveFailed          = errors.New("could not save the tool")
)

// %[1]s is GOOS, %[2]s is GOARCH.
type toolSource struct {
	version       string
	manualURL     string
	url           string
	checksumURL   string
	pathInTarball string
}

// Pinned so every trial gets versions tested together.
var toolSources = map[string]toolSource{
	"kubectl": {
		version:     "v1.37.1",
		manualURL:   "https://kubernetes.io/docs/tasks/tools/",
		url:         "https://dl.k8s.io/release/v1.37.1/bin/%[1]s/%[2]s/kubectl",
		checksumURL: "https://dl.k8s.io/release/v1.37.1/bin/%[1]s/%[2]s/kubectl.sha256",
	},
	"helm": {
		version:       "v4.3.0",
		manualURL:     "https://helm.sh/docs/intro/install/",
		url:           "https://get.helm.sh/helm-v4.3.0-%[1]s-%[2]s.tar.gz",
		checksumURL:   "https://get.helm.sh/helm-v4.3.0-%[1]s-%[2]s.tar.gz.sha256sum",
		pathInTarball: "%[1]s-%[2]s/helm",
	},
	"kind": {
		version:     "v0.33.0",
		manualURL:   "https://kind.sigs.k8s.io/docs/user/quick-start/#installation",
		url:         "https://kind.sigs.k8s.io/dl/v0.33.0/kind-%[1]s-%[2]s",
		checksumURL: "https://kind.sigs.k8s.io/dl/v0.33.0/kind-%[1]s-%[2]s.sha256sum",
	},
}

type ToolInstaller struct {
	dir     string
	goos    string
	goarch  string
	sources map[string]toolSource
	client  *http.Client
}

func NewToolInstaller() (*ToolInstaller, error) {
	dir, err := ToolsDir()
	if err != nil {
		return nil, err
	}
	return &ToolInstaller{
		dir:     dir,
		goos:    runtime.GOOS,
		goarch:  runtime.GOARCH,
		sources: toolSources,
		client:  &http.Client{Timeout: 10 * time.Minute},
	}, nil
}

// Our own folder, so the user's versions stay untouched.
func ToolsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".testkube", "bin"), nil
}

// Appended, not prepended: the user's own tools still win.
func AddToolsDirToPath() error {
	dir, err := ToolsDir()
	if err != nil {
		return err
	}
	return os.Setenv("PATH", os.Getenv("PATH")+string(os.PathListSeparator)+dir)
}

func ToolVersion(name string) string {
	return toolSources[name].version
}

func ToolManualURL(name string) string {
	return toolSources[name].manualURL
}

func (i *ToolInstaller) Install(ctx context.Context, name string) (string, error) {
	src, ok := i.sources[name]
	// Windows is supported through WSL, which reports linux.
	if !ok || (i.goos != "linux" && i.goos != "darwin") || (i.goarch != "amd64" && i.goarch != "arm64") {
		return "", ErrUnsupportedPlatform
	}
	sum, err := i.fetch(ctx, fmt.Sprintf(src.checksumURL, i.goos, i.goarch))
	if err != nil {
		return "", err
	}
	data, err := i.fetch(ctx, fmt.Sprintf(src.url, i.goos, i.goarch))
	if err != nil {
		return "", err
	}
	// Checksum files hold "<hash>" or "<hash>  <filename>".
	fields := strings.Fields(string(sum))
	actual := sha256.Sum256(data)
	if len(fields) == 0 || !strings.EqualFold(fields[0], hex.EncodeToString(actual[:])) {
		return "", ErrChecksumMismatch
	}
	if src.pathInTarball != "" {
		if data, err = extractFromTarball(data, fmt.Sprintf(src.pathInTarball, i.goos, i.goarch)); err != nil {
			return "", err
		}
	}
	path, err := i.writeExecutable(name, data)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrSaveFailed, err)
	}
	return path, nil
}

func (i *ToolInstaller) fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := i.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func extractFromTarball(data []byte, path string) ([]byte, error) {
	tr, err := archive.GetTarballReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	for {
		hdr, err := tr.Next()
		if err != nil {
			return nil, fmt.Errorf("%s not found in download: %w", path, err)
		}
		if hdr.Name == path {
			return io.ReadAll(tr)
		}
	}
}

// Rename makes the swap atomic: no half-written binary on failure.
func (i *ToolInstaller) writeExecutable(name string, data []byte) (string, error) {
	if err := os.MkdirAll(i.dir, 0o755); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(i.dir, name+".tmp-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if err := os.Chmod(tmp.Name(), 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(i.dir, name)
	return path, os.Rename(tmp.Name(), path)
}
