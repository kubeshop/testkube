package containerexecutor

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/utils"
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
				return c.TailPodLogs(id, namespace, pod, logs)

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
				return c.TailPodLogs(id, namespace, pod, logs)
			}
		}
	}
	return
}

func (c *ContainerExecutor) TailPodLogs(id, namespace string, pod corev1.Pod, logs chan []byte) (err error) {
	var containers []string
	for _, container := range pod.Spec.InitContainers {
		containers = append(containers, container.Name)
	}

	for _, container := range pod.Spec.Containers {
		containers = append(containers, container.Name)
	}

	l := c.log.With("method", "tailPodLogs", "containers", len(containers))

	wg := sync.WaitGroup{}

	wg.Add(len(containers))
	ctx := context.Background()

	for _, container := range containers {
		if !executor.IsWhitelistedContainer(container, id, c.whitelistedContainers) {
			wg.Done()
			continue
		}
		go func(container string) {
			defer wg.Done()
			podLogOptions := corev1.PodLogOptions{
				Follow:    true,
				Container: container,
			}

			podLogRequest := c.clientSet.CoreV1().
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
				switch {
				case errors.Is(err, io.EOF):
					return
				case err != nil:
					l.Errorw("scanner error", "error", err)
					return
				}
				logs <- b
				l.Debugw("log chunk pushed", "out", string(b), "pod", pod.Name)
			}
		}(container)
	}

	go func() {
		defer close(logs)
		l.Debugw("log stream - waiting for all containers to finish", "containers", containers)
		wg.Wait()
		l.Debugw("log stream - finished")
	}()

	return
}
