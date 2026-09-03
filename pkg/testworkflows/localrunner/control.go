package localrunner

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/transport/spdy"

	initconstants "github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/control"
	workflowconstants "github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

// IOStreams supplies terminal streams for a remote TestWorkflow shell.
type IOStreams struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
	TTY bool
}

// LocalControl reaches a single run's active workflow Pod through standard
// Kubernetes API endpoints. It never assumes a routable Pod IP, which keeps
// Kind and k3d access identical.
type LocalControl struct {
	client     kubernetes.Interface
	restConfig *rest.Config
	namespace  string
}

type workflowControlClient interface {
	Pause() error
	Resume() error
}

const localControlOperationTimeout = 15 * time.Second

const (
	localControlForwardStopTimeout = 5 * time.Second
	maxRecentPodEvents             = 10
)

func NewLocalControl(client kubernetes.Interface, restConfig *rest.Config, namespace string) *LocalControl {
	return &LocalControl{client: client, restConfig: restConfig, namespace: namespace}
}

func (c *LocalControl) Pause(ctx context.Context, runID string) error {
	return c.withControl(ctx, runID, func(client workflowControlClient) error { return client.Pause() })
}

func (c *LocalControl) Resume(ctx context.Context, runID string) error {
	return c.withControl(ctx, runID, func(client workflowControlClient) error { return client.Resume() })
}

func (c *LocalControl) withControl(ctx context.Context, runID string, operation func(workflowControlClient) error) error {
	operationCtx, cancel := context.WithTimeout(ctx, localControlOperationTimeout)
	defer cancel()
	pod, err := c.ActivePod(operationCtx, runID)
	if err != nil {
		return err
	}
	restConfig := rest.CopyConfig(c.restConfig)
	transport, upgrader, err := spdy.RoundTripperFor(restConfig)
	if err != nil {
		return fmt.Errorf("create local control port-forward transport: %w", err)
	}
	url := c.client.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Namespace(c.namespace).
		Name(pod.Name).
		SubResource("portforward").
		URL()
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, url)
	stop := make(chan struct{})
	ready := make(chan struct{})
	forwarder, err := portforward.NewOnAddresses(
		dialer,
		[]string{"127.0.0.1"},
		[]string{fmt.Sprintf("0:%d", initconstants.ControlServerPort)},
		stop,
		ready,
		io.Discard,
		io.Discard,
	)
	if err != nil {
		return fmt.Errorf("create local control port-forward: %w", err)
	}
	forwardResult := make(chan error, 1)
	go func() { forwardResult <- forwarder.ForwardPorts() }()
	forwardDone := false
	defer func() {
		close(stop)
		if forwardDone {
			return
		}
		select {
		case <-forwardResult:
		case <-time.After(localControlForwardStopTimeout):
		}
	}()

	select {
	case <-ready:
	case err := <-forwardResult:
		forwardDone = true
		if err == nil {
			err = fmt.Errorf("port-forward stopped before becoming ready")
		}
		return fmt.Errorf("start local control port-forward: %w", err)
	case <-operationCtx.Done():
		return operationCtx.Err()
	}
	ports, err := forwarder.GetPorts()
	if err != nil || len(ports) != 1 {
		if err == nil {
			err = fmt.Errorf("unexpected number of forwarded ports: %d", len(ports))
		}
		return fmt.Errorf("discover local control port: %w", err)
	}
	client, err := control.NewClient(operationCtx, "127.0.0.1", int(ports[0].Local))
	if err != nil {
		return fmt.Errorf("connect to local workflow control server: %w", err)
	}
	defer client.Close()
	operationResult := make(chan error, 1)
	go func() { operationResult <- operation(client) }()
	select {
	case err := <-operationResult:
		if err != nil {
			return fmt.Errorf("send local workflow control operation: %w", err)
		}
	case <-operationCtx.Done():
		client.Close()
		return fmt.Errorf("send local workflow control operation: %w", operationCtx.Err())
	}
	return nil
}

