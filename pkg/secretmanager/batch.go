package secretmanager

import (
	"context"
	errors2 "errors"
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
)

const maxSecretSize = 750 * 1024

var (
	ErrAutoCreateDisabled = errors.New("auto-creation of secrets is disabled")
)

type batch struct {
	clientSet kubernetes.Interface
	namespace string
	prefix    string
	name      string
	secrets   []corev1.Secret
	lengths   []int
	index     int
	disabled  bool
}

func NewBatch(clientSet kubernetes.Interface, namespace, prefix, name string, disabled bool) *batch {
	return &batch{
		clientSet: clientSet,
		namespace: namespace,
		prefix:    prefix,
		name:      name,
		disabled:  disabled,
	}
}

func (s *batch) createSecret() {
	s.index++
	suffix := fmt.Sprintf("-%s%d%s", rand.String(2), s.index, rand.String(3))
	maxNameChars := maxSecretNameLength - len(s.prefix) - len(suffix)
	name := s.name
	if len(name) > maxNameChars {
		name = name[:maxNameChars]
	}
	name = s.prefix + name + suffix
	s.secrets = append(s.secrets, corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{secretCreatedByLabelName: secretCreatedByTestkubeValue},
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{},
	})
	s.lengths = append(s.lengths, 0)
}

func (s *batch) Append(key string, value string) (*corev1.EnvVarSource, error) {
	if s.disabled {
		return nil, ErrAutoCreateDisabled
	}

	// Append to existing secret if it's small enough
	length := len(key) + len(value) + 4
	for i := range s.secrets {
		if s.lengths[i]+length <= maxSecretSize {
			for _, ok := s.secrets[i].StringData[key]; ok; _, ok = s.secrets[i].StringData[key] {
				key += rand.String(2)
			}
			s.secrets[i].StringData[key] = value
			s.lengths[i] += length
			return &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: s.secrets[i].Name},
					Key:                  key,
				},
			}, nil
		}
	}

	// Create a new secret if there is no space for it
	s.createSecret()
	s.secrets[len(s.secrets)-1].StringData[key] = value
	s.lengths[len(s.secrets)-1] += length
	return &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: s.secrets[len(s.secrets)-1].Name},
			Key:                  key,
		},
	}, nil
}

func (s *batch) Create(ctx context.Context, owner *metav1.OwnerReference) error {
	created := make([]string, 0)
	for _, secret := range s.secrets {
		if owner != nil {
			secret.OwnerReferences = []metav1.OwnerReference{*owner}
		} else {
			secret.OwnerReferences = nil
		}
		obj, err := s.clientSet.CoreV1().Secrets(s.namespace).Create(ctx, &secret, metav1.CreateOptions{})
		if err != nil {
			errs := []error{err}
			for _, name := range created {
				err = s.clientSet.CoreV1().Secrets(s.namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
				if err != nil {
					errs = append(errs, errors.Wrapf(err, "failed to delete obsolete secret '%s'", name))
				}
			}
			return errors2.Join(errs...)
		}
		created = append(created, obj.Name)
	}
	return nil
}
