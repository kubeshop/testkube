package sidecar

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	tcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/logs/client"
	"github.com/kubeshop/testkube/pkg/logs/events"
	"github.com/kubeshop/testkube/pkg/utils"
)

var (
	ErrStopSignalReceived = errors.New("stop signal received")
)

const (
	pollInterval    = time.Second
	podStartTimeout = 30 * time.Minute
	logsBuffer      = 1000
)

func NewProxy(clientset kubernetes.Interface, podsClient tcorev1.PodInterface, logsStream client.Stream, js jetstream.JetStream, log *zap.SugaredLogger,
	namespace, executionId, source string) *Proxy {
	return &Proxy{
		log:         log.With("service", "logs-proxy", "namespace", namespace, "executionId", executionId),
		js:          js,
		clientset:   clientset,
		namespace:   namespace,
		executionId: executionId,
		podsClient:  podsClient,
		logsStream:  logsStream,
		source:      source,
	}
}

type Proxy struct {
	log         *zap.SugaredLogger
	js          jetstream.JetStream
	clientset   kubernetes.Interface
	namespace   string
	executionId string
	source      string
	podsClient  tcorev1.PodInterface
	logsStream  client.InitializedStreamPusher
}

func (p *Proxy) Run(ctx context.Context) error {

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	logs := make(chan *events.Log, logsBuffer)

	// create stream for incoming logs
	_, err := p.logsStream.Init(ctx, p.executionId)
	if err != nil {
		return err
	}

	go func() {
		p.log.Debugw("logs proxy stream started")
		err := p.streamLogs(ctx, logs)
		if err != nil {
			p.handleError(err, "logs proxy stream error")
		}
	}()

	for l := range logs {
		select {
		case <-sigs:
			p.log.Warn("logs proxy received signal, exiting", "signal", sigs)
			p.handleError(ErrStopSignalReceived, "context cancelled stopping logs proxy")
			return ErrStopSignalReceived
		case <-ctx.Done():
			p.log.Warn("logs proxy context cancelled, exiting")
			return nil
		default:
			err = p.logsStream.Push(ctx, p.executionId, l)
			if err != nil {
				p.handleError(err, "error pushing logs to stream")
				return err
			}
		}
	}

	p.log.Infow("logs proxy sending completed")
	return nil
}

func (p *Proxy) streamLogs(ctx context.Context, logs chan *events.Log) (err error) {
	pods, err := executor.GetJobPods(ctx, p.podsClient, p.executionId, 1, 10)
	if err != nil {
		p.handleError(err, "error getting job pods")
		return err
	}

	for _, pod := range pods.Items {
		l := p.log.With("namespace", pod.Namespace, "podName", pod.Name, "podStatus", pod.Status)

		switch pod.Status.Phase {

		case corev1.PodRunning:
			l.Debug("streaming pod logs: immediately")
			return p.streamLogsFromPod(pod, logs)

		case corev1.PodFailed:
			err := fmt.Errorf("can't get pod logs, pod failed: %s/%s", pod.Namespace, pod.Name)
			p.handleError(err, "streaming pod logs: pod failed")
			return err

		default:
			l.Debugw("streaming pod logs: waiting for pod to be ready")
			testFunc := p.isPodLoggable(pod.Name)
			if err = wait.PollUntilContextTimeout(ctx, pollInterval, podStartTimeout, true, testFunc); err != nil {
				// try to get pod container statuses from Waiting and Terminated states
				status := p.getPodContainerStatuses(pod)
				p.handleError(err, "can't get pod container status after pod failure")
				return errors.Wrap(err, status)
			}

			l.Debug("streaming pod logs: pod is loggable")
			return p.streamLogsFromPod(pod, logs)
		}
	}
	return
}

func (p *Proxy) streamLogsFromPod(pod corev1.Pod, logs chan *events.Log) (err error) {
	defer close(logs)

	var containers []string
	for _, container := range pod.Spec.InitContainers {
		containers = append(containers, container.Name)
	}

	for _, container := range pod.Spec.Containers {
		containers = append(containers, container.Name)
	}

	for _, container := range containers {
		// We skip logsidecar container logs,
		// because following the logs for it will never finish
		// and the sidecar will run forever.
		if strings.HasSuffix(container, "-logs") {
			continue
		}
		req := p.podsClient.GetLogs(
			pod.Name,
			&corev1.PodLogOptions{
				Follow:     true,
				Timestamps: true,
				Container:  container,
			})

		stream, err := req.Stream(context.Background())
		if err != nil {
			p.handleError(err, "error getting pod logs stream")
			return err
		}

		reader := bufio.NewReader(stream)
		for {
			b, err := utils.ReadLongLine(reader)
			if err != nil {
				if errors.Is(err, io.EOF) {
					err = nil
				}
				break
			}

			// parse log line - also handle old (output.Output) and new format (just unstructured []byte)
			source := events.SourceJobPod
			if p.source != "" {
				source = p.source
			}

			logs <- events.NewLogFromBytes(b).
				WithSource(source)
		}

		if err != nil {
			p.handleError(err, "error while reading pod logs stream")
			return err
		}
	}

	return

}

// isPodLoggable checks if pod can be logged through kubernetes API
func (p *Proxy) isPodLoggable(podName string) wait.ConditionWithContextFunc {

	namespace := p.namespace
	c := p.clientset

	return func(ctx context.Context) (bool, error) {
		pod, err := c.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodRunning {
			return true, nil
		}

		if err = executor.IsPodFailed(pod); err != nil {
			return true, err
		}

		return false, nil
	}
}

// getPodContainerStatuses returns string with container statuses in case of failure or timeouted
func (p *Proxy) getPodContainerStatuses(pod corev1.Pod) (status string) {
	for _, s := range pod.Status.ContainerStatuses {
		if s.State.Terminated != nil {
			t := s.State.Terminated
			status += fmt.Sprintf("Pod container '%s' terminated: %s (exit code: %v, reason: %s, signal: %d); ", s.Name, t.Message, t.ExitCode, t.Reason, t.Signal)
		}

		if s.State.Waiting != nil {
			w := s.State.Waiting
			status += fmt.Sprintf("Pod conitainer '%s' waiting: %s (reason: %s); ", s.Name, w.Message, w.Reason)
		}
	}

	return status
}

// handleError will handle errors and push it as log chunk to logs stream
func (p *Proxy) handleError(err error, title string) {
	if err != nil {
		p.log.Errorw(title, "error", err)
		err = p.logsStream.Push(context.Background(), p.executionId, events.NewErrorLog(err).WithSource("logs-proxy"))
		if err != nil {
			p.log.Errorw("error pushing error to stream", "title", title, "error", err)
		}

	}
}
