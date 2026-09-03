package artifacts

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kubeshop/testkube/pkg/testworkflows/localartifacts"
)

// LocalReceiver is the run-owned in-cluster endpoint used by testkube local
// artifact stages. It accepts only authenticated PUTs and writes each upload
// atomically below Root. It is deliberately independent from the Cloud
// artifact storage protocol.
type LocalReceiver struct {
	root     string
	token    []byte
	maxBytes int64

	mu         sync.Mutex
	totalBytes int64
	fileCount  int
}

// NewLocalReceiver validates and prepares a directory for a local artifact
// relay. The token comes from a Kubernetes Secret environment variable; callers
// must never put it in a URL, command argument, or toolkit configuration.
func NewLocalReceiver(root, token string, maxBytes int64) (*LocalReceiver, error) {
	if token == "" {
		return nil, fmt.Errorf("local artifact token is not configured")
	}
	if maxBytes <= 0 {
		return nil, fmt.Errorf("local artifact maximum bytes must be greater than zero")
	}
	if root == "" {
		return nil, fmt.Errorf("local artifact root is not configured")
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve local artifact root: %w", err)
	}
	if err := os.MkdirAll(absRoot, 0o700); err != nil {
		return nil, fmt.Errorf("create local artifact root: %w", err)
	}
	rootInfo, err := os.Lstat(absRoot)
	if err != nil {
		return nil, fmt.Errorf("inspect local artifact root: %w", err)
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return nil, fmt.Errorf("local artifact root must be a directory, not a symlink")
	}
	// /srv is normally the root of a Kubernetes EmptyDir volume. This container
	// deliberately runs as an unprivileged UID, while kubelet may leave the
	// pre-existing volume root owned by root and grant write access through
	// fsGroup. Do not chmod that existing root: an unprivileged receiver could
	// otherwise fail before readiness even though it can safely create private
	// 0700 child directories below it.

	return &LocalReceiver{
		root:     absRoot,
		token:    []byte(token),
		maxBytes: maxBytes,
	}, nil
}

// Handler exposes a narrowly scoped relay API:
//
//   - GET /healthz confirms the relay is ready;
//   - PUT /upload stores one authenticated artifact.
//
// Every other path and method is rejected.
func (r *LocalReceiver) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", r.handleHealth)
	mux.HandleFunc("PUT /upload", r.handleUpload)
	return mux
}

func (r *LocalReceiver) handleHealth(writer http.ResponseWriter, _ *http.Request) {
	writer.WriteHeader(http.StatusOK)
}

func (r *LocalReceiver) handleUpload(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()
	if !r.authenticated(request) {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
		return
	}
	requestedPath, ok := singleHeader(request, localartifacts.PathHeader)
	if !ok {
		http.Error(writer, "invalid artifact path", http.StatusBadRequest)
		return
	}
	requestedPath, err := localartifacts.ValidateRelativePath(requestedPath)
	if err != nil {
		http.Error(writer, "invalid artifact path", http.StatusBadRequest)
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.write(requestedPath, request.Body, request.ContentLength); err != nil {
		switch {
		case errors.Is(err, errLocalArtifactTooLarge):
			http.Error(writer, "artifact limit exceeded", http.StatusRequestEntityTooLarge)
		case errors.Is(err, errLocalArtifactTooMany):
			http.Error(writer, "artifact file limit exceeded", http.StatusRequestEntityTooLarge)
		case errors.Is(err, errLocalArtifactExists):
			http.Error(writer, "artifact path already exists", http.StatusConflict)
		default:
			http.Error(writer, "failed to store artifact", http.StatusInternalServerError)
		}
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (r *LocalReceiver) authenticated(request *http.Request) bool {
	token, ok := singleHeader(request, localartifacts.TokenHeader)
	if !ok {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), r.token) == 1
}

func singleHeader(request *http.Request, name string) (string, bool) {
	values := request.Header.Values(name)
	return request.Header.Get(name), len(values) == 1 && values[0] != ""
}

var (
	errLocalArtifactTooLarge = errors.New("local artifact limit exceeded")
	errLocalArtifactExists   = errors.New("local artifact path already exists")
	errLocalArtifactTooMany  = errors.New("local artifact file limit exceeded")
)

func (r *LocalReceiver) write(relativePath string, body io.Reader, contentLength int64) error {
	if r.fileCount >= localartifacts.MaxArtifactFiles {
		return errLocalArtifactTooMany
	}
	remaining := r.maxBytes - r.totalBytes
	if remaining < 0 || (contentLength >= 0 && contentLength > remaining) {
		return errLocalArtifactTooLarge
	}

	destination, err := r.destination(relativePath)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(destination); err == nil {
		return errLocalArtifactExists
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	parent := filepath.Dir(destination)
	if err := r.ensureDirectories(parent); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(parent, ".local-artifact-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return err
	}

	limited := &io.LimitedReader{R: body, N: remaining}
	written, copyErr := io.Copy(temporary, limited)
	if copyErr != nil {
		_ = temporary.Close()
		return copyErr
	}
	if written == remaining {
		for {
			var extra [1]byte
			n, readErr := body.Read(extra[:])
			if n > 0 {
				_ = temporary.Close()
				return errLocalArtifactTooLarge
			}
			if errors.Is(readErr, io.EOF) {
				break
			}
			if readErr != nil {
				_ = temporary.Close()
				return readErr
			}
		}
	}
	if written > remaining {
		_ = temporary.Close()
		return errLocalArtifactTooLarge
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, destination); err != nil {
		return err
	}
	r.totalBytes += written
	r.fileCount++
	return nil
}

func (r *LocalReceiver) destination(relativePath string) (string, error) {
	components := strings.Split(relativePath, "/")
	destination := r.root
	for _, component := range components {
		if component == "" || component == "." || component == ".." {
			return "", fmt.Errorf("invalid local artifact path")
		}
		destination = filepath.Join(destination, component)
	}
	if destination == r.root || !strings.HasPrefix(destination, r.root+string(os.PathSeparator)) {
		return "", fmt.Errorf("local artifact path escapes receiver root")
	}
	return destination, nil
}

func (r *LocalReceiver) ensureDirectories(directory string) error {
	relative, err := filepath.Rel(r.root, directory)
	if err != nil || filepath.IsAbs(relative) {
		return fmt.Errorf("invalid local artifact destination")
	}
	if relative == "." {
		return nil
	}
	current := r.root
	for _, component := range strings.Split(relative, string(os.PathSeparator)) {
		if component == "" || component == "." || component == ".." {
			return fmt.Errorf("invalid local artifact destination")
		}
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if errors.Is(err, fs.ErrNotExist) {
			if err := os.Mkdir(current, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
				return err
			}
			info, err = os.Lstat(current)
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("local artifact destination contains a non-directory")
		}
	}
	return nil
}
