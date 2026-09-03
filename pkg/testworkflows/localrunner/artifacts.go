package localrunner

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/kubeshop/testkube/pkg/testworkflows/localartifacts"
	processorconstants "github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

const (
	// DefaultMaxArtifactBytes bounds one local run's total artifact-export
	// payload. Local artifact output is explicit because it may contain files a
	// workflow intentionally mounted from Kubernetes Secrets or other sensitive
	// development inputs.
	DefaultMaxArtifactBytes int64 = 100 << 20

	artifactRelayPort     = 8080
	artifactRelayTimeout  = 2 * time.Minute
	maxArtifactEntries    = 10_000
	maxArtifactPathLength = localartifacts.MaxRelativePathLength
	// A relay archive may legitimately include a PAX header and a 4 KiB name
	// for every bounded artifact file. Keep the compressed transport cap above
	// that structural overhead while retaining a finite stream bound.
	artifactArchiveOverhead int64 = int64(maxArtifactEntries) * int64(2*maxArtifactPathLength+2_048)
	artifactRelayRoot             = "/srv"
)

// ArtifactPlan describes an explicit host destination for a workflow's
// standard step.artifacts blocks. Destination is always absolute and no
// directory is created during Prepare or --dry-run.
type ArtifactPlan struct {
	Destination string
	MaxBytes    int64
}

// ArtifactSummary describes the private per-run directory published after a
// successful relay download. Bytes counts regular-file payload bytes.
type ArtifactSummary struct {
	Destination string
	Files       int
	Bytes       int64
}

// ArtifactRelay is a run-owned in-cluster receiver. Its token never appears
// in TK_CFG or an artifact command argument: the workflow artifacts stage and
// receiver each read it from the exact-labelled Secret.
type ArtifactRelay struct {
	Name       string
	Namespace  string
	URL        string
	SecretName string

	client     kubernetes.Interface
	restConfig *rest.Config
	podName    string
	token      string
	maxBytes   int64
}

// PrepareArtifactPlan validates the host base directory without mutating it.
// A per-run child is created only once Kubernetes execution is about to begin.
func PrepareArtifactPlan(destination string, maxBytes int64) (*ArtifactPlan, error) {
	if strings.TrimSpace(destination) == "" {
		return nil, nil
	}
	if maxBytes <= 0 {
		return nil, UsageError("--max-artifact-bytes must be greater than zero")
	}
	if _, err := artifactArchiveLimit(maxBytes); err != nil {
		return nil, UsageError("--max-artifact-bytes: %v", err)
	}
	abs, err := filepath.Abs(destination)
	if err != nil {
		return nil, UsageError("resolve --artifacts-dir: %v", err)
	}
	abs = filepath.Clean(abs)
	if err = validateSecureArtifactPath(abs); err != nil {
		return nil, UsageError("validate --artifacts-dir: %v", err)
	}
	return &ArtifactPlan{Destination: abs, MaxBytes: maxBytes}, nil
}

// CreateArtifactRelay creates the token Secret and receiver Pod. The workflow
// talks directly to the ready Pod IP instead of a selector-based Service: a
// different namespace Pod must not be able to add matching observable labels
// and receive the relay bearer token or artifact payload.
// Every object carries the exact local-run labels so the existing
// ResourceManager can remove a partial relay after any later failure.
func CreateArtifactRelay(ctx context.Context, client kubernetes.Interface, restConfig *rest.Config, namespace, runID string, maxBytes int64) (*ArtifactRelay, error) {
	if maxBytes <= 0 {
		return nil, fmt.Errorf("max artifact bytes must be greater than zero")
	}
	labels, err := Labels(runID, "artifact-relay")
	if err != nil {
		return nil, err
	}
	token, err := randomToken()
	if err != nil {
		return nil, fmt.Errorf("generate local artifact relay token: %w", err)
	}
	name := localObjectName("local-artifacts", runID)
	relay := &ArtifactRelay{
		Name:       name,
		Namespace:  namespace,
		SecretName: localObjectName("local-artifact-token", runID),
		client:     client,
		restConfig: restConfig,
		podName:    name,
		token:      token,
		maxBytes:   maxBytes,
	}

	secret := artifactRelaySecret(relay.SecretName, namespace, labels, token)
	if _, err = client.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
		return nil, fmt.Errorf("create local artifact relay token Secret: %w", err)
	}
	pod := artifactRelayPod(name, namespace, labels, relay.SecretName, maxBytes)
	if _, err = client.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{}); err != nil {
		return nil, fmt.Errorf("create local artifact relay pod: %w", err)
	}
	if err = relay.waitReady(ctx); err != nil {
		return nil, err
	}
	return relay, nil
}

