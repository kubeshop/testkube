package stream

import (
	"bufio"
	"context"
	"io"

	"github.com/kubeshop/testkube/pkg/utils"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

func PodLogsProxy(
	log *zap.SugaredLogger,
	c kubernetes.Interface,
	namespace string,
	pod corev1.Pod,
	logsStream LogsStream,
	id string,
) {
	ctx := context.Background()
	log = log.With("executionId", id)

	var containers []string
	for _, container := range pod.Spec.InitContainers {
		containers = append(containers, container.Name)
	}

	for _, container := range pod.Spec.Containers {
		containers = append(containers, container.Name)
	}

	go func() {
		for _, container := range containers {
			podLogOptions := corev1.PodLogOptions{
				Follow:    true,
				Container: container,
			}

			podLogRequest := c.CoreV1().
				Pods(namespace).
				GetLogs(pod.Name, &podLogOptions)

			stream, err := podLogRequest.Stream(ctx)
			if err != nil {
				log.Errorw("logs stream error", "error", err)
				continue
			}

			reader := bufio.NewReader(stream)

			for {
				b, err := utils.ReadLongLine(reader)
				if err == io.EOF {
					return
				} else if err != nil {
					log.Errorw("scanner error", "error", err)
					return
				}
				log.Debugw("reading log line", "line", string(b), "pod", pod.Name)
				logsStream.Publish(ctx, id, b)

			}
		}
	}()
}