// PodDetails returns the exact active workflow pod and its most recent Events.
// Event selection uses the pod UID when available and always filters the
// returned list again, so a relay or similarly named pod can never leak into a
// breakpoint inspection result.
func (c *LocalControl) PodDetails(ctx context.Context, runID string) (*corev1.Pod, []corev1.Event, error) {
	pod, err := c.ActivePod(ctx, runID)
	if err != nil {
		return nil, nil, err
	}
	fieldSelector := fields.OneTermEqualSelector("involvedObject.name", pod.Name).String()
	if pod.UID != "" {
		fieldSelector = fields.AndSelectors(
			fields.OneTermEqualSelector("involvedObject.name", pod.Name),
			fields.OneTermEqualSelector("involvedObject.uid", string(pod.UID)),
		).String()
	}
	listed, err := c.client.CoreV1().Events(c.namespace).List(ctx, metav1.ListOptions{FieldSelector: fieldSelector})
	if err != nil {
		return nil, nil, fmt.Errorf("list local workflow pod events for %q: %w", pod.Name, err)
	}
	events := make([]corev1.Event, 0, len(listed.Items))
	for _, event := range listed.Items {
		if event.InvolvedObject.Name != pod.Name {
			continue
		}
		if pod.UID != "" && event.InvolvedObject.UID != pod.UID {
			continue
		}
		if event.InvolvedObject.Kind != "" && event.InvolvedObject.Kind != "Pod" {
			continue
		}
		events = append(events, event)
	}
	sort.SliceStable(events, func(i, j int) bool {
		return podEventTime(events[i]).After(podEventTime(events[j]))
	})
	if len(events) > maxRecentPodEvents {
		events = events[:maxRecentPodEvents]
	}
	return pod, events, nil
}

func podEventTime(event corev1.Event) time.Time {
	if !event.EventTime.IsZero() {
		return event.EventTime.Time
	}
	if !event.LastTimestamp.IsZero() {
		return event.LastTimestamp.Time
	}
	if !event.FirstTimestamp.IsZero() {
		return event.FirstTimestamp.Time
	}
	return event.CreationTimestamp.Time
}

// Shell opens the TestWorkflow-provided shell in the currently running
// execution container. It intentionally does not fall back to an arbitrary
// image shell: during initialization this tells the developer to wait instead.
func (c *LocalControl) Shell(ctx context.Context, runID string, streams IOStreams) error {
	pod, err := c.ActivePod(ctx, runID)
	if err != nil {
		return err
	}
	container, err := activeContainer(pod)
	if err != nil {
		return err
	}
	if streams.In == nil {
		return UsageError("local shell requires standard input")
	}
	if streams.Out == nil {
		streams.Out = io.Discard
	}
	if streams.Err == nil {
		streams.Err = io.Discard
	}
	request := c.client.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Namespace(c.namespace).
		Name(pod.Name).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   []string{workflowconstants.DefaultShellPath},
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       streams.TTY,
		}, scheme.ParameterCodec)
	executor, err := remotecommand.NewSPDYExecutor(c.restConfig, http.MethodPost, request.URL())
	if err != nil {
		return fmt.Errorf("create local workflow shell session: %w", err)
	}
	if err := executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  streams.In,
		Stdout: streams.Out,
		Stderr: streams.Err,
		Tty:    streams.TTY,
	}); err != nil {
		return fmt.Errorf("run local workflow shell in pod %q container %q: %w", pod.Name, container, err)
	}
	return nil
}

// ActivePod selects the one nonterminal workflow Pod with this exact local-run
// ownership. The source relay has a different component label and is excluded.
func (c *LocalControl) ActivePod(ctx context.Context, runID string) (*corev1.Pod, error) {
	labels, err := Labels(runID, "workflow")
	if err != nil {
		return nil, err
	}
	selector := fmt.Sprintf("%s=%s,%s=%s,%s=%s", LocalLabel, localLabelValue, LocalRunIDLabel, labels[LocalRunIDLabel], LocalComponentLabel, labels[LocalComponentLabel])
	pods, err := c.client.CoreV1().Pods(c.namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, fmt.Errorf("list local workflow pods for run %q: %w", runID, err)
	}
	active := make([]corev1.Pod, 0, len(pods.Items))
	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodFailed && pod.Status.Phase != corev1.PodSucceeded {
			active = append(active, pod)
		}
	}
	if len(active) == 0 {
		return nil, ExecutionError("no active workflow pod found for local run %q", runID)
	}
	if len(active) > 1 {
		names := make([]string, 0, len(active))
		for _, pod := range active {
			names = append(names, pod.Name)
		}
		return nil, ExecutionError("multiple active workflow pods found for local run %q: %s", runID, strings.Join(names, ", "))
	}
	return active[0].DeepCopy(), nil
}

func activeContainer(pod *corev1.Pod) (string, error) {
	for _, status := range append(append([]corev1.ContainerStatus{}, pod.Status.InitContainerStatuses...), pod.Status.ContainerStatuses...) {
		if status.State.Running != nil {
			return status.Name, nil
		}
	}
	var states []string
	for _, status := range append(append([]corev1.ContainerStatus{}, pod.Status.InitContainerStatuses...), pod.Status.ContainerStatuses...) {
		state := "waiting"
		if status.State.Terminated != nil {
			state = "terminated"
		}
		states = append(states, status.Name+"="+state)
	}
	if len(states) == 0 {
		states = []string{"no container status has been reported"}
	}
	return "", ExecutionError("local workflow pod %q has no running shell container yet (%s)", pod.Name, strings.Join(states, ", "))
}