// Download streams the receiver's fixed-argv tar archive through Kubernetes
// SPDY exec, validates every archive entry, then atomically publishes it at
// <artifact-root>/<run-id>. No user-controlled artifact path reaches a shell
// or remote command argument.
func (r *ArtifactRelay) Download(ctx context.Context, destinationRoot, runID string) (summary ArtifactSummary, err error) {
	if r == nil {
		return ArtifactSummary{}, fmt.Errorf("local artifact relay is required")
	}
	if strings.TrimSpace(runID) == "" {
		return ArtifactSummary{}, fmt.Errorf("local run ID is required")
	}
	if err = ensureSecureArtifactDirectory(destinationRoot); err != nil {
		return ArtifactSummary{}, fmt.Errorf("create local artifact destination: %w", err)
	}
	final := filepath.Join(destinationRoot, runID)
	if err = ensureArtifactDestinationAbsent(final); err != nil {
		return ArtifactSummary{}, err
	}
	partial, err := os.MkdirTemp(destinationRoot, "."+runID+".partial-")
	if err != nil {
		return ArtifactSummary{}, fmt.Errorf("create local artifact staging directory: %w", err)
	}
	if err = os.Chmod(partial, 0o700); err != nil {
		_ = os.RemoveAll(partial)
		return ArtifactSummary{}, fmt.Errorf("secure local artifact staging directory: %w", err)
	}
	published := false
	defer func() {
		if !published {
			_ = os.RemoveAll(partial)
		}
	}()

	request := r.client.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Namespace(r.Namespace).
		Name(r.podName).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: "receiver",
			Command:   []string{processorconstants.DefaultInitImageBusyboxBinaryPath + "/tar", "-C", artifactRelayRoot, "-czf", "-", "."},
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)
	executor, err := remotecommand.NewSPDYExecutor(r.restConfig, http.MethodPost, request.URL())
	if err != nil {
		return ArtifactSummary{}, fmt.Errorf("create local artifact download session: %w", err)
	}

	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	reader, writer := io.Pipe()
	type extractionResult struct {
		summary ArtifactSummary
		err     error
	}
	extraction := make(chan extractionResult, 1)
	go func() {
		summary, extractionErr := extractLocalArtifactArchive(streamCtx, reader, partial, r.maxBytes)
		_ = reader.CloseWithError(extractionErr)
		extraction <- extractionResult{summary: summary, err: extractionErr}
	}()

	var stderr bytes.Buffer
	execErr := executor.StreamWithContext(streamCtx, remotecommand.StreamOptions{
		Stdout: writer,
		Stderr: &stderr,
		Tty:    false,
	})
	_ = writer.CloseWithError(execErr)
	extracted := <-extraction
	_ = reader.Close()
	if extracted.err != nil {
		return ArtifactSummary{}, fmt.Errorf("extract local artifacts: %w", extracted.err)
	}
	if execErr != nil {
		message := strings.TrimSpace(stderr.String())
		if message != "" {
			return ArtifactSummary{}, fmt.Errorf("download local artifacts: %w: %s", execErr, message)
		}
		return ArtifactSummary{}, fmt.Errorf("download local artifacts: %w", execErr)
	}
	if err = os.Rename(partial, final); err != nil {
		return ArtifactSummary{}, fmt.Errorf("publish local artifacts: %w", err)
	}
	published = true
	extracted.summary.Destination = final
	return extracted.summary, nil
}

