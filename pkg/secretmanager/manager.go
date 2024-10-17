package secretmanager

import (
	"context"
	"errors"
	"fmt"
	"maps"

	errors2 "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/mapper/secrets"
)

const (
	maxSecretNameLength          = 63
	secretCreatedByLabelName     = "createdBy"
	secretCreatedByTestkubeValue = "testkube"
)

var (
	ErrManagementDisabled = errors.New("secret management is disabled")
	ErrDeleteDisabled     = errors.New("deleting secrets is disabled")
	ErrCreateDisabled     = errors.New("creating secrets is disabled")
	ErrModifyDisabled     = errors.New("modifying secrets is disabled")
	ErrNotFound           = errors.New("secret not found")
	ErrNotControlled      = errors.New("secret is not controlled by Testkube")
)

type CreateOptions struct {
	Type   string
	Labels map[string]string
	Owner  *metav1.OwnerReference
	Bypass bool
}

type UpdateOptions struct {
	Labels map[string]string
	Owner  *metav1.OwnerReference
}

type secretManager struct {
	clientSet kubernetes.Interface
	config    testkube.SecretConfig
}

type SecretManager interface {
	Batch(prefix, name string) *batch
	InsertBatch(ctx context.Context, namespace string, batch *batch, owner *metav1.OwnerReference) error
	List(ctx context.Context, namespace string, all bool) ([]testkube.Secret, error)
	Get(ctx context.Context, namespace, name string) (testkube.Secret, error)
	Delete(ctx context.Context, namespace, name string) error
	Create(ctx context.Context, namespace, name string, data map[string]string, opts CreateOptions) (testkube.Secret, error)
	Update(ctx context.Context, namespace, name string, data map[string]string, opts UpdateOptions) (testkube.Secret, error)
}

func New(clientSet kubernetes.Interface, config testkube.SecretConfig) SecretManager {
	return &secretManager{
		clientSet: clientSet,
		config:    config,
	}
}

func (s *secretManager) Batch(prefix, name string) *batch {
	return NewBatch(s.config.Prefix+prefix, name, !s.config.AutoCreate)
}

func (s *secretManager) InsertBatch(ctx context.Context, namespace string, batch *batch, owner *metav1.OwnerReference) error {
	if !batch.HasData() {
		return nil
	} else if !s.config.AutoCreate {
		return ErrAutoCreateDisabled
	}

	created := make([]string, 0)
	for _, secret := range batch.Get() {
		if owner != nil {
			secret.OwnerReferences = []metav1.OwnerReference{*owner}
		} else {
			secret.OwnerReferences = nil
		}
		obj, err := s.clientSet.CoreV1().Secrets(namespace).Create(ctx, &secret, metav1.CreateOptions{})
		if err != nil {
			errs := []error{err}
			for _, name := range created {
				err = s.clientSet.CoreV1().Secrets(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
				if err != nil {
					errs = append(errs, errors2.Wrapf(err, "failed to delete obsolete secret '%s'", name))
				}
			}
			return errors.Join(errs...)
		}
		created = append(created, obj.Name)
	}
	return nil
}

func (s *secretManager) List(ctx context.Context, namespace string, all bool) ([]testkube.Secret, error) {
	if !s.config.List {
		return nil, ErrManagementDisabled
	}
	if all && !s.config.ListAll {
		all = false
	}
	selector := ""
	if !all {
		selector = fmt.Sprintf("%s=%s", secretCreatedByLabelName, secretCreatedByTestkubeValue)
	}
	list, err := s.clientSet.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return nil, err
	}
	results := make([]testkube.Secret, len(list.Items))
	for i, secret := range list.Items {
		results[i] = secrets.MapSecretKubeToAPI(&secret)
	}
	return results, nil
}

func (s *secretManager) Get(ctx context.Context, namespace, name string) (testkube.Secret, error) {
	if !s.config.List {
		return testkube.Secret{}, ErrManagementDisabled
	}

	// Get the secret details
	secret, err := s.clientSet.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	mayAccess := secret == nil || secret.Labels[secretCreatedByLabelName] == secretCreatedByTestkubeValue || s.config.ListAll

	// When no permissions, make it the same as when it's actually not found, to avoid blind search
	if err == nil && !mayAccess {
		err = k8serrors.NewNotFound(schema.GroupResource{Group: "", Resource: "Secret"}, name)
	}
	if err != nil {
		// Return same not found error for both not found, and not permitted, to avoid blind search
		if k8serrors.IsNotFound(err) {
			return testkube.Secret{}, ErrNotFound
		}
		return testkube.Secret{}, err
	}

	return secrets.MapSecretKubeToAPI(secret), nil
}

