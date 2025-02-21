package testworkflowprocessor

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

type BundleOptions struct {
	Secrets     []corev1.Secret
	Config      testworkflowconfig.InternalConfig
	ScheduledAt time.Time
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

func (b *Bundle) Deploy(ctx context.Context, clientSet kubernetes.Interface, namespace string) (err error) {
	if b.Job.Namespace != "" {
		namespace = b.Job.Namespace
	}
	for _, item := range b.Secrets {
		_, err = clientSet.CoreV1().Secrets(namespace).Create(ctx, &item, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to deploy secrets")
		}
	}
	for _, item := range b.ConfigMaps {
		_, err = clientSet.CoreV1().ConfigMaps(namespace).Create(ctx, &item, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to deploy config maps")
		}
	}
	for _, item := range b.Pvcs {
		_, err = clientSet.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, &item, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to deploy pvcs")
		}
	}

	_, err = clientSet.BatchV1().Jobs(namespace).Create(ctx, &b.Job, metav1.CreateOptions{})
	return errors.Wrap(err, "failed to deploy job")
}
