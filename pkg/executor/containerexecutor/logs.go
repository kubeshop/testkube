package containerexecutor

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/utils"
)

const (
	logsStereamTimeout = 30 * time.Minute
)

// TailJobLogs - locates logs for job pod(s)
// These methods here are similar to Job executor, but they don't require the json structure.
func (c *ContainerExecutor) TailJobLogs(ctx context.Context, id, namespace string, logs chan []byte) (err error) {
	podsClient := c.clientSet.CoreV1().Pods(namespace)
	pods, err := executor.GetJobPods(ctx, podsClient, id, 1, 10)
	if err != nil {
		close(logs)
		return err
	}

	for _, pod := range pods.Items {
		if pod.Labels["job-name"] == id {

			l := c.log.With("podNamespace", pod.Namespace, "podName", pod.Name, "podStatus", pod.Status)

			switch pod.Status.Phase {

			case corev1.PodRunning:
				l.Debug("tailing pod logs: immediately")
				return tailPodLogs(c.log, c.clientSet, namespace, pod, logs)

			case corev1.PodFailed:
				err := fmt.Errorf("can't get pod logs, pod failed: %s/%s", pod.Namespace, pod.Name)
				l.Errorw(err.Error())
				return err

			default:
				l.Debugw("tailing job logs: waiting for pod to be ready")
				if err = wait.PollUntilContextTimeout(ctx, pollInterval, c.podStartTimeout, true, executor.IsPodLoggable(c.clientSet, pod.Name, namespace)); err != nil {
					l.Errorw("poll immediate error when tailing logs", "error", err)
					return err
				}

				l.Debug("tailing pod logs")
				return tailPodLogs(c.log, c.clientSet, namespace, pod, logs)
			}
		}
	}
	return
}

func tailPodLogs(l *zap.SugaredLogger, c kubernetes.Interface, namespace string, pod corev1.Pod, logs chan []byte) (err error) {
	var containers []string
	for _, container := range pod.Spec.InitContainers {
		containers = append(containers, container.Name)
	}

	for _, container := range pod.Spec.Containers {
		containers = append(containers, container.Name)
	}

	l = l.With("method", "tailPodLogs", "containers", len(containers))

	wg := sync.WaitGroup{}
	defer close(logs)

	wg.Add(len(containers))
	ctx := context.Background()

	var count int64 = 1

	for _, container := range containers {
		go func(container string) {
			defer wg.Done()
			podLogOptions := corev1.PodLogOptions{
				TailLines: &count,
				Follow:    true,
				Container: container,
			}

			podLogRequest := c.CoreV1().
				Pods(namespace).
				GetLogs(pod.Name, &podLogOptions)

			stream, err := podLogRequest.Stream(ctx)
			if err != nil {
				l.Errorw("stream error", "error", err)
				return
			}

			reader := bufio.NewReader(stream)

			for {
				b, err := utils.ReadLongLine(reader)
				if err != nil {
					if err == io.EOF {
						err = nil
					} else {
						l.Errorw("scanner error", "error", err)
					}
					break
				}
				logs <- b
				l.Debugw("log chunk pushed", "out", string(b), "pod", pod.Name)
			}
		}(container)
	}

	go func() {
		l.Debugw("waiting for all containers to finish", "containers", containers)
		wg.Wait()
		l.Infow("log stream finished")
	}()

	return
}