func (s *secretManager) Delete(ctx context.Context, namespace, name string) error {
	if !s.config.Delete {
		return ErrDeleteDisabled
	}

	// Get the secret details
	secret, err := s.clientSet.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	isControlled := secret != nil && secret.Labels[secretCreatedByLabelName] == secretCreatedByTestkubeValue
	mayAccess := secret == nil || isControlled || s.config.ListAll

	// When no permissions, make it the same as when it's actually not found, to avoid blind search
	if err == nil && !mayAccess {
		err = k8serrors.NewNotFound(schema.GroupResource{Group: "", Resource: "Secret"}, name)
	}
	if err != nil {
		// Return same not found error for both not found, and not permitted, to avoid blind search
		if k8serrors.IsNotFound(err) {
			return ErrNotFound
		}
		return err
	}

	// Disallow when it is not controlled by Testkube
	if !isControlled {
		return ErrNotControlled
	}

	return s.clientSet.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{
		GracePeriodSeconds: common.Ptr(int64(0)),
		PropagationPolicy:  common.Ptr(metav1.DeletePropagationBackground),
	})
}

func (s *secretManager) Create(ctx context.Context, namespace, name string, data map[string]string, opts CreateOptions) (testkube.Secret, error) {
	if !s.config.Create && !opts.Bypass {
		return testkube.Secret{}, ErrCreateDisabled
	}

	// Build the secret
	labels := map[string]string{}
	maps.Copy(labels, opts.Labels)
	labels[secretCreatedByLabelName] = secretCreatedByTestkubeValue
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: s.config.Prefix + name, Labels: opts.Labels},
		Type:       corev1.SecretType(opts.Type),
		StringData: data,
	}
	if opts.Owner != nil && opts.Owner.UID != "" {
		secret.OwnerReferences = []metav1.OwnerReference{*opts.Owner}
	}

	// Create the resource
	secret, err := s.clientSet.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return testkube.Secret{}, err
	}
	return secrets.MapSecretKubeToAPI(secret), nil
}

func (s *secretManager) Update(ctx context.Context, namespace, name string, data map[string]string, opts UpdateOptions) (testkube.Secret, error) {
	if !s.config.Modify {
		return testkube.Secret{}, ErrModifyDisabled
	}

	// Get the secret details
	secret, err := s.clientSet.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	isControlled := secret != nil && secret.Labels[secretCreatedByLabelName] == secretCreatedByTestkubeValue
	mayAccess := secret == nil || isControlled || s.config.ListAll

	// When no permissions, make it the same as when it's actually not found, to avoid blind search
	if err == nil && !mayAccess {
		err = k8serrors.NewNotFound(schema.GroupResource{Group: "", Resource: "Secret"}, name)
	}
	if err != nil {
		// Return same not found error for both not found, and not permitted, to avoid blind search
		if k8serrors.IsNotFound(err) {
			return testkube.Secret{}, ErrNotFound
		}
		return testkube.Secret{}, err
	}

	// Disallow when it is not controlled by Testkube
	if !isControlled {
		return testkube.Secret{}, ErrNotControlled
	}

	// Build the secret
	if opts.Labels != nil {
		labels := map[string]string{}
		maps.Copy(labels, opts.Labels)
		labels[secretCreatedByLabelName] = secretCreatedByTestkubeValue
	}

	if data != nil {
		secret.Data = nil
		secret.StringData = data
	}
	if opts.Owner != nil {
		if opts.Owner.UID != "" {
			secret.OwnerReferences = []metav1.OwnerReference{*opts.Owner}
		} else {
			secret.OwnerReferences = nil
		}
	}

	// Create the resource
	secret, err = s.clientSet.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	if err != nil {
		return testkube.Secret{}, err
	}
	return secrets.MapSecretKubeToAPI(secret), nil
}