func artifactRelaySecret(name, namespace string, labels map[string]string, token string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: copyStringMap(labels)},
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{localartifacts.TokenSecretKey: token},
	}
}

func artifactRelayPod(name, namespace string, labels map[string]string, secretName string, maxBytes int64) *corev1.Pod {
	nonRoot := true
	noEscalation := false
	readOnlyRoot := true
	user := int64(1001)
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: copyStringMap(labels)},
		Spec: corev1.PodSpec{
			RestartPolicy:                corev1.RestartPolicyNever,
			AutomountServiceAccountToken: ptr(false),
			SecurityContext: &corev1.PodSecurityContext{
				RunAsNonRoot: &nonRoot,
				RunAsUser:    &user,
				RunAsGroup:   &user,
				FSGroup:      &user,
			},
			Containers: []corev1.Container{{
				Name:            "receiver",
				Image:           processorconstants.DefaultToolkitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command: []string{"/toolkit", "local-artifact-receiver",
					"--listen", ":8080", "--root", artifactRelayRoot,
					"--max-bytes", strconv.FormatInt(maxBytes, 10),
				},
				Env: []corev1.EnvVar{{
					Name: localartifacts.TokenEnvName,
					ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
						Key:                  localartifacts.TokenSecretKey,
					}},
				}},
				Ports: []corev1.ContainerPort{{Name: "http", ContainerPort: artifactRelayPort}},
				ReadinessProbe: &corev1.Probe{ProbeHandler: corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{
					Path: "/healthz", Port: intstr.FromInt(artifactRelayPort),
				}}},
				SecurityContext: &corev1.SecurityContext{
					AllowPrivilegeEscalation: &noEscalation,
					ReadOnlyRootFilesystem:   &readOnlyRoot,
					Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
				},
				VolumeMounts: []corev1.VolumeMount{{Name: "artifacts", MountPath: artifactRelayRoot}},
			}},
			Volumes: []corev1.Volume{{Name: "artifacts", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}},
		},
	}
}

func (r *ArtifactRelay) waitReady(ctx context.Context) error {
	readyCtx, cancel := context.WithTimeout(ctx, artifactRelayTimeout)
	defer cancel()
	for {
		pod, err := r.client.CoreV1().Pods(r.Namespace).Get(readyCtx, r.podName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("get local artifact relay pod: %w", err)
		}
		if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded {
			return fmt.Errorf("local artifact relay pod ended before becoming ready: %s", pod.Status.Reason)
		}
		if artifactRelayPodReady(pod) && pod.Status.PodIP != "" {
			r.URL = artifactRelayURL(pod.Status.PodIP)
			return nil
		}
		select {
		case <-readyCtx.Done():
			return fmt.Errorf("waiting for local artifact relay pod to become ready: %w", readyCtx.Err())
		case <-time.After(250 * time.Millisecond):
		}
	}
}

func artifactRelayURL(podIP string) string {
	return "http://" + net.JoinHostPort(podIP, strconv.Itoa(artifactRelayPort)) + "/upload"
}

