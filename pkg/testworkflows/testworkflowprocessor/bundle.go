package testworkflowprocessor

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

// RuntimeOptions contains runtime overrides for test workflow execution
type RuntimeOptions struct {
	Variables map[string]string
}

type BundleOptions struct {
	Secrets                []corev1.Secret
	Config                 testworkflowconfig.InternalConfig
	ScheduledAt            time.Time
	CommonEnvVariables     []corev1.EnvVar
	AllowLowSecurityFields bool
	Runtime                *RuntimeOptions // Runtime configuration overrides
}

type Bundle struct {
	Secrets       []corev1.Secret
	ConfigMaps    []corev1.ConfigMap
	Pvcs          []corev1.PersistentVolumeClaim
	Job           batchv1.Job
	Signature     []stage.Signature
	FullSignature []stage.Signature
}

func (b *Bundle) Actions() (actions actiontypes.ActionGroups) {
	_ = json.Unmarshal([]byte(b.Job.Spec.Template.Annotations[constants.SpecAnnotationName]), &actions)
	return
}

func (b *Bundle) LiteActions() (actions lite.LiteActionGroups) {
	_ = json.Unmarshal([]byte(b.Job.Spec.Template.Annotations[constants.SpecAnnotationName]), &actions)
	return
}

func (b *Bundle) SetGroupId(groupId string) {
	AnnotateGroupId(&b.Job, groupId)
	for i := range b.ConfigMaps {
		AnnotateGroupId(&b.ConfigMaps[i], groupId)
	}
	for i := range b.Secrets {
		AnnotateGroupId(&b.Secrets[i], groupId)
	}
	for i := range b.Pvcs {
		AnnotateGroupId(&b.Pvcs[i], groupId)
	}
}

func (b *Bundle) SetRunnerId(runnerId string) {
	AnnotateRunnerId(&b.Job, runnerId)
	for i := range b.ConfigMaps {
		AnnotateRunnerId(&b.ConfigMaps[i], runnerId)
	}
	for i := range b.Secrets {
		AnnotateRunnerId(&b.Secrets[i], runnerId)
	}
	for i := range b.Pvcs {
		AnnotateRunnerId(&b.Pvcs[i], runnerId)
	}
}

func (b *Bundle) Deploy(ctx context.Context, clientSet kubernetes.Interface, namespace string) (err error) {
	if b.Job.Namespace != "" {
		namespace = b.Job.Namespace
	}
	for _, item := range b.Secrets {
		_, err = clientSet.CoreV1().Secrets(namespace).Create(ctx, &item, metav1.CreateOptions{})
		if err != nil {
			return executionworkertypes.WithStartReason(errors.Wrap(err, "failed to deploy secrets"), testkube.StartReasonResourceFailed)
		}
	}
	for _, item := range b.ConfigMaps {
		_, err = clientSet.CoreV1().ConfigMaps(namespace).Create(ctx, &item, metav1.CreateOptions{})
		if err != nil {
			return executionworkertypes.WithStartReason(errors.Wrap(err, "failed to deploy config maps"), testkube.StartReasonResourceFailed)
		}
	}
	for _, item := range b.Pvcs {
		_, err = clientSet.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, &item, metav1.CreateOptions{})
		if err != nil {
			return executionworkertypes.WithStartReason(errors.Wrap(err, "failed to deploy pvcs"), testkube.StartReasonResourceFailed)
		}
	}

	_, err = clientSet.BatchV1().Jobs(namespace).Create(ctx, &b.Job, metav1.CreateOptions{})
	return jobCreateError(err)
}

// jobFieldPrefix is the part of a field path that the job adds to the pod of the workflow. The
// workflow names neither the template nor the container, so the path starts after them.
var jobFieldPrefix = regexp.MustCompile(`^spec\.template\.spec\.((initContainers|containers)\[\d+\]\.)?`)

// jobCreateError puts the refusal of Kubernetes into words. Kubernetes names the job by its
// generated ID and repeats a cause once for each container that holds the same field, so the
// message keeps each cause once, without the ID.
func jobCreateError(err error) error {
	var statusErr *apierrors.StatusError
	if !errors.As(err, &statusErr) || statusErr.ErrStatus.Details == nil || len(statusErr.ErrStatus.Details.Causes) == 0 {
		return errors.WithStack(err)
	}
	seen := make(map[string]struct{}, len(statusErr.ErrStatus.Details.Causes))
	causes := make([]string, 0, len(statusErr.ErrStatus.Details.Causes))
	for _, cause := range statusErr.ErrStatus.Details.Causes {
		text := cause.Message
		if field := jobFieldPrefix.ReplaceAllString(cause.Field, ""); field != "" {
			text = field + ": " + text
		}
		if _, ok := seen[text]; ok {
			continue
		}
		seen[text] = struct{}{}
		causes = append(causes, text)
	}
	header := "the job cannot be created"
	if statusErr.ErrStatus.Reason == metav1.StatusReasonInvalid {
		header = "the job is invalid"
	}
	return &jobErr{message: header + ": " + strings.Join(causes, "; "), err: err}
}

// jobErr keeps the original error for errors.As and errors.Is, and gives the words for people.
type jobErr struct {
	message string
	err     error
}

func (e *jobErr) Error() string { return e.message }
func (e *jobErr) Unwrap() error { return e.err }
