package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/logs/config"
	"github.com/kubeshop/testkube/pkg/logs/events"

	"github.com/kubeshop/testkube/pkg/executor"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	tcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {

	log := log.DefaultLogger.With("service", "logs-sidecar")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := Must(config.Get())

	// Event bus
	natsConn := Must(bus.NewNATSConnection(cfg.NatsURI))
	defer func() {
		log.Infof("closing nats connection")
		natsConn.Close()
	}()

	js := Must(jetstream.New(natsConn))

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	logs := make(chan events.LogChunk, 1000)

	// detect them from hostname?
	id := cfg.ExecutionId
	ns := cfg.Namespace

	// create stream for incoming logs
	streamName := "lg" + id
	s, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:    streamName,
		Storage: jetstream.FileStorage, // durable stream
	})

	if err != nil {
		log.Errorw("error creating stream", "error", err, "id", id, "stream", streamName)
		return
	}

	log.Infow("stream upserted", "stream", streamName, "info", s.CachedInfo())

	go func() {
		err := streamLogs(ctx, log, id, ns, logs)
		if err != nil {
			panic(err)
		}
	}()

	for l := range logs {
		select {
		case <-sigs:
			log.Warn("received signal, exiting", "signal", sigs)
			return
		case <-ctx.Done():
			log.Warn("context cancelled, exiting")
			return
		default:
			fmt.Printf("%s\n", l.Encode())

			js.Publish(ctx, streamName, l.Encode())
		}
	}

	log.Warn("logs sending completed")
}

func streamLogs(ctx context.Context, l *zap.SugaredLogger, id, ns string, logs chan events.LogChunk) (err error) {
	pollInterval := time.Second
	podStartTimeout := time.Second * 10

	clientset, err := ConnectToK8s()
	if err != nil {
		return err
	}

	podsClient := clientset.CoreV1().Pods(ns)

	pods, err := executor.GetJobPods(ctx, podsClient, id, 1, 10)
	if err != nil {
		panic(err)
	}

	for _, pod := range pods.Items {
		l = l.With("namespace", pod.Namespace, "podName", pod.Name, "podStatus", pod.Status)

		switch pod.Status.Phase {

		case corev1.PodRunning:
			l.Debug("tailing pod logs: immediately")
			return streamLogsFromPod(l, podsClient, ns, pod.Name, id, logs)

		case corev1.PodFailed:
			err := fmt.Errorf("can't get pod logs, pod failed: %s/%s", pod.Namespace, pod.Name)
			return err

		default:
			l.Debugw("tailing job logs: waiting for pod to be ready")
			if err = wait.PollUntilContextTimeout(
				ctx,
				pollInterval,
				podStartTimeout,
				true,
				isPodLoggable(clientset, pod.Name, ns),
			); err != nil {
				status := getPodContainerStatuses(&pod)
				return errors.Wrap(err, status)
			}

			l.Debug("tailing pod logs: pod is loggable")
			return streamLogsFromPod(l, podsClient, ns, pod.Name, id, logs)
		}
	}
	return

}

func streamLogsFromPod(log *zap.SugaredLogger, podsClient tcorev1.PodInterface, namespace, podName, container string, logs chan events.LogChunk) (err error) {
	defer close(logs)

	podLogRequest := podsClient.GetLogs(
		podName,
		&corev1.PodLogOptions{
			Follow:     true,
			Timestamps: true,
			Container:  container,
		})

	stream, err := podLogRequest.Stream(context.Background())
	if err != nil {
		log.Errorw("stream error", "error", err)
		return err
	}

	reader := bufio.NewReader(stream)
	for {
		b, err := ReadLongLine(reader)
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			break
		}

		// parse log line - also handle old and new format
		logs <- events.NewLogChunkFromBytes(b)
	}

	if err != nil {
		return err
	}

	return

}

// isPodLoggable checks if pod can be logged through kubernetes API
func isPodLoggable(c kubernetes.Interface, podName, namespace string) wait.ConditionWithContextFunc {
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
func getPodContainerStatuses(pod *corev1.Pod) (status string) {
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

// Must helper function to panic on error
func Must[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}
	return val
}

func NewNatsConnection(log *zap.SugaredLogger, natsURI string) (*nats.Conn, error) {
	nc, err := nats.Connect(natsURI)
	if err != nil {
		log.Fatalw("error connecting to nats", "error", err)
		return nil, err
	}

	return nc, nil
}

// ConnectToK8s establishes a connection to the k8s and returns a *kubernetes.Clientset
func ConnectToK8s() (*kubernetes.Clientset, error) {
	config, err := GetK8sClientConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

// GetK8sClientConfig returns k8s client config from kubeconfig or in-cluster config
// kubernetes .kube/config based is needed only for local development
// this function was copied from testkube repo
func GetK8sClientConfig() (*rest.Config, error) {
	var err error
	var config *rest.Config
	k8sConfigExists := false
	homeDir, _ := os.UserHomeDir()
	cubeConfigPath := path.Join(homeDir, ".kube/config")

	if _, err = os.Stat(cubeConfigPath); err == nil {
		k8sConfigExists = true
	}

	if cfg, exists := os.LookupEnv("KUBECONFIG"); exists {
		config, err = clientcmd.BuildConfigFromFlags("", cfg)
	} else if k8sConfigExists {
		config, err = clientcmd.BuildConfigFromFlags("", cubeConfigPath)
	} else {
		config, err = rest.InClusterConfig()
		if err == nil {
			config.QPS = 40.0
			config.Burst = 400.0
		}
	}

	if err != nil {
		return nil, err
	}

	return config, nil
}

// ReadLongLine reads long line - copied from Testkube
func ReadLongLine(r *bufio.Reader) (line []byte, err error) {
	var buffer []byte
	var isPrefix bool

	for {
		buffer, isPrefix, err = r.ReadLine()
		line = append(line, buffer...)
		if err != nil {
			break
		}

		if !isPrefix {
			break
		}
	}

	return line, err
}