func artifactRelayPodReady(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func extractLocalArtifactArchive(ctx context.Context, source io.Reader, destination string, maxBytes int64) (ArtifactSummary, error) {
	if ctx == nil {
		return ArtifactSummary{}, fmt.Errorf("artifact extraction context is required")
	}
	if err := ctx.Err(); err != nil {
		return ArtifactSummary{}, err
	}
	if maxBytes <= 0 {
		return ArtifactSummary{}, fmt.Errorf("max artifact bytes must be greater than zero")
	}
	if err := ensureSecureArtifactDirectory(destination); err != nil {
		return ArtifactSummary{}, err
	}
	archiveLimit, err := artifactArchiveLimit(maxBytes)
	if err != nil {
		return ArtifactSummary{}, err
	}
	limited := &io.LimitedReader{R: source, N: archiveLimit}
	gzipReader, err := gzip.NewReader(limited)
	if err != nil {
		return ArtifactSummary{}, fmt.Errorf("open gzip artifact archive: %w", err)
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)
	seen := map[string]struct{}{}
	var summary ArtifactSummary
	entries := 0
	for {
		if err := ctx.Err(); err != nil {
			return ArtifactSummary{}, err
		}
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return ArtifactSummary{}, fmt.Errorf("read artifact archive: %w", err)
		}
		entries++
		if entries > maxArtifactEntries {
			return ArtifactSummary{}, fmt.Errorf("artifact archive exceeds %d entries", maxArtifactEntries)
		}
		name, isRoot, err := safeArtifactArchiveName(header.Name, header.Typeflag == tar.TypeDir)
		if err != nil {
			return ArtifactSummary{}, err
		}
		if isRoot {
			if header.Typeflag != tar.TypeDir {
				return ArtifactSummary{}, fmt.Errorf("artifact archive root entry must be a directory")
			}
			continue
		}
		if _, exists := seen[name]; exists {
			return ArtifactSummary{}, fmt.Errorf("artifact archive contains duplicate path %q", name)
		}
		seen[name] = struct{}{}
		target, err := safeArtifactTarget(destination, name)
		if err != nil {
			return ArtifactSummary{}, err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := mkdirArtifactDirectory(destination, target); err != nil {
				return ArtifactSummary{}, err
			}
		case tar.TypeReg, tar.TypeRegA:
			if header.Size < 0 {
				return ArtifactSummary{}, fmt.Errorf("artifact archive file %q has a negative size", name)
			}
			if header.Size > maxBytes-summary.Bytes {
				return ArtifactSummary{}, fmt.Errorf("artifact archive exceeds max artifact bytes (%d) while adding %q", maxBytes, name)
			}
			if err := mkdirArtifactDirectory(destination, filepath.Dir(target)); err != nil {
				return ArtifactSummary{}, err
			}
			file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
			if err != nil {
				return ArtifactSummary{}, fmt.Errorf("create artifact file %q: %w", name, err)
			}
			written, copyErr := copyArtifactFile(ctx, file, tarReader, header.Size)
			closeErr := file.Close()
			if copyErr != nil {
				_ = os.Remove(target)
				return ArtifactSummary{}, fmt.Errorf("write artifact file %q: %w", name, copyErr)
			}
			if closeErr != nil {
				_ = os.Remove(target)
				return ArtifactSummary{}, fmt.Errorf("close artifact file %q: %w", name, closeErr)
			}
			if written != header.Size {
				_ = os.Remove(target)
				return ArtifactSummary{}, fmt.Errorf("artifact file %q ended early", name)
			}
			summary.Files++
			summary.Bytes += written
		default:
			return ArtifactSummary{}, fmt.Errorf("artifact archive contains unsupported entry type for %q", name)
		}
	}
	return summary, nil
}

func artifactArchiveLimit(maxBytes int64) (int64, error) {
	if maxBytes > math.MaxInt64-artifactArchiveOverhead {
		return 0, fmt.Errorf("max artifact bytes is too large")
	}
	return maxBytes + artifactArchiveOverhead, nil
}

func safeArtifactArchiveName(value string, directory bool) (name string, isRoot bool, err error) {
	if value == "" || strings.ContainsRune(value, '\x00') || strings.Contains(value, "\\") || len(value) > maxArtifactPathLength {
		return "", false, fmt.Errorf("artifact archive has an unsafe path")
	}
	name = value
	for strings.HasPrefix(name, "./") {
		name = strings.TrimPrefix(name, "./")
	}
	if directory && strings.HasSuffix(name, "/") {
		// GNU/BusyBox tar emits ordinary directory entries as "./path/". Strip
		// exactly that one syntactic directory separator, while retaining the
		// canonical-path check below to reject doubled or otherwise ambiguous
		// separators.
		name = strings.TrimSuffix(name, "/")
	}
	if name == "" || name == "." {
		return "", true, nil
	}
	if path.IsAbs(name) || filepath.IsAbs(name) || path.Clean(name) != name || name == ".." || strings.HasPrefix(name, "../") || strings.Contains(name, ":") {
		return "", false, fmt.Errorf("artifact archive has an unsafe path %q", value)
	}
	return name, false, nil
}

