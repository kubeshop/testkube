package sidecar

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/logs/events"
	"github.com/kubeshop/testkube/pkg/utils"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	tcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

var (
	ErrStopSignalReceived = errors.New("stop signal received")
)

const (
	proxyStreamPrefix = "lg"
	pollInterval      = time.Second
	podStartTimeout   = time.Second * 60
	logsBuffer        = 1000
)

func NewProxy(clientset *kubernetes.Clientset, podsClient tcorev1.PodInterface, js jetstream.JetStream, log *zap.SugaredLogger, namespace, executionId string) *Proxy {
	streamName := proxyStreamPrefix + executionId
	return &Proxy{
		log:         log.With("namespace", namespace, "executionId", executionId, "stream", streamName),
		js:          js,
		clientset:   clientset,
		namespace:   namespace,
		executionId: executionId,
		streamName:  streamName,
		podsClient:  podsClient,
	}
}

type Proxy struct {
	log         *zap.SugaredLogger
	js          jetstream.JetStream
	clientset   *kubernetes.Clientset
	namespace   string
	executionId string
	streamName  string
	podsClient  tcorev1.PodInterface
}

func (p *Proxy) Run(ctx context.Context) error {

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	logs := make(chan events.LogChunk, logsBuffer)

	// create stream for incoming logs

	s, err := p.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:    p.streamName,
		Storage: jetstream.FileStorage, // durable stream
	})
	if err != nil {
		p.log.Errorw("error creating stream", "error", err)
		return err
	}
	p.log.Debugw("logs proxy stream upserted", "info", s.CachedInfo())

	go func() {
		p.log.Debugw("logs proxy stream started")
		err := p.streamLogs(ctx, logs)
		if err != nil {
			p.log.Errorw("logs stream failed", "error", err)
		}
	}()

	for l := range logs {
		select {
		case <-sigs:
			p.log.Warn("logs proxy received signal, exiting", "signal", sigs)
			return ErrStopSignalReceived
		case <-ctx.Done():
			p.log.Warn("logs proxy context cancelled, exiting")
			return nil
		default:
			p.js.Publish(ctx, p.streamName, l.Encode())
		}
	}

	p.log.Infow("logs proxy sending completed")
	return nil
}

func (p *Proxy) streamLogs(ctx context.Context, logs chan events.LogChunk) (err error) {
	pods, err := executor.GetJobPods(ctx, p.podsClient, p.executionId, 1, 10)
	if err != nil {
		panic(err)
	}

	for _, pod := range pods.Items {
		l := p.log.With("namespace", pod.Namespace, "podName", pod.Name, "podStatus", pod.Status)

		switch pod.Status.Phase {

		case corev1.PodRunning:
			l.Debug("streaming pod logs: immediately")
			return p.streamLogsFromPod(pod, logs)

		case corev1.PodFailed:
			err := fmt.Errorf("can't get pod logs, pod failed: %s/%s", pod.Namespace, pod.Name)
			return err

		default:
			l.Debugw("streaming pod logs: waiting for pod to be ready")
			testFunc := p.isPodLoggable(pod.Name)
			if err = wait.PollUntilContextTimeout(ctx, pollInterval, podStartTimeout, true, testFunc); err != nil {
				status := p.getPodContainerStatuses(pod)
				return errors.Wrap(err, status)
			}

			l.Debug("streaming pod logs: pod is loggable")
			return p.streamLogsFromPod(pod, logs)
		}
	}
	return
}

func (p *Proxy) streamLogsFromPod(pod corev1.Pod, logs chan events.LogChunk) (err error) {
	defer close(logs)

	var containers []string
	for _, container := range pod.Spec.InitContainers {
		containers = append(containers, container.Name)
	}

	for _, container := range pod.Spec.Containers {
		containers = append(containers, container.Name)
	}

	for _, container := range containers {

		req := p.podsClient.GetLogs(
			pod.Name,
			&corev1.PodLogOptions{
				Follow:     true,
				Timestamps: true,
				Container:  container,
			})

		stream, err := req.Stream(context.Background())
		if err != nil {
			p.log.Errorw("stream error", "error", err)
			return err
		}

		reader := bufio.NewReader(stream)
		for {
			b, err := utils.ReadLongLine(reader)
			if err != nil {
				if err == io.EOF {
					err = nil
				}
				break
			}

			// parse log line - also handle old (output.Output) and new format (just unstructured []byte)
			logs <- events.NewLogChunkFromBytes(b)
		}

		if err != nil {
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
