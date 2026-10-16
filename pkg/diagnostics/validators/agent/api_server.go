package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/pkg/diagnostics/validators"
	"github.com/kubeshop/testkube/pkg/k8sclient"
)

const listTimeout = 10 * time.Second

// NewAPIServerValidator checks that the pods of the Testkube API server in the namespace are ready.
// The client is created on validation, so that an unreachable cluster is reported as a result.
func NewAPIServerValidator(connect func() (kubernetes.Interface, error), namespace string) APIServerValidator {
	return APIServerValidator{connect: connect, namespace: namespace}
}

type APIServerValidator struct {
	connect   func() (kubernetes.Interface, error)
	namespace string
}

func (v APIServerValidator) Name() string {
	return "Testkube API server"
}

func (v APIServerValidator) Validate(_ any) (r validators.ValidationResult) {
	client, err := v.connect()
	if err != nil {
		return r.WithError(ErrClusterUnreachable.WithDetails(err.Error())).WithBreak()
	}

	ctx, cancel := context.WithTimeout(context.Background(), listTimeout)
	defer cancel()
	pods, err := client.CoreV1().Pods(v.namespace).List(ctx, metav1.ListOptions{LabelSelector: k8sclient.APIServerSelector})
	if err != nil {
		return r.WithError(ErrClusterUnreachable.WithDetails(err.Error())).WithBreak()
	}
	if len(pods.Items) == 0 {
		return r.WithError(ErrAPIServerNotFound.WithDetails(fmt.Sprintf("no pods with label %s in namespace %s", k8sclient.APIServerSelector, v.namespace)))
	}

	var notReady []string
	for _, pod := range pods.Items {
		if reason := notReadyReason(pod); reason != "" {
			notReady = append(notReady, fmt.Sprintf("%s: %s", pod.Name, reason))
		}
	}
	if len(notReady) > 0 {
		return r.WithError(ErrAPIServerNotReady.WithDetails(strings.Join(notReady, "; ")))
	}

	return r.WithValidStatus().WithAdditionalInfo(fmt.Sprintf("%d/%d pods ready in namespace %s", len(pods.Items), len(pods.Items), v.namespace))
}

// notReadyReason returns why a pod is not ready, or "" when it is ready.
func notReadyReason(pod corev1.Pod) string {
	if pod.DeletionTimestamp != nil {
		return "terminating"
	}
	for _, c := range pod.Status.ContainerStatuses {
		if c.State.Waiting != nil && c.State.Waiting.Reason != "" {
			return fmt.Sprintf("container %s is %s", c.Name, c.State.Waiting.Reason)
		}
	}
	for _, c := range pod.Status.Conditions {
		if c.Type == corev1.PodReady {
			if c.Status == corev1.ConditionTrue {
				return ""
			}
			if c.Reason != "" {
				return fmt.Sprintf("not ready (%s)", c.Reason)
			}
			return "not ready"
		}
	}
	return fmt.Sprintf("not ready (phase %s)", pod.Status.Phase)
}