func safeArtifactTarget(root, name string) (string, error) {
	target := filepath.Join(root, filepath.FromSlash(name))
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("artifact archive path %q escapes the destination", name)
	}
	return target, nil
}

func copyArtifactFile(ctx context.Context, destination *os.File, source io.Reader, expected int64) (int64, error) {
	buffer := make([]byte, 32*1024)
	var written int64
	for written < expected {
		if err := ctx.Err(); err != nil {
			return written, err
		}
		remaining := expected - written
		readSize := int64(len(buffer))
		if remaining < readSize {
			readSize = remaining
		}
		n, readErr := source.Read(buffer[:readSize])
		if n > 0 {
			writeN, writeErr := destination.Write(buffer[:n])
			written += int64(writeN)
			if writeErr != nil {
				return written, writeErr
			}
			if writeN != n {
				return written, io.ErrShortWrite
			}
		}
		if readErr != nil {
			if readErr == io.EOF && written == expected {
				break
			}
			return written, readErr
		}
	}
	return written, nil
}

func validateSecureArtifactPath(destination string) error {
	if !filepath.IsAbs(destination) {
		return fmt.Errorf("artifact destination must be absolute after resolution")
	}
	current := filepath.VolumeName(destination) + string(filepath.Separator)
	for _, component := range strings.Split(strings.TrimPrefix(destination, current), string(filepath.Separator)) {
		if component == "" {
			continue
		}
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("inspect %q: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%q must not traverse a symbolic link", current)
		}
		if !info.IsDir() {
			return fmt.Errorf("%q is not a directory", current)
		}
	}
	return nil
}

func ensureSecureArtifactDirectory(destination string) error {
	if err := validateSecureArtifactPath(destination); err != nil {
		return err
	}
	current := filepath.VolumeName(destination) + string(filepath.Separator)
	for _, component := range strings.Split(strings.TrimPrefix(destination, current), string(filepath.Separator)) {
		if component == "" {
			continue
		}
		current = filepath.Join(current, component)
		created := false
		if err := os.Mkdir(current, 0o700); err != nil {
			if !errors.Is(err, os.ErrExist) {
				return fmt.Errorf("create %q: %w", current, err)
			}
		} else {
			created = true
		}
		info, err := os.Lstat(current)
		if err != nil {
			return fmt.Errorf("inspect %q: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("%q is not a secure directory", current)
		}
		if created {
			if err := os.Chmod(current, 0o700); err != nil {
				return fmt.Errorf("secure %q: %w", current, err)
			}
		}
	}
	return nil
}

func ensureArtifactDestinationAbsent(destination string) error {
	_, err := os.Lstat(destination)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect local artifact destination: %w", err)
	}
	return fmt.Errorf("local artifact destination %q already exists", destination)
}

func mkdirArtifactDirectory(root, target string) error {
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("artifact directory escapes the destination")
	}
	current := root
	if relative == "." {
		return nil
	}
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		if component == "" || component == "." || component == ".." {
			return fmt.Errorf("artifact archive has an unsafe directory path")
		}
		current = filepath.Join(current, component)
		if err := os.Mkdir(current, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("create artifact directory: %w", err)
		}
		info, err := os.Lstat(current)
		if err != nil {
			return fmt.Errorf("inspect artifact directory: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("artifact archive directory conflicts with a file")
		}
		if err := os.Chmod(current, 0o700); err != nil {
			return fmt.Errorf("secure artifact directory: %w", err)
		}
	}
	return nil
}

func ptr[T any](value T) *T { return &value }
