package localrunner

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
)

const (
	sourceRelayImage        = "busybox:1.37.0"
	sourceRelayPort         = 8080
	sourceRelayReadyTimeout = 2 * time.Minute
)

// SourcePlan is a resolved local-source replacement. URL is intentionally
// only used internally; callers must never include it in ordinary terminal
// output because the random path is a short-lived bearer-like capability.
type SourcePlan struct {
	Options   SourceOptions
	MountPath string
	URL       string
}

// SourceRelay is the short-lived in-cluster HTTP endpoint for a single local
// run. Its labels make it eligible only for that run's exact cleanup.
type SourceRelay struct {
	Name      string
	Namespace string
	URL       string

	client     kubernetes.Interface
	restConfig *rest.Config
	podName    string
	token      string
}

// ResolveSourceMount validates the source mount and chooses a Git workflow's
// existing mount path before falling back to the local-runner default.
func ResolveSourceMount(workflow *testworkflowsv1.TestWorkflow, requested string) (string, error) {
	mountPath := strings.TrimSpace(requested)
	if mountPath == "" && workflow.Spec.Content != nil && workflow.Spec.Content.Git != nil {
		mountPath = workflow.Spec.Content.Git.MountPath
	}
	if mountPath == "" {
		mountPath = "/data/repo"
	}
	if !strings.HasPrefix(mountPath, "/") || path.Clean(mountPath) != mountPath || mountPath == "/" {
		return "", UsageError("--source-mount must be a clean absolute POSIX path below /, got %q", mountPath)
	}
	return mountPath, nil
}

// RewriteWorkflowWithSource deep-copies a supported workflow and replaces only
// its top-level Git source with a relay tarball. The input YAML object is never
// modified, so a developer's versioned workflow remains byte-for-byte intact.
func RewriteWorkflowWithSource(workflow *testworkflowsv1.TestWorkflow, sourceURL, mountPath string) (*testworkflowsv1.TestWorkflow, error) {
	if strings.TrimSpace(sourceURL) == "" {
		return nil, fmt.Errorf("source relay URL is required")
	}
	if err := ValidateSourceMountAvailable(workflow, mountPath); err != nil {
		return nil, err
	}
	copy := workflow.DeepCopy()
	if copy.Spec.Content == nil {
		copy.Spec.Content = &testworkflowsv1.Content{}
	}
	copy.Spec.Content.Git = nil
	mount := true
	copy.Spec.Content.Tarball = append(copy.Spec.Content.Tarball, testworkflowsv1.ContentTarball{
		Url:   sourceURL,
		Path:  mountPath,
		Mount: &mount,
	})
	return copy, nil
}

// CreateSourceRelay creates and readies the network endpoint. The caller must
// call Upload before starting the workflow Job; on any returned error its exact
// ownership labels allow ResourceManager.Clean to remove partial resources.
func CreateSourceRelay(ctx context.Context, client kubernetes.Interface, restConfig *rest.Config, namespace, runID string) (*SourceRelay, error) {
	labels, err := Labels(runID, "source-relay")
	if err != nil {
		return nil, err
	}
	token, err := randomToken()
	if err != nil {
		return nil, fmt.Errorf("generate local source relay token: %w", err)
	}
	name := localObjectName("local-source", runID)
	relay := &SourceRelay{
		Name:       name,
		Namespace:  namespace,
		URL:        fmt.Sprintf("http://%s:%d/%s.tar.gz", name, sourceRelayPort, token),
		client:     client,
		restConfig: restConfig,
		podName:    name,
		token:      token,
	}

	pod := relayPod(name, namespace, labels)
	if _, err = client.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{}); err != nil {
		return nil, fmt.Errorf("create local source relay pod: %w", err)
	}
	service := relayService(name, namespace, labels)
	if _, err = client.CoreV1().Services(namespace).Create(ctx, service, metav1.CreateOptions{}); err != nil {
		return nil, fmt.Errorf("create local source relay service: %w", err)
	}
	if err = relay.waitReady(ctx); err != nil {
		return nil, err
	}
	return relay, nil
}

