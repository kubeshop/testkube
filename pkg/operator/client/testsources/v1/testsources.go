package testsources

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	testsourcev1 "github.com/kubeshop/testkube/api/testsource/v1"
	"github.com/kubeshop/testkube/pkg/operator/secret"
)

const (
	// secretKind is a kind of the secrets
	secretKind = "source-secrets"
	// gitUsernameSecretName is git username secret name
	gitUsernameSecretName = "git-username"
	// gitTokenSecretName is git token secret name
	gitTokenSecretName = "git-token"
)

//go:generate go tool mockgen -source=./testsources.go -destination=./mock_testsources.go -package=testsources "github.com/kubeshop/testkube/pkg/operator/client/testsources/v1" Interface
type Interface interface {
	List(selector string) (*testsourcev1.TestSourceList, error)
	Get(name string) (*testsourcev1.TestSource, error)
	Create(testSource *testsourcev1.TestSource, options ...Option) (*testsourcev1.TestSource, error)
	Update(testSource *testsourcev1.TestSource, options ...Option) (*testsourcev1.TestSource, error)
	Delete(name string) error
	DeleteByLabels(selector string) error
}

// Option contain test source options
type Option struct {
	Secrets map[string]string
}

// NewClient returns new client instance, needs kubernetes client to be passed as dependecy
func NewClient(client client.Client, namespace string) *TestSourcesClient {
	return &TestSourcesClient{
		k8sClient:    client,
		namespace:    namespace,
		secretClient: secret.NewClient(client, namespace, secret.TestkubeTestSourcesSecretLabel),
	}
}

// TestSourcesClient client for getting test sources CRs
type TestSourcesClient struct {
	k8sClient    client.Client
	namespace    string
	secretClient *secret.Client
}

// List shows list of available test sources
func (s TestSourcesClient) List(selector string) (*testsourcev1.TestSourceList, error) {
	list := &testsourcev1.TestSourceList{}
	reqs, err := labels.ParseToRequirements(selector)
	if err != nil {
		return list, err
	}

	options := &client.ListOptions{
		Namespace:     s.namespace,
		LabelSelector: labels.NewSelector().Add(reqs...),
	}

	err = s.k8sClient.List(context.Background(), list, options)
	return list, err
}

// Get gets test source by name in given namespace
func (s TestSourcesClient) Get(name string) (*testsourcev1.TestSource, error) {
	testSource := &testsourcev1.TestSource{}
	err := s.k8sClient.Get(context.Background(), client.ObjectKey{Namespace: s.namespace, Name: name}, testSource)
	return testSource, err
}

// Create creates new test source CRD
func (s TestSourcesClient) Create(testSource *testsourcev1.TestSource, options ...Option) (*testsourcev1.TestSource, error) {
	if len(options) != 0 {
		secrets := make(map[string]string, 0)
		for _, option := range options {
			for key, value := range option.Secrets {
				secrets[key] = value
			}
		}

		secretName := secret.GetMetadataName(testSource.Name, secretKind)
		if len(secrets) != 0 {
			if err := s.secretClient.Create(secretName, testSource.Labels, secrets); err != nil {
				return nil, err
			}

			updateTestSourceSecrets(testSource, secretName, secrets)
		}
	}

	if err := s.k8sClient.Create(context.Background(), testSource); err != nil {
		return nil, err
	}

	return testSource, nil
}

// Update updates test source
func (s TestSourcesClient) Update(testSource *testsourcev1.TestSource, options ...Option) (*testsourcev1.TestSource, error) {
	if len(options) != 0 {
		secrets := make(map[string]string, 0)
		for _, option := range options {
			for key, value := range option.Secrets {
				secrets[key] = value
			}
		}

		secretName := secret.GetMetadataName(testSource.Name, secretKind)
		if len(secrets) != 0 {
			if err := s.secretClient.Apply(secretName, testSource.Labels, secrets); err != nil {
				return nil, err
			}

			updateTestSourceSecrets(testSource, secretName, secrets)
		} else {
			if err := s.secretClient.Delete(secretName); err != nil && !errors.IsNotFound(err) {
				return nil, err
			}

			clearTestSourceSecrets(testSource, secretName)
		}
	}

	if err := s.k8sClient.Update(context.Background(), testSource); err != nil {
		return nil, err
	}

	return testSource, nil
}

// Delete deletes test source by name
func (s TestSourcesClient) Delete(name string) error {
	testSource := &testsourcev1.TestSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: s.namespace,
		},
	}

	secretName := secret.GetMetadataName(testSource.Name, secretKind)
	if err := s.secretClient.Delete(secretName); err != nil && !errors.IsNotFound(err) {
		return err
	}

	err := s.k8sClient.Delete(context.Background(), testSource)
	return err
}

// DeleteByLabels deletes test sources by labels
func (s TestSourcesClient) DeleteByLabels(selector string) error {
	reqs, err := labels.ParseToRequirements(selector)
	if err != nil {
		return err
	}

	if err := s.secretClient.DeleteAll(selector); err != nil {
		return err
	}

	u := &unstructured.Unstructured{}
	u.SetKind("TestSource")
	u.SetAPIVersion("tests.testkube.io/v1")
	err = s.k8sClient.DeleteAllOf(context.Background(), u, client.InNamespace(s.namespace),
		client.MatchingLabelsSelector{Selector: labels.NewSelector().Add(reqs...)})
	return err
}

func updateTestSourceSecrets(testSource *testsourcev1.TestSource, secretName string, secrets map[string]string) {
	if _, ok := secrets[gitUsernameSecretName]; ok {
		if testSource.Spec.Repository != nil && testSource.Spec.Repository.UsernameSecret == nil {
			testSource.Spec.Repository.UsernameSecret = &testsourcev1.SecretRef{
				Name: secretName,
				Key:  gitUsernameSecretName,
			}
		}
	} else {
		if testSource.Spec.Repository != nil && testSource.Spec.Repository.UsernameSecret != nil &&
			testSource.Spec.Repository.UsernameSecret.Name == secretName {
			testSource.Spec.Repository.UsernameSecret = nil
		}
	}

	if _, ok := secrets[gitTokenSecretName]; ok {
		if testSource.Spec.Repository != nil && testSource.Spec.Repository.TokenSecret == nil {
			testSource.Spec.Repository.TokenSecret = &testsourcev1.SecretRef{
				Name: secretName,
				Key:  gitTokenSecretName,
			}
		}
	} else {
		if testSource.Spec.Repository != nil && testSource.Spec.Repository.TokenSecret != nil &&
			testSource.Spec.Repository.TokenSecret.Name == secretName {
			testSource.Spec.Repository.TokenSecret = nil
		}
	}
}

func clearTestSourceSecrets(testSource *testsourcev1.TestSource, secretName string) {
	if testSource.Spec.Repository != nil && testSource.Spec.Repository.UsernameSecret != nil &&
		testSource.Spec.Repository.UsernameSecret.Name == secretName {
		testSource.Spec.Repository.UsernameSecret = nil
	}

	if testSource.Spec.Repository != nil && testSource.Spec.Repository.TokenSecret != nil &&
		testSource.Spec.Repository.TokenSecret.Name == secretName {
		testSource.Spec.Repository.TokenSecret = nil
	}
}