// Upload streams the archive directly through the Kubernetes exec connection.
// It does not create a host-side archive or log the random object name.
func (r *SourceRelay) Upload(ctx context.Context, source SourceOptions) (SourceSummary, error) {
	request := r.client.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Namespace(r.Namespace).
		Name(r.podName).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: "httpd",
			Command: []string{
				"sh", "-ceu", "mkdir -p /srv && umask 077 && cat > /srv/" + r.token + ".tar.gz",
			},
			Stdin:  true,
			Stdout: true,
			Stderr: true,
			TTY:    false,
		}, scheme.ParameterCodec)
	executor, err := remotecommand.NewSPDYExecutor(r.restConfig, http.MethodPost, request.URL())
	if err != nil {
		return SourceSummary{}, fmt.Errorf("create source relay upload session: %w", err)
	}

	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	reader, writer := io.Pipe()
	type sourceResult struct {
		summary SourceSummary
		err     error
	}
	writeResult := make(chan sourceResult, 1)
	go func() {
		summary, writeErr := WriteSourceArchive(streamCtx, source, writer)
		_ = writer.CloseWithError(writeErr)
		writeResult <- sourceResult{summary: summary, err: writeErr}
	}()
	defer reader.Close()

	var stderr bytes.Buffer
	execErr := executor.StreamWithContext(streamCtx, remotecommand.StreamOptions{
		Stdin:  reader,
		Stdout: io.Discard,
		Stderr: &stderr,
		Tty:    false,
	})
	// Closing the reader wakes the archive goroutine if the exec request failed
	// early, while cancellation covers a blocked Kubernetes stream.
	cancel()
	_ = reader.Close()
	result := <-writeResult
	if result.err != nil {
		return SourceSummary{}, redactLocalSourceError(fmt.Errorf("package local source: %w", result.err), r.URL, r.token)
	}
	if execErr != nil {
		message := strings.TrimSpace(stderr.String())
		if message != "" {
			return SourceSummary{}, redactLocalSourceError(fmt.Errorf("upload local source archive: %w: %s", execErr, message), r.URL, r.token)
		}
		return SourceSummary{}, redactLocalSourceError(fmt.Errorf("upload local source archive: %w", execErr), r.URL, r.token)
	}
	return result.summary, nil
}

func (r *SourceRelay) waitReady(ctx context.Context) error {
	readyCtx, cancel := context.WithTimeout(ctx, sourceRelayReadyTimeout)
	defer cancel()
	for {
		pod, err := r.client.CoreV1().Pods(r.Namespace).Get(readyCtx, r.podName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("get local source relay pod: %w", err)
		}
		if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded {
			return fmt.Errorf("local source relay pod ended before becoming ready: %s", pod.Status.Reason)
		}
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
				return nil
			}
		}
		select {
		case <-readyCtx.Done():
			return fmt.Errorf("waiting for local source relay pod to become ready: %w", readyCtx.Err())
		case <-time.After(250 * time.Millisecond):
		}
	}
}

func relayPod(name, namespace string, labels map[string]string) *corev1.Pod {
	deadline := int64(defaultJobTTL / time.Second)
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: copyStringMap(labels)},
		Spec: corev1.PodSpec{
			RestartPolicy:         corev1.RestartPolicyNever,
			ActiveDeadlineSeconds: &deadline,
			Containers: []corev1.Container{{
				Name:            "httpd",
				Image:           sourceRelayImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"sh", "-ceu", "mkdir -p /srv && exec httpd -f -p 8080 -h /srv"},
				Ports:           []corev1.ContainerPort{{ContainerPort: sourceRelayPort, Name: "http"}},
				VolumeMounts:    []corev1.VolumeMount{{Name: "source", MountPath: "/srv"}},
			}},
			Volumes: []corev1.Volume{{Name: "source", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}},
		},
	}
}

func relayService(name, namespace string, labels map[string]string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: copyStringMap(labels)},
		Spec: corev1.ServiceSpec{
			Selector: copyStringMap(labels),
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Protocol:   corev1.ProtocolTCP,
				Port:       sourceRelayPort,
				TargetPort: intstr.FromInt(sourceRelayPort),
			}},
		},
	}
}

func localObjectName(prefix, runID string) string {
	name := prefix + "-" + runID
	if len(name) <= 63 {
		return name
	}
	// Retain the random run-ID suffix when shortening. Keeping only the front
	// of a maximum-length ID would leave just one hexadecimal suffix character
	// and make two concurrent source relays with similarly named workflows far
	// too likely to collide.
	suffix := runID
	if len(suffix) > 12 {
		suffix = suffix[len(suffix)-12:]
	}
	baseLimit := 63 - len(prefix) - len(suffix) - 2 // two separating dashes
	base := strings.TrimRight(runID[:baseLimit], "-")
	if base == "" {
		base = "run"
	}
	return prefix + "-" + base + "-" + suffix
}

func randomToken() (string, error) {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func copyStringMap(values map[string]string) map[string]string {
	copy := make(map[string]string, len(values))
	for key, value := range values {
		copy[key] = value
	}
	return copy
}
